package iprepd

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"go.mozilla.org/mozlogrus"
	yaml "gopkg.in/yaml.v2"
)

type serverRuntime struct {
	cfg              ServerCfg
	redis            redisLink
	versionResponse  []byte
	exceptionsLoaded chan bool
	statsd           *statsdClient
}

type ServerCfg struct {
	Listen string
	Redis  struct {
		Addr         string
		Replicas     []string
		ReadTimeout  int
		WriteTimeout int
		DialTimeout  int
		MaxPoolSize  int
		MinIdleConn  int
	}
	Auth struct {
		DisableAuth bool
		Hawk        map[string]string
		APIKey      map[string]string
		ROHawk      map[string]string
		ROAPIKey    map[string]string
	}
	IP6Prefix  int
	Violations []Violation
	Decay      struct {
		Points   int
		Interval time.Duration
	}
	Exceptions struct {
		File []string
		AWS  bool
	}
	VersionResponse string
	Statsd          struct {
		Addr string
	}
	Sync struct {
		MaxLimit          int
		MinimumReputation int
		DeleteFile        bool
		GCS               struct {
			Filename   string
			Bucketname string
		}
	}
}

func (cfg *ServerCfg) validate() error {
	if cfg.VersionResponse == "" {
		cfg.VersionResponse = "./version.json"
	}
	if cfg.Redis.ReadTimeout == 0 {
		cfg.Redis.ReadTimeout = 100
	}
	if cfg.Redis.WriteTimeout == 0 {
		cfg.Redis.WriteTimeout = 100
	}
	if cfg.Redis.DialTimeout == 0 {
		cfg.Redis.DialTimeout = 250
	}
	if cfg.Redis.MinIdleConn == 0 {
		cfg.Redis.MinIdleConn = 20
	}
	if cfg.IP6Prefix == 0 {
		cfg.IP6Prefix = 64
	}
	return nil
}

func (cfg *ServerCfg) getViolation(v string) *Violation {
	for _, x := range cfg.Violations {
		if x.Name == v {
			return &x
		}
	}
	return nil
}

var sruntime serverRuntime

func init() {
	mozlogrus.Enable("iprepd")
	rand.Seed(time.Now().Unix())
}

func LoadCfg(confpath string) (ret ServerCfg, err error) {
	buf, err := ioutil.ReadFile(confpath)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(buf, &ret)
	if err != nil {
		return
	}
	// prefer STATSD_HOST env var over config file (#13)
	statsdHost := os.Getenv("STATSD_HOST")
	if statsdHost != "" {
		statsdPort := os.Getenv("STATSD_PORT")
		if statsdPort == "" {
			statsdPort = "8125"
		}
		ret.Statsd.Addr = fmt.Sprintf("%s:%s", statsdHost, statsdPort)
	}
	return ret, ret.validate()
}

func CreateServerRuntime(confpath string) {
	var err error
	sruntime.exceptionsLoaded = make(chan bool, 1)
	sruntime.cfg, err = LoadCfg(confpath)
	if err != nil {
		log.Fatalf(err.Error())
	}
	sruntime.statsd, err = newStatsdClient(sruntime.cfg)
	if err != nil {
		log.Fatalf(err.Error())
	}
	sruntime.redis, err = newRedisLink(sruntime.cfg)
	if err != nil {
		log.Fatalf(err.Error())
	}
	sruntime.versionResponse, err = ioutil.ReadFile(sruntime.cfg.VersionResponse)
	if err != nil {
		log.Warnf(err.Error())
	}
}

// StartDaemon starts a new instance of iprepd using configuration file confpath.
func StartDaemon(confpath string) {
	log.Infof("starting daemon")

	CreateServerRuntime(confpath)

	go startExceptions()
	select {
	case <-sruntime.exceptionsLoaded:
		log.Infof("initial exception load completed, starting API")
	case <-time.After(5 * time.Second):
		log.Fatalf("initial exception load timed out")
	}
	err := startAPI()
	if err != nil {
		log.Fatalf(err.Error())
	}
}
