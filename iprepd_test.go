package iprepd

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/stretchr/testify/assert"
)

const (
	testBuild   = "testing"
	testCommit  = "testcommit"
	testVersion = "testversion"
	testSource  = "https://github.com/mozilla-services/iprepd"
)

type mockStatsClient struct {
	statsd.NoOpClient
	NumInvalid float64
}

func newMockStatsClient() *mockStatsClient {
	return &mockStatsClient{
		NumInvalid: 0.0,
	}
}

func (m *mockStatsClient) Incr(name string, tags []string, rate float64) error {
	if name == ("handler.invalid_url") {
		m.NumInvalid = m.NumInvalid + 1.0
	}
	return nil
}

func baseTest() error {
	_, err := sruntime.redis.flushAll().Result()
	if err != nil {
		return err
	}
	sruntime.cfg.Decay.Points = 0
	sruntime.cfg.Decay.Interval = time.Minute
	vrBytes, err := json.Marshal(&VersionResponse{
		Build:   testBuild,
		Commit:  testCommit,
		Version: testVersion,
		Source:  testSource,
	})
	if err != nil {
		return err
	}
	sruntime.versionResponse = vrBytes
	r := Reputation{
		Object:     "192.168.0.1",
		Type:       TypeIP,
		Reputation: 50,
	}
	err = r.set()
	if err != nil {
		return err
	}
	r = Reputation{
		Object:     "10.0.0.1",
		Type:       TypeIP,
		Reputation: 25,
	}
	err = r.set()
	if err != nil {
		return err
	}
	r = Reputation{
		Object:     "2001:db8:a0b:12f0::1",
		Type:       TypeIP,
		Reputation: 50,
	}
	err = r.set()
	if err != nil {
		return err
	}
	r = Reputation{
		Object:     "usr@mozilla.com",
		Type:       TypeEmail,
		Reputation: 50,
	}
	err = r.set()
	if err != nil {
		return err
	}
	return nil
}

func TestLoadSampleConfig(t *testing.T) {
	_, err := LoadCfg("./iprepd.yaml.sample")
	assert.Nil(t, err)
}

func TestMain(m *testing.M) {
	var (
		err  error
		tcfg ServerCfg
	)
	tcfg.Redis.Addr = "127.0.0.1:6379"
	err = tcfg.validate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	renv := os.Getenv("IPREPD_TEST_REDISADDR")
	if renv != "" {
		tcfg.Redis.Addr = renv
	}
	sruntime.statsd, err = newStatsdClient(tcfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	sruntime.redis, err = newRedisLink(tcfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	sruntime.cfg.Auth.Hawk = map[string]string{"root": "toor", "user": "secret"}
	sruntime.cfg.Auth.APIKey = map[string]string{"u1": "key1", "u2": "key2"}
	sruntime.cfg.Auth.ROHawk = map[string]string{"roroot": "rotoor"}
	sruntime.cfg.Auth.ROAPIKey = map[string]string{"rou1": "rokey1"}
	sruntime.cfg.Exceptions.File = []string{"./testdata/exceptions.txt"}
	sruntime.cfg.Exceptions.AWS = true
	sruntime.cfg.Decay.Points = 0
	sruntime.cfg.Decay.Interval = time.Minute
	sruntime.cfg.Violations = []Violation{
		{"violation1", 5, 25},
		{"violation2", 50, 50},
		{"violation3", 0, 0},
	}
	sruntime.cfg.IP6Prefix = 64
	loadExceptions()
	os.Exit(m.Run())
}
