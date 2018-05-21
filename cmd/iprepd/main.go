package main

import (
	"flag"

	"github.com/mozilla-services/iprepd"
)

func main() {
	confpath := flag.String("c", "./iprepd.yaml", "path to configuration")
	flag.Parse()
	iprepd.StartDaemon(*confpath)
}
