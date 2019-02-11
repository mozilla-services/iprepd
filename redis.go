package iprepd

import (
	"fmt"
	"math/rand"
	"runtime"
	"time"

	"github.com/go-redis/redis"
	log "github.com/sirupsen/logrus"
)

type redisLink struct {
	master      *redis.Client
	readClients []*redis.Client
}

func (r *redisLink) keys(pattern string) *redis.StringSliceCmd {
	return r.master.Keys(pattern)
}

func (r *redisLink) del(k ...string) *redis.IntCmd {
	return r.master.Del(k...)
}

func (r *redisLink) flushAll() *redis.StatusCmd {
	return r.master.FlushAll()
}

func (r *redisLink) get(k string) (ret []byte, err error) {
	p := rand.Perm(len(r.readClients))
	for _, i := range p {
		ret, err = r.readClients[i].Get(k).Bytes()
		if err == nil || (err != nil && err == redis.Nil) {
			return
		}
		log.Error(err.Error())
	}
	// None of the read clients could satisfy the request, return the last
	// error we have seen
	return
}

func (r *redisLink) ping() *redis.StatusCmd {
	return r.master.Ping()
}

func (r *redisLink) set(k string, v interface{}, e time.Duration) *redis.StatusCmd {
	return r.master.Set(k, v, e)
}

func instrumentRedisCmd(old func(cmd redis.Cmder) error) func(cmd redis.Cmder) error {
	return func(cmd redis.Cmder) error {
		s := time.Now()
		err := old(cmd)
		sruntime.statsd.Timing(fmt.Sprintf("redis.%s.timing", cmd.Name()), time.Since(s))
		return err
	}
}

func newRedisLink(cfg serverCfg) (ret redisLink, err error) {
	minIdleConns := cfg.Redis.MinIdleConn
	if cfg.Redis.MaxPoolSize != 0 && cfg.Redis.MaxPoolSize < 20 {
		minIdleConns = cfg.Redis.MaxPoolSize
	} else if cfg.Redis.MaxPoolSize == 0 && (10*runtime.NumCPU()) < 20 {
		minIdleConns = 10 * runtime.NumCPU()
	}

	ret.master = redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.Addr,
		DB:           0,
		ReadTimeout:  time.Millisecond * time.Duration(cfg.Redis.ReadTimeout),
		WriteTimeout: time.Millisecond * time.Duration(cfg.Redis.WriteTimeout),
		DialTimeout:  time.Millisecond * time.Duration(cfg.Redis.DialTimeout),
		PoolSize:     cfg.Redis.MaxPoolSize,
		MinIdleConns: minIdleConns,
	})
	ret.master.WrapProcess(instrumentRedisCmd)
	_, err = ret.ping().Result()
	if err != nil {
		return
	}
	ret.readClients = make([]*redis.Client, 0)
	for _, x := range cfg.Redis.Replicas {
		// We are going to add the master later; if we also see it specified in the replica
		// configuration just skip it for now
		if x == cfg.Redis.Addr {
			continue
		}
		y := redis.NewClient(&redis.Options{
			Addr:         x,
			DB:           0,
			ReadTimeout:  time.Millisecond * time.Duration(cfg.Redis.ReadTimeout),
			DialTimeout:  time.Millisecond * time.Duration(cfg.Redis.DialTimeout),
			PoolSize:     cfg.Redis.MaxPoolSize,
			MinIdleConns: minIdleConns,
		})
		y.WrapProcess(instrumentRedisCmd)
		ret.readClients = append(ret.readClients, y)
	}
	// Also use the master for reads
	ret.readClients = append(ret.readClients, ret.master)
	return
}
