package iprepd

import (
	"io/ioutil"
	"time"

	"github.com/go-redis/redis"
	log "github.com/sirupsen/logrus"
	"go.mozilla.org/mozlogrus"
	yaml "gopkg.in/yaml.v2"
)

type serverRuntime struct {
	cfg             serverCfg
	redis           *redis.Client
	versionResponse []byte
}

type serverCfg struct {
	Listen string
	Redis  struct {
		Addr string
	}
	Auth struct {
		DisableAuth bool
		Hawk        map[string]string
		APIKey      map[string]string
	}
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
}

func (cfg *serverCfg) validate() error {
	if cfg.VersionResponse == "" {
		cfg.VersionResponse = "./version.json"
	}
	return nil
}

func (cfg *serverCfg) getViolation(v string) *Violation {
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
}

func initRedis(addr string) (ret *redis.Client, err error) {
	ret = redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   0,
	})
	_, err = ret.Ping().Result()
	return
}

func loadCfg(confpath string) (ret serverCfg, err error) {
	buf, err := ioutil.ReadFile(confpath)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(buf, &ret)
	if err != nil {
		return
	}
	return ret, ret.validate()
}

// StartDaemon starts a new instance of iprepd using configuration file confpath.
func StartDaemon(confpath string) {
	log.Infof("starting daemon")

	var err error
	sruntime.cfg, err = loadCfg(confpath)
	if err != nil {
		log.Fatalf(err.Error())
	}
	sruntime.redis, err = initRedis(sruntime.cfg.Redis.Addr)
	if err != nil {
		log.Fatalf(err.Error())
	}
	sruntime.versionResponse, err = ioutil.ReadFile(sruntime.cfg.VersionResponse)
	if err != nil {
		log.Warnf(err.Error())
	}
	go startExceptions()
	err = startAPI()
	if err != nil {
		log.Fatalf(err.Error())
	}
}
