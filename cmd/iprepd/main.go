package main

import (
	"flag"

	"go.mozilla.org/iprepd"
)

func main() {
	confpath := flag.String("c", "./iprepd.yaml", "path to configuration")
	flag.Parse()
	iprepd.StartDaemon(*confpath)
}
