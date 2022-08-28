package main

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
)

type NSUpdater struct {
	config Config
}

func (u *NSUpdater) Init(config Config) *NSUpdater {
	u.config = config
	return u
}

func Reverse4(ip string) string {
	spip := strings.Split(ip, ".")
	for i, j := 0, len(spip)-1; i < j; i, j = i+1, j-1 {
		spip[i], spip[j] = spip[j], spip[i]
	}
	return strings.Join(spip, ".") + ".in-addr.arpa."
}

func (u *NSUpdater) ensureReverse(hostname string, ip string, just_clean bool) {
	full_ptr := Reverse4(ip)
	dnssrvcnf, zonename, err := u.GetServerConfigForHostname(full_ptr)
	if err != nil {
		log.Warningln(err)
		return
	}

	c := new(dns.Client)
	m := new(dns.Msg)
	m.SetQuestion(full_ptr, dns.TypePTR)
	m.RecursionDesired = true
	result, _, err := c.Exchange(m, net.JoinHostPort(dnssrvcnf.Server, "53"))
	if err != nil {
		log.Fatal(err)
	}

	if result.Rcode != dns.RcodeSuccess && result.Rcode != dns.RcodeNameError {
		log.Fatalf("query %s failed against server %s with code %s", full_ptr, dnssrvcnf.Server, dns.RcodeToString[result.Rcode])
	}

	hostname_found := false
	var rr_to_cleanup []dns.RR
	if result.Rcode == dns.RcodeSuccess {
		// Check and/or cleanup time
		for _, resrec := range result.Answer {
			if resrec.(*dns.PTR).Ptr != hostname || just_clean {
				log.Debugf("set to remove %v from %v", resrec.(*dns.PTR).Ptr, full_ptr)
				rr_to_cleanup = append(rr_to_cleanup, resrec)
			} else {
				log.Debugf("%v in %v exists", resrec.(*dns.PTR).Ptr, full_ptr)
				hostname_found = true
			}
		}
	}
	rev_msg := new(dns.Msg)
	rev_msg.SetUpdate(zonename) // zone name
	if len(rr_to_cleanup) != 0 {
		rev_msg.RemoveName(rr_to_cleanup)
	}
	if !hostname_found && !just_clean {
		log.Debugf("create %v in %v", hostname, full_ptr)
		rev_msg.Insert([]dns.RR{&dns.PTR{
			Hdr: dns.RR_Header{
				Name:   full_ptr,
				Rrtype: dns.TypePTR,
				Class:  dns.ClassINET,
				Ttl:    60,
			},
			Ptr: hostname,
		}})
	}
	if len(rev_msg.Ns) > 0 {
		rev_msg.SetTsig(dns.Fqdn(dnssrvcnf.KeyName), dns.Fqdn(dnssrvcnf.Algo), 300, time.Now().Unix())
		// perform
		c := new(dns.Client)
		c.TsigSecret = map[string]string{dns.Fqdn(dnssrvcnf.KeyName): dnssrvcnf.Secret}
		rsp, _, err := c.Exchange(rev_msg, dnssrvcnf.Server+":53")
		if err != nil {
			log.Errorln("dns:Client.Exchange ... %v: %v", dnssrvcnf.Server, err)
			return
		}

		if rsp.Rcode == dns.RcodeSuccess {
			log.Debugf("Update ok %+v", rev_msg.Ns)
		} else {
			log.Errorln("error updating rcode=%v for %+v", dns.RcodeToString[rsp.Rcode], rev_msg.Ns)
		}
	} else {
		log.Debugf("%v already ok in %v", hostname, full_ptr)
	}
}

func (u *NSUpdater) ensureStraight(hostname string, ip string, just_clean bool) {
	dnssrvcnf, zonename, err := u.GetServerConfigForHostname(hostname)
	if err != nil {
		log.Warningln(err)
		return
	}

	c := new(dns.Client)
	m := new(dns.Msg)
	m.SetQuestion(hostname, dns.TypeA)
	m.RecursionDesired = true
	result, _, err := c.Exchange(m, net.JoinHostPort(dnssrvcnf.Server, "53"))
	if err != nil {
		log.Fatal(err)
	}

	if result.Rcode != dns.RcodeSuccess && result.Rcode != dns.RcodeNameError {
		log.Fatalf("query %s failed against server %s with code %s", hostname, dnssrvcnf.Server, dns.RcodeToString[result.Rcode])
	}

	hostname_found := false
	var rr_to_cleanup []dns.RR
	if result.Rcode == dns.RcodeSuccess {
		// Check and/or cleanup time
		for _, resrec := range result.Answer {
			if resrec.(*dns.A).A.String() != ip || just_clean {
				log.Debugf("set to remove %v from %v", resrec.(*dns.A).A.String(), hostname)
				rr_to_cleanup = append(rr_to_cleanup, resrec.(*dns.A))
			} else {
				log.Debugf("%v in %v exists", resrec.(*dns.A).A.String(), hostname)
				hostname_found = true
			}
		}
	}
	rev_msg := new(dns.Msg)
	rev_msg.SetUpdate(zonename) // zone name
	if len(rr_to_cleanup) != 0 {
		rev_msg.Remove(rr_to_cleanup)
	}
	if !hostname_found && !just_clean {
		log.Debugf("create %v to %v", hostname, ip)
		rev_msg.Insert([]dns.RR{&dns.A{
			Hdr: dns.RR_Header{
				Name:   hostname,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    60,
			},
			A: net.ParseIP(ip),
		}})
	}
	if len(rev_msg.Ns) > 0 {
		rev_msg.SetTsig(dns.Fqdn(dnssrvcnf.KeyName), dns.Fqdn(dnssrvcnf.Algo), 300, time.Now().Unix())
		// perform
		c := new(dns.Client)
		c.TsigSecret = map[string]string{dns.Fqdn(dnssrvcnf.KeyName): dnssrvcnf.Secret}
		rsp, _, err := c.Exchange(rev_msg, dnssrvcnf.Server+":53")
		if err != nil {
			log.Errorf("dns:Client.Exchange ... %v: %v", dnssrvcnf.Server, err)
			return
		}

		if rsp.Rcode == dns.RcodeSuccess {
			log.Debugf("Update ok %+v", rev_msg.Ns)
		} else {
			log.Errorln("error updating rcode=%v for %+v", dns.RcodeToString[rsp.Rcode], rev_msg.Ns)
		}
	} else {
		log.Debugf("%v already ok in %v", hostname, ip)
	}
}

func (u *NSUpdater) Ensure(hostname string, ip string) {
	hostname = dns.Fqdn(hostname)
	// Reverse
	u.ensureReverse(hostname, ip, false)
	// Straight
	u.ensureStraight(hostname, ip, false)
}

func (u *NSUpdater) Delete(hostname string, ip string) {
	hostname = dns.Fqdn(hostname)
	// Reverse
	u.ensureReverse(hostname, ip, true)
	// Straight
	if hostname != "." {
		u.ensureStraight(hostname, ip, true)
	}
}

func (s *NSUpdater) GetServerConfigForHostname(hostname string) (*ServerDefinition, string, error) {
	var elected_serverdef ServerDefinition
	var elected_zonename string
	matchlen := 0
	for _, definitionholder := range []map[string]ServerDefinition{
		s.config.Forward,
		s.config.Reverse} {
		for zonename, serverdef := range definitionholder {
			if strings.HasSuffix(hostname, "."+zonename) {
				clen := len(zonename)
				if clen > matchlen {
					matchlen = clen
					elected_serverdef = serverdef
					elected_zonename = zonename
				}
			}
		}
	}
	if matchlen == 0 {
		return nil, "", fmt.Errorf("no server found to handle hostname %v", hostname)
	} else {
		return &elected_serverdef, elected_zonename, nil
	}
}
