package main

import (
	"fmt"
)

var (
	Version        = "dev"
	CommitHash     = "000000"
	BuildTimestamp = "n/a"
	Builder        = "unkown"
)

func BuildVersion() string {
	return fmt.Sprintf("phpipam2dns %s-%s build on %s using %s", Version, CommitHash, BuildTimestamp, Builder)
}
