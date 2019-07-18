package iprepd

import (
	"encoding/json"
	"fmt"
	"time"
)

// Reputation stores information related to the reputation of a given object
type Reputation struct {
	// Object is the object associated with the reputation entry. For example
	// if the type is "ip", object will be an IP address.
	Object string `json:"object"`

	// Type describes the type of object the reputation entry is for
	Type string `json:"type"`

	// IP is a legacy field that is associated with reputation requests for IP
	// addresses, and is intended to maintain reverse compatibility.
	//
	// For responses from the API for IP address objects, Object and IP will
	// be set to the same value. For reputation update requests for IP type
	// objects, either can be used but Object will take precedence.
	IP string `json:"ip,omitempty"`

	// Reputation is the reputation score for the object, ranging from 0 to
	// 100 where 100 indicates no violations have been applied to it.
	Reputation int `json:"reputation"`

	// Reviewed is true if the entry has been manually reviewed, this flag indicates
	// a firm confidence in the entry.
	Reviewed bool `json:"reviewed"`

	// LastUpdated indicates when a reputation was last either set manually or via
	// a violation on this entry
	LastUpdated time.Time `json:"lastupdated"`

	// DecayAfter is used to temporarily stop reputation recovery until after the
	// current time has passed the time indicated by DecayAfter. This can be used
	// to for example enforce a mandatory minimum reputation decrease for an object
	// for a set period of time.
	DecayAfter time.Time `json:"decayafter,omitempty"`
}

// Validate performs validation  of a Reputation type.
func (r *Reputation) Validate() error {
	if r.Object == "" {
		return fmt.Errorf("reputation entry missing required field object")
	}
	if r.Type == "" {
		return fmt.Errorf("reputation entry missing required field type")
	}
	if r.Type != "ip" && r.IP != "" {
		return fmt.Errorf("ip field set and type is not ip")
	}
	if r.Reputation < 0 || r.Reputation > 100 {
		return fmt.Errorf("invalid reputation score %v", r.Reputation)
	}
	return nil
}

func keyFromTypeAndValue(typestr string, valstr string) (string, error) {
	if typestr == "" || valstr == "" {
		return "", fmt.Errorf("type or value was not set")
	}
	if typestr == "ip" {
		return valstr, nil
	}
	return typestr + " " + valstr, nil
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
	key, err := keyFromTypeAndValue(r.Type, r.Object)
	if err != nil {
		return err
	}
	return sruntime.redis.set(key, buf, time.Hour*336).Err()
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
	// If DecayAfter is set and we haven't past the indicated timestamp yet
	// don't do anything with the current reputation value.
	//
	// If the value is set and we have passed the indicated point in time, replace
	// the value with the zero value.
	if !r.DecayAfter.IsZero() {
		if r.DecayAfter.After(time.Now().UTC()) {
			return nil
		}
		r.DecayAfter = time.Time{}
	}

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

// Violation describes a violation penalty that can be applied to an object.
type Violation struct {
	// Name of violation as specified in iprepd cfg
	Name string `json:"name"`

	// Penalty is how many points a reputation will be decreased by if this
	// violation is submitted for an object
	Penalty int `json:"penalty"`

	// DecreaseLimit is the lowest possible value this violation will decrease a
	// reputation to. Since the same violation can be applied multiple times to
	// the same object, this can be used to place a lower bound on the total decrease.
	DecreaseLimit int `json:"decreaselimit"`
}

func repGet(typestr string, valstr string) (ret Reputation, err error) {
	var key string
	key, err = keyFromTypeAndValue(typestr, valstr)
	if err != nil {
		return
	}
	buf, err := sruntime.redis.get(key)
	if err != nil {
		return
	}

	err = json.Unmarshal(buf, &ret)
	if err != nil {
		return
	}

	// Apply some compatibility fixups here for IP type requests
	if typestr == "ip" {
		if ret.Object == "" && ret.IP != "" {
			// If we have an IP field set but object is unset, set the object field
			// to IP as this is likely a legacy entry.
			ret.Object = ret.IP
		} else {
			// Otherwise, just set the IP field to the value of the object field
			// to maintain compatibility with older clients
			ret.IP = ret.Object
		}
	}

	// If the type field is unset in the stored entry, set it to the type that was
	// used to make the request
	if ret.Type == "" {
		ret.Type = typestr
	}

	err = ret.applyDecay()
	return
}

func repDelete(typestr string, valstr string) (err error) {
	key, err := keyFromTypeAndValue(typestr, valstr)
	if err != nil {
		return err
	}
	_, err = sruntime.redis.del(key).Result()
	return
}

func repDump() (ret []Reputation, err error) {
	keys, err := sruntime.redis.keys("*").Result()
	if err != nil {
		return
	}

	// Collect and return all entries from the database; note that this is a raw dump
	// and no compatibility fixups or any validation occurs on the returned entries.
	for _, obj := range keys {
		buf, err := sruntime.redis.get(obj)
		if err != nil {
			return ret, err
		}
		reputation := Reputation{}
		err = json.Unmarshal(buf, &reputation)
		if err != nil {
			return ret, err
		}
		ret = append(ret, reputation)
	}

	return
}
