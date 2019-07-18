package iprepd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// ViolationRequest represents the structure used to apply a violation to a given
// object. This structure is used as the basis for unmarshaling requests to
// violation handlers in the API.
type ViolationRequest struct {
	// The violation name to be applied
	Violation string `json:"violation,omitempty"`

	// The object the violation should be applied to.
	Object string `json:"object,omitempty"`

	// The type of object (e.g., ip).
	Type string `json:"type,omitempty"`

	// An optional recovery suppression value in seconds. If set, it indicates the
	// number of seconds which must pass before the reputation for the object will
	// begin to recover.
	SuppressRecovery int `json:"suppress_recovery,omitempty"`

	// The IP field supports reverse compatibility with older clients. It is essentially
	// the same thing as passing an IP address in the object field, with a type set to
	// ip.
	IP string `json:"ip,omitempty"`
}

// Fixup is used to convert legacy format violations
func (v *ViolationRequest) Fixup(typestr string) {
	// Only apply fixup to ip type requests
	if typestr != "ip" {
		return
	}
	// If the type field is not set, set it to the type specified in the request
	// path
	if v.Type == "" {
		v.Type = typestr
	}
	// If object is not set but the IP field is set, use that as the object
	if v.Object == "" && v.IP != "" {
		v.Object = v.IP
	}
}

// Validate performs validation of a ViolationRequest type
func (v *ViolationRequest) Validate() error {
	if v.Violation == "" {
		return fmt.Errorf("violation request missing required field violation")
	}
	if v.Object == "" {
		return fmt.Errorf("violation request missing required field object")
	}
	if v.Type == "" {
		return fmt.Errorf("violation request missing required field type")
	}
	if v.SuppressRecovery > 1209600 {
		return fmt.Errorf("invalid suppress recovery value %v", v.SuppressRecovery)
	}
	return nil
}

func mwHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s := time.Now()
		defer func() {
			sruntime.statsd.Timing("http.timing", time.Since(s))
		}()
		w.Header().Add("X-Frame-Options", "DENY")
		w.Header().Add("X-Content-Type-Options", "nosniff")
		w.Header().Add("Content-Security-Policy",
			"default-src 'none'; frame-ancestors 'none'; report-uri /__cspreport__")
		w.Header().Add("Strict-Transport-Security", "max-age=31536000")
		h.ServeHTTP(w, r)
	})
}

func newRouter() *mux.Router {
	r := mux.NewRouter().StrictSlash(true)

	// Unauthenticated endpoints
	r.HandleFunc("/__lbheartbeat__", httpHeartbeat).Methods("GET")
	r.HandleFunc("/__heartbeat__", httpHeartbeat).Methods("GET")
	r.HandleFunc("/__version__", httpVersion).Methods("GET")

	// Legacy IP reputation endpoints
	//
	// To maintain compatibility with previous API versions, wrap legacy API
	// calls to add the type field and route to the correct handler
	r.HandleFunc("/{value:(?:[0-9]{1,3}\\.){3}[0-9]{1,3}}",
		auth(wrapLegacyIPRequest(httpGetReputation))).Methods("GET")
	r.HandleFunc("/{value:(?:[0-9]{1,3}\\.){3}[0-9]{1,3}}",
		auth(wrapLegacyIPRequest(httpPutReputation))).Methods("PUT")
	r.HandleFunc("/{value:(?:[0-9]{1,3}\\.){3}[0-9]{1,3}}",
		auth(wrapLegacyIPRequest(httpDeleteReputation))).Methods("DELETE")
	r.HandleFunc("/violations/{value:(?:[0-9]{1,3}\\.){3}[0-9]{1,3}}",
		auth(wrapLegacyIPRequest(httpPutViolation))).Methods("PUT")
	r.HandleFunc("/violations", auth(wrapLegacyIPRequest(httpPutViolations))).Methods("PUT")

	r.HandleFunc("/violations", auth(httpGetViolations)).Methods("GET")
	r.HandleFunc("/dump", auth(httpGetAllReputation)).Methods("GET")
	r.HandleFunc("/type/{type:[a-z]{1,12}}/{value}", auth(httpGetReputation)).Methods("GET")
	r.HandleFunc("/type/{type:[a-z]{1,12}}/{value}", auth(httpPutReputation)).Methods("PUT")
	r.HandleFunc("/type/{type:[a-z]{1,12}}/{value}", auth(httpDeleteReputation)).Methods("DELETE")
	r.HandleFunc("/violations/type/{type:[a-z]{1,12}}/{value}", auth(httpPutViolation)).Methods("PUT")
	r.HandleFunc("/violations/type/{type:[a-z]{1,12}}", auth(httpPutViolations)).Methods("PUT")

	return r
}

func startAPI() error {
	return http.ListenAndServe(sruntime.cfg.Listen, mwHandler(newRouter()))
}

func wrapLegacyIPRequest(rf func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		m := mux.Vars(r)
		m["type"] = "ip"
		mux.SetURLVars(r, m)
		rf(w, r)
	}
}

func hasValidType(r *http.Request) error {
	t := mux.Vars(r)["type"]
	_, ok := validators[t]
	if !ok {
		return fmt.Errorf("type %v is invalid", t)
	}
	return nil
}

func verifyTypeAndValue(r *http.Request) (t string, v string, err error) {
	err = hasValidType(r)
	if err != nil {
		return t, v, err
	}
	t = mux.Vars(r)["type"]
	if t == "" {
		return t, v, fmt.Errorf("type was not set")
	}
	v = mux.Vars(r)["value"]
	if v == "" {
		return t, v, fmt.Errorf("value was not set")
	}
	return t, v, validateType(t, v)
}

func httpVersion(w http.ResponseWriter, r *http.Request) {
	w.Write(sruntime.versionResponse)
}

func httpHeartbeat(w http.ResponseWriter, r *http.Request) {
	_, err := sruntime.redis.ping().Result()
	if err != nil {
		log.Warnf(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func httpGetViolations(w http.ResponseWriter, r *http.Request) {
	buf, err := json.Marshal(sruntime.cfg.Violations)
	if err != nil {
		log.Warnf(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func httpGetAllReputation(w http.ResponseWriter, r *http.Request) {
	allRep, err := repDump()
	if err != nil {
		if err == redis.Nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		log.Warnf(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	buf, err := json.Marshal(allRep)
	if err != nil {
		log.Warnf(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func httpGetReputation(w http.ResponseWriter, r *http.Request) {
	s := time.Now()
	defer func() {
		sruntime.statsd.Timing("http.get_reputation.timing", time.Since(s))
	}()
	typestr, valstr, err := verifyTypeAndValue(r)
	if err != nil {
		log.Warnf(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// If the request is for an IP type object, consult the exception list. Currently
	// exceptions only apply to IP objects.
	if typestr == "ip" {
		if isException(valstr) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}
	rep, err := repGet(typestr, valstr)
	if err != nil {
		if err == redis.Nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		log.Warnf(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	buf, err := json.Marshal(rep)
	if err != nil {
		log.Warnf(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func httpPutReputation(w http.ResponseWriter, r *http.Request) {
	typestr, valstr, err := verifyTypeAndValue(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warnf(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var rep Reputation
	err = json.Unmarshal(buf, &rep)
	if err != nil {
		log.Warnf(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Force object field and type to match value specified in request path
	rep.Object = valstr
	rep.Type = typestr
	err = rep.Validate()
	if err != nil {
		log.Warnf(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err = rep.set()
	if err != nil {
		log.Warnf(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	exc := false
	if rep.Type == "ip" {
		exc = isException(rep.Object)
	}
	log.WithFields(log.Fields{
		"object":     rep.Object,
		"type":       rep.Type,
		"reputation": rep.Reputation,
		"exception":  exc,
	}).Info("reputation set")
}

func httpDeleteReputation(w http.ResponseWriter, r *http.Request) {
	typestr, valstr, err := verifyTypeAndValue(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err = repDelete(typestr, valstr)
	if err != nil {
		log.Warnf(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func httpPutViolation(w http.ResponseWriter, r *http.Request) {
	typestr, valstr, err := verifyTypeAndValue(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warnf(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var v ViolationRequest
	err = json.Unmarshal(buf, &v)
	if err != nil {
		log.Warnf(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Force object field and type to match value specified in request path
	v.Object = valstr
	v.Type = typestr
	httpPutViolationsInner(w, r, typestr, []ViolationRequest{v})
}

func httpPutViolations(w http.ResponseWriter, r *http.Request) {
	// We only have a type to verify here
	err := hasValidType(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	typestr := mux.Vars(r)["type"]
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warnf(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var vs []ViolationRequest
	err = json.Unmarshal(buf, &vs)
	if err != nil {
		log.Warnf(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	httpPutViolationsInner(w, r, typestr, vs)
}

func httpPutViolationsInner(w http.ResponseWriter, r *http.Request, typestr string, vs []ViolationRequest) {
	for _, v := range vs {
		v.Fixup(typestr)
		// Force type field to match value specified in request path
		v.Type = typestr
		err := v.Validate()
		if err != nil {
			log.Warnf(err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		rep, err := repGet(typestr, v.Object)
		if err == redis.Nil {
			rep = Reputation{
				Object:     v.Object,
				Type:       typestr,
				Reputation: 100,
			}
		} else if err != nil {
			log.Warnf(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = rep.Validate()
		if err != nil {
			log.Warnf(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// If recovery suppression was specified add the correct timestamp to the reputation
		// entry. Is suppression is already indicated, only update it if it results in a new
		// timestamp that is beyond what the existing value is.
		if v.SuppressRecovery > 0 {
			nd := time.Now().UTC().Add(time.Second *
				time.Duration(v.SuppressRecovery))
			if rep.DecayAfter.IsZero() || rep.DecayAfter.Before(nd) {
				rep.DecayAfter = nd
			}
		}

		origRep := rep.Reputation
		found, err := rep.applyViolation(v.Violation)
		if err != nil {
			if !found {
				// Don't treat submitting an unknown violation as an error, instead
				// just log it
				log.WithFields(log.Fields{
					"violation": v.Violation,
					"object":    v.Object,
					"type":      v.Type,
				}).Warn("ignoring unknown violation")
				continue
			}
			log.Warnf(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		err = rep.set()
		if err != nil {
			log.Warnf(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		exc := false
		if rep.Type == "ip" {
			exc = isException(rep.Object)
		}
		log.WithFields(log.Fields{
			"violation":           v.Violation,
			"object":              rep.Object,
			"type":                rep.Type,
			"reputation":          rep.Reputation,
			"decay_after":         rep.DecayAfter,
			"original_reputation": origRep,
			"exception":           exc,
		}).Info("violation applied")
	}
}
