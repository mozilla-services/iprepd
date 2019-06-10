package iprepd

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/zmap/go-iptree/iptree"
)

var activeTree *iptree.IPTree
var treeLock sync.Mutex
var isExceptionUpdate = false

type awsIPRanges struct {
	Prefixes []struct {
		IPPrefix string `json:"ip_prefix"`
	} `json:"prefixes"`
}

const awsIPRangeURL = "https://ip-ranges.amazonaws.com/ip-ranges.json"

func startExceptions() {
	for {
		loadExceptions()

		// If this was the first exception load, send a note to the main thread
		// to indicate the API can begin processing requests
		if !isExceptionUpdate {
			sruntime.exceptionsLoaded <- true
			isExceptionUpdate = true
		}

		time.Sleep(time.Hour)
	}
}

func loadExceptions() {
	log.Info("starting exception refresh")
	t := iptree.New()

	for _, x := range sruntime.cfg.Exceptions.File {
		log.Infof("loading file exceptions from %v", x)
		fd, err := os.Open(x)
		if err != nil {
			log.Fatal(err.Error())
		}
		scn := bufio.NewScanner(fd)
		for scn.Scan() {
			_, n, err := net.ParseCIDR(scn.Text())
			if err != nil {
				log.Fatal(err.Error())
			}
			t.Add(n, 0)
		}
		if err = scn.Err(); err != nil {
			log.Fatal(err.Error())
		}
	}

	if sruntime.cfg.Exceptions.AWS {
		log.Infof("loading AWS exceptions from %v", awsIPRangeURL)
		resp, err := http.Get(awsIPRangeURL)
		if err != nil {
			log.Fatal(err.Error())
		}
		defer resp.Body.Close()
		buf, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err.Error())
		}
		var awsp awsIPRanges
		err = json.Unmarshal(buf, &awsp)
		if err != nil {
			log.Fatal(err.Error())
		}
		for _, v := range awsp.Prefixes {
			_, n, err := net.ParseCIDR(v.IPPrefix)
			if err != nil {
				log.Fatal(err.Error())
			}
			t.Add(n, 0)
		}
	}

	treeLock.Lock()
	activeTree = t
	treeLock.Unlock()

	log.Info("completed exception refresh")
}

func isException(ipstr string) bool {
	treeLock.Lock()
	_, f, err := activeTree.GetByString(ipstr)
	treeLock.Unlock()
	if err != nil {
		log.Fatalf(err.Error())
	}
	return f
}
