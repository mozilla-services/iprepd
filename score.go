package iprepd

import (
	"encoding/json"
	"fmt"
	"time"
)

// Reputation stores information related to the reputation of a given IP address
type Reputation struct {
	// IP is the IP address associated with the entry
	IP string `json:"ip"`

	// Reputation is the reputation score for the IP address, ranging from 0 to
	// 100 where 100 indicates no violations have been applied to it.
	Reputation int `json:"reputation"`

	// Reviewed is true if the entry has been manually reviewed, this flag indicates
	// a firm confidence in the entry.
	Reviewed bool `json:"reviewed"`

	// LastUpdated indicates when a reputation was last either set manually or via
	// a violation on this entry
	LastUpdated time.Time `json:"lastupdated"`
}

// Validate performs validation and normalization of a Reputation type.
func (r *Reputation) Validate() error {
	// Assume we already have verification of the IP address through the handlers,
	// so just check the reputation score here
	if r.Reputation < 0 || r.Reputation > 100 {
		return fmt.Errorf("invalid reputation score %v", r.Reputation)
	}
	return nil
}

func (r *Reputation) set() error {
	err := r.Validate()
	if err != nil {
		return err
	}
	r.LastUpdated = time.Now().UTC()
	buf, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return sruntime.redis.set(r.IP, buf, time.Hour*24).Err()
}

func (r *Reputation) applyViolation(v string) (found bool, err error) {
	viol := sruntime.cfg.getViolation(v)
	if viol == nil {
		return false, fmt.Errorf("invalid violation: %v", v)
	}
	found = true
	if r.Reputation <= viol.DecreaseLimit {
		return
	}
	if (r.Reputation - viol.Penalty) < viol.DecreaseLimit {
		r.Reputation = viol.DecreaseLimit
	} else {
		r.Reputation -= viol.Penalty
	}
	return
}

func (r *Reputation) applyDecay() error {
	x := sruntime.cfg.Decay.Points *
		int(time.Since(r.LastUpdated)/sruntime.cfg.Decay.Interval)
	if r.Reputation+x > 100 {
		r.Reputation = 100
		r.Reviewed = false
	} else {
		r.Reputation += x
	}
	return nil
}

// Violation describes a violation penalty that can be applied to IP addresses.
type Violation struct {
	// Name of violation as specified in iprepd cfg
	Name string `json:"name"`

	// Penalty is how many points a reputation will be decreased by if this
	// violation is submitted for an IP
	Penalty int `json:"penalty"`

	// DecreaseLimit is the lowest possible value this violation will decrease a
	// reputation to. Since the same violation can be applied multiple times to
	// the same IP, this can be used to place a lower bound on the total decrease.
	DecreaseLimit int `json:"decreaselimit"`
}

func repGet(ipstr string) (ret Reputation, err error) {
	buf, err := sruntime.redis.get(ipstr)
	if err != nil {
		return
	}
	err = json.Unmarshal(buf, &ret)
	if err != nil {
		return
	}
	err = ret.applyDecay()
	return
}

func repDelete(ipstr string) (err error) {
	_, err = sruntime.redis.del(ipstr).Result()
	return
}
