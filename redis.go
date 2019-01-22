package iprepd

import (
	"math/rand"
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

func newRedisLink(cfg serverCfg) (ret redisLink, err error) {
	ret.master = redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.Addr,
		DB:           0,
		ReadTimeout:  time.Millisecond * time.Duration(cfg.Redis.ReadTimeout),
		WriteTimeout: time.Millisecond * time.Duration(cfg.Redis.WriteTimeout),
		DialTimeout:  time.Millisecond * time.Duration(cfg.Redis.DialTimeout),
	})
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
			Addr:        x,
			DB:          0,
			ReadTimeout: time.Millisecond * time.Duration(cfg.Redis.ReadTimeout),
			DialTimeout: time.Millisecond * time.Duration(cfg.Redis.DialTimeout),
		})
		ret.readClients = append(ret.readClients, y)
	}
	// Also use the master for reads
	ret.readClients = append(ret.readClients, ret.master)
	return
}
