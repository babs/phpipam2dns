package main

import (
	"flag"
	"os"
	"path"
	"runtime"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
)

func main() {
	initlog()

	config := new(Config)
	config.Load("config.yml")
	// fmt.Println(serverconfig)

	nsu := NSUpdater{}
	nsu.Init(*config)

	state := NewState("state.json")

	skipState := flag.Bool("skip-state", false, "Skip state")
	onlyOnce := flag.Bool("once", false, "run once")
	flag.Parse()

	if *skipState {
		state.MaxId = 0
	}

	ipamdb := IpamDB{}
	ipamdb.Open(config.DSN)
	defer ipamdb.Close()
	for {
		ipamdb.ProcessChangelogRecords(state.MaxId, func(record ChangelogRecord) bool {
			log.Debugf(
				"changeid=%v  action=%v  object[type=%v id=%v] => hostname:%v  ip:%v\n",
				record.cid,
				record.caction,
				record.ctype,
				record.coid,
				record.hostname,
				record.ip,
			)

			if record.caction == "delete" {
				nsu.Delete(record.hostname, record.ip)
			}

			if record.caction == "add" || record.caction == "edit" {
				nsu.Ensure(record.hostname, record.ip)
			}

			return true
		})
		state.SetMaxId(ipamdb.maxlogid).Save()
		if *onlyOnce {
			break
		}
		time.Sleep(1 * time.Second)
	}
}

func initlog() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
		CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
			fileName := path.Base(frame.File) + ":" + strconv.Itoa(frame.Line)
			//return frame.Function, fileName
			return "", fileName
		},
	})
	log.SetReportCaller(true)

	lvl, ok := os.LookupEnv("LOG_LEVEL")
	if !ok || lvl == "" {
		lvl = "info"
	}
	ll, err := log.ParseLevel(lvl)
	if err != nil {
		ll = log.DebugLevel
	}
	log.SetLevel(ll)
}
