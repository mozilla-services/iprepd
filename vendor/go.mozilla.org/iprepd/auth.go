package iprepd

import (
	"bytes"
	"crypto/sha256"
	"errors"
	log "github.com/sirupsen/logrus"
	"go.mozilla.org/hawk"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"strings"
	"time"
)

func auth(rf func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if !sruntime.cfg.Auth.DisableAuth {
			hdr := r.Header.Get("Authorization")
			v := false
			if strings.HasPrefix(hdr, "Hawk ") {
				v = hawkAuth(r)
			} else if strings.HasPrefix(hdr, "APIKey ") {
				v = apiAuth(r)
			}
			if !v {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		}
		rf(w, r)
	}
}

func apiAuth(r *http.Request) bool {
	hdr := r.Header.Get("Authorization")
	hdr = strings.TrimPrefix(hdr, "APIKey ")
	for _, v := range sruntime.cfg.Auth.APIKey {
		if hdr == v {
			return true
		}
	}
	return false
}

func hawkLookupCreds(creds *hawk.Credentials) error {
	creds.Key = "-"
	creds.Hash = sha256.New
	c, ok := sruntime.cfg.Auth.Hawk[creds.ID]
	if !ok {
		return errors.New("unknown hawk id")
	}
	creds.Key = c
	return nil
}

func hawkAuth(r *http.Request) bool {
	auth, err := hawk.NewAuthFromRequest(r, hawkLookupCreds,
		func(n string, t time.Time, creds *hawk.Credentials) bool { return true })
	if err != nil {
		log.Warnf(err.Error())
		return false
	}

	err = auth.Valid()
	if err != nil {
		log.Warnf(err.Error())
		return false
	}

	contentType := r.Header.Get("Content-Type")
	if r.Method != "GET" && r.Method != "DELETE" && contentType == "" {
		log.Warnf("hawk: missing content-type")
		return false
	}

	var mediaType string
	if contentType != "" {
		mediaType, _, err = mime.ParseMediaType(contentType)
		if err != nil && contentType != "" {
			log.Warnf(err.Error())
			return false
		}

		buf, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Warnf(err.Error())
			return false
		}

		r.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
		hash := auth.PayloadHash(mediaType)
		io.Copy(hash, ioutil.NopCloser(bytes.NewBuffer(buf)))
		if !auth.ValidHash(hash) {
			log.Warnf("hawk: invalid payload hash")
			return false
		}
	}

	return true
}
