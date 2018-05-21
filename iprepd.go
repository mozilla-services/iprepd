package iprepd

import (
	"io/ioutil"
	"time"

	"github.com/go-redis/redis"
	"github.com/mozilla-services/yaml"
	log "github.com/sirupsen/logrus"
	"go.mozilla.org/mozlogrus"
)

type serverRuntime struct {
	cfg   serverCfg
	redis *redis.Client
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
	return
}

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
	go startExceptions()
	err = startAPI()
	if err != nil {
		log.Fatalf(err.Error())
	}
}
