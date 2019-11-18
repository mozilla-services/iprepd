package main

import (
	"flag"

	"go.mozilla.org/repd"
)

func main() {
	confpath := flag.String("c", "./repd.yaml", "path to configuration")
	flag.Parse()
	repd.StartDaemon(*confpath)
}
