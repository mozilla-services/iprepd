package iprepd

import (
	"time"

	"github.com/DataDog/datadog-go/statsd"
)

type statsdClient struct {
	client statsd.ClientInterface
}

func newStatsdClient(cfg ServerCfg) (*statsdClient, error) {
	if cfg.Statsd.Addr == "" {
		return &statsdClient{client: nil}, nil
	}
	c, err := statsd.New(cfg.Statsd.Addr)
	if err != nil {
		return nil, err
	}
	c.Namespace = "iprepd_server."

	return &statsdClient{client: c}, nil
}

func (sc statsdClient) Timing(name string, value time.Duration) error {
	if sc.client == nil {
		return nil
	}
	return sc.client.Timing(name, value, []string{}, 1)
}

func (sc statsdClient) InvalidUrl() error {
	if sc.client == nil {
		return nil
	}
	return sc.client.Incr("handler.invalid_url", []string{}, 1)
}
