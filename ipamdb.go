package main

import (
	"database/sql"
	"regexp"
	"time"

	log "github.com/sirupsen/logrus"
)

type ChangelogRecord struct {
	cid      int
	ctype    string
	coid     int
	caction  string
	cdate    string
	hostname string
	ip       string
	cdiff    string
}

type IpamDB struct {
	db       *sql.DB
	maxlogid int
}

func (i *IpamDB) Close() {
	if i != nil {
		i.db.Close()
	}
}

func (i *IpamDB) Open(dsn string) *IpamDB {
	if dsn == "" {
		panic("No DSN configured")
	}

	db, err := sql.Open("mysql", dsn)
	i.db = db
	if err != nil {
		panic(err)
	}
	log.Info("Connected to database")

	db.SetConnMaxIdleTime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	return i
}

var changelogHostnameFinder = regexp.MustCompile(`\[hostname\]. (.+)\r`)
var changelogIPFinder = regexp.MustCompile(`\[ip_addr\]. (.*?)\r`)

func (i *IpamDB) ProcessChangelogRecords(resumeFrom int, clrprocessor func(ChangelogRecord) bool) {
	i.db.QueryRow("SELECT MAX(cid) FROM changelog").Scan(&i.maxlogid)
	log.Debug("max id found in DB: ", i.maxlogid, "\tmax id previous run: ", resumeFrom)

	rows, err := i.db.Query(`
	SELECT cid, ctype, coid, caction, cdate, COALESCE(hostname, ""), COALESCE(INET_NTOA(ip_addr), INET_NTOA(ip_addr), "") AS ip, cdiff
	  FROM changelog 
	  LEFT JOIN ipaddresses ON ctype = 'ip_addr' AND ipaddresses.id = coid
	  WHERE 1
		AND ( hostname IS NOT NULL OR caction = "delete" )
		AND cid > ? AND cid <= ?
		AND cresult = 'success'
		AND ctype = 'ip_addr'
		AND ( cdiff LIKE '%\n[hostname]_ %' OR cdiff LIKE '[hostname]_ %' )
		`, resumeFrom, i.maxlogid)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		clrecord := ChangelogRecord{}
		if err := rows.Scan(&clrecord.cid, &clrecord.ctype, &clrecord.coid, &clrecord.caction, &clrecord.cdate, &clrecord.hostname, &clrecord.ip, &clrecord.cdiff); err != nil {
			panic(err)
		}

		if clrecord.ip == "" {
			matches := changelogIPFinder.FindAllStringSubmatch(clrecord.cdiff, -1)
			clrecord.ip = matches[0][1]
		}
		if clrecord.hostname == "" {
			matches := changelogHostnameFinder.FindAllStringSubmatch(clrecord.cdiff, -1)
			if matches[0][1] != "/ " {
				clrecord.hostname = matches[0][1]
			}
		}
		clrprocessor(clrecord)
	}
}
