package iprepd

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"go.mozilla.org/hawk"
)

func auth(rf func(http.ResponseWriter, *http.Request), needsWrite bool) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if !sruntime.cfg.Auth.DisableAuth {
			hdr := r.Header.Get("Authorization")
			v, wr := false, false
			if strings.HasPrefix(hdr, "Hawk ") {
				v, wr = hawkAuth(r)
			} else if strings.HasPrefix(hdr, "APIKey ") {
				v, wr = apiAuth(r)
			}
			if !v {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if needsWrite && !wr {
				w.WriteHeader(http.StatusForbidden)
				return
			}
		}
		rf(w, r)
	}
}

func apiAuth(r *http.Request) (bool, bool) {
	hdr := r.Header.Get("Authorization")
	hdr = strings.TrimPrefix(hdr, "APIKey ")
	for _, v := range sruntime.cfg.Auth.APIKey {
		if hdr == v {
			return true, true
		}
	}
	for _, v := range sruntime.cfg.Auth.ROAPIKey {
		if hdr == v {
			return true, false
		}
	}
	return false, false
}

func hawkAuth(r *http.Request) (bool, bool) {

	wr := false

	credsLookupFunc := func(creds *hawk.Credentials) error {
		creds.Key = "-"
		creds.Hash = sha256.New
		key, ok := sruntime.cfg.Auth.Hawk[creds.ID]
		if ok {
			wr = true
			creds.Key = key
			return nil
		}
		key, ok = sruntime.cfg.Auth.ROHawk[creds.ID]
		if ok {
			creds.Key = key
			return nil
		}
		return errors.New("unknown hawk id")
	}

	nonceCheckFunc := func(n string, t time.Time, creds *hawk.Credentials) bool { return true }

	auth, err := hawk.NewAuthFromRequest(r, credsLookupFunc, nonceCheckFunc)
	if err != nil {
		log.Warnf(err.Error())
		return false, false
	}

	err = auth.Valid()
	if err != nil {
		log.Warnf(err.Error())
		return false, false
	}

	contentType := r.Header.Get("Content-Type")
	if r.Method != "GET" && r.Method != "DELETE" && contentType == "" {
		log.Warnf("hawk: missing content-type")
		return false, false
	}

	var mediaType string
	if contentType != "" {
		mediaType, _, err = mime.ParseMediaType(contentType)
		if err != nil && contentType != "" {
			log.Warnf(err.Error())
			return false, false
		}

		buf, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Warnf(err.Error())
			return false, false
		}

		r.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
		hash := auth.PayloadHash(mediaType)
		io.Copy(hash, ioutil.NopCloser(bytes.NewBuffer(buf)))
		if !auth.ValidHash(hash) {
			log.Warnf("hawk: invalid payload hash")
			return false, false
		}
	}

	return true, wr
}
