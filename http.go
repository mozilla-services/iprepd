package iprepd

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// ViolationRequest represents the structure used to apply a violation to a given
// IP address. This structure is used as the basis for unmarshaling requests to
// violation handlers in the API.
type ViolationRequest struct {
	IP        string
	Violation string
}

func mwHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("X-Frame-Options", "DENY")
		w.Header().Add("X-Content-Type-Options", "nosniff")
		w.Header().Add("Content-Security-Policy",
			"default-src 'none'; frame-ancestors 'none'; report-uri /__cspreport__")
		h.ServeHTTP(w, r)
	})
}

func newRouter() *mux.Router {
	r := mux.NewRouter().StrictSlash(true)

	// unauth endpoints
	r.HandleFunc("/__lbheartbeat__", httpHeartbeat).Methods("GET")
	r.HandleFunc("/__heartbeat__", httpHeartbeat).Methods("GET")
	r.HandleFunc("/__version__", httpVersion).Methods("GET")

	// auth endpoints
	r.HandleFunc("/violations", auth(httpGetViolations)).Methods("GET")
	r.HandleFunc("/{ip:(?:[0-9]{1,3}\\.){3}[0-9]{1,3}}", auth(httpGetReputation)).Methods("GET")
	r.HandleFunc("/{ip:(?:[0-9]{1,3}\\.){3}[0-9]{1,3}}", auth(httpPutReputation)).Methods("PUT")
	r.HandleFunc("/{ip:(?:[0-9]{1,3}\\.){3}[0-9]{1,3}}", auth(httpDeleteReputation)).Methods("DELETE")
	r.HandleFunc("/violations/{ip:(?:[0-9]{1,3}\\.){3}[0-9]{1,3}}", auth(httpPutViolation)).Methods("PUT")
	r.HandleFunc("/violations", auth(httpPutViolations)).Methods("PUT")

	return r
}

func startAPI() error {
	return http.ListenAndServe(sruntime.cfg.Listen, mwHandler(newRouter()))
}

func httpVersion(w http.ResponseWriter, r *http.Request) {
	w.Write(sruntime.versionResponse)
}

func httpHeartbeat(w http.ResponseWriter, r *http.Request) {
	_, err := sruntime.redis.Ping().Result()
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

func httpGetReputation(w http.ResponseWriter, r *http.Request) {
	ipstr := mux.Vars(r)["ip"]
	if net.ParseIP(ipstr) == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if isException(ipstr) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	rep, err := repGet(ipstr)
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
	ipstr := mux.Vars(r)["ip"]
	if net.ParseIP(ipstr) == nil {
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
	log.WithFields(log.Fields{
		"ip":         rep.IP,
		"reputation": rep.Reputation,
	}).Info("reputation set")
}

func httpDeleteReputation(w http.ResponseWriter, r *http.Request) {
	ipstr := mux.Vars(r)["ip"]
	if net.ParseIP(ipstr) == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err := repDelete(ipstr)
	if err != nil {
		log.Warnf(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func httpPutViolation(w http.ResponseWriter, r *http.Request) {
	ipstr := mux.Vars(r)["ip"]
	if net.ParseIP(ipstr) == nil {
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
	v.IP = ipstr
	httpPutViolationsInner(w, r, []ViolationRequest{v})
}

func httpPutViolations(w http.ResponseWriter, r *http.Request) {
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
	httpPutViolationsInner(w, r, vs)
}

func httpPutViolationsInner(w http.ResponseWriter, r *http.Request, vs []ViolationRequest) {
	for _, v := range vs {
		rep, err := repGet(v.IP)
		if err == redis.Nil {
			rep = Reputation{IP: v.IP, Reputation: 100}
		} else if err != nil {
			log.Warnf(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		origRep := rep.Reputation
		found, err := rep.applyViolation(v.Violation)
		if err != nil {
			if !found {
				// Don't treat submitting an unknown violation as an error, instead
				// just log it
				log.WithFields(log.Fields{
					"violation": v.Violation,
					"ip":        v.IP,
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
		log.WithFields(log.Fields{
			"violation":           v.Violation,
			"ip":                  rep.IP,
			"reputation":          rep.Reputation,
			"original_reputation": origRep,
		}).Info("violation applied")
	}
}
