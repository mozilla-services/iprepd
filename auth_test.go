package iprepd

import (
	"bytes"
	"crypto/sha256"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mozilla.org/hawk"
)

func TestAuth(t *testing.T) {
	assert.Nil(t, baseTest())
	h := mwHandler(newRouter())

	// auth'd endpoint, should fail
	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, httptest.NewRequest("GET", "/type/ip/192.168.0.1", nil))
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)

	// valid api key
	recorder = httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/type/ip/192.168.0.1", nil)
	req.Header.Set("Authorization", "APIKey key1")
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)

	// invalid api key
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/type/ip/192.168.0.1", nil)
	req.Header.Set("Authorization", "APIKey key1invalid")
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)

	// zero length api key
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/type/ip/192.168.0.1", nil)
	req.Header.Set("Authorization", "APIKey ")
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)

	// valid Read-Only API key for write-not-required endpoint
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/type/ip/192.168.0.1", nil)
	req.Header.Set("Authorization", "APIKey rokey1")
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)

	// valid Read-Only API key for write-required endpoint
	recorder = httptest.NewRecorder()
	buf := "{\"ip\": \"192.168.0.1\", \"reputation\": 50}"
	req = httptest.NewRequest("PUT", "/type/ip/192.168.0.1", bytes.NewReader([]byte(buf)))
	req.Header.Set("Authorization", "APIKey rokey1")
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)

	// valid hawk header
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/type/ip/192.168.0.1", nil)
	auth := hawk.NewRequestAuth(req, &hawk.Credentials{
		ID:   "root",
		Key:  "toor",
		Hash: sha256.New,
	}, 0)
	req.Header.Set("Authorization", auth.RequestHeader())
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)

	// valid Read-Only hawk for a write-not-required endpoint
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/type/ip/192.168.0.1", nil)
	auth = hawk.NewRequestAuth(req, &hawk.Credentials{
		ID:   "roroot",
		Key:  "rotoor",
		Hash: sha256.New,
	}, 0)
	req.Header.Set("Authorization", auth.RequestHeader())
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)

	// valid Read-Only hawk for a write-required endpoint
	recorder = httptest.NewRecorder()
	buf = "{\"ip\": \"192.168.0.1\", \"reputation\": 50}"
	req = httptest.NewRequest("PUT", "/type/ip/192.168.0.1", bytes.NewReader([]byte(buf)))
	auth = hawk.NewRequestAuth(req, &hawk.Credentials{
		ID:   "roroot",
		Key:  "rotoor",
		Hash: sha256.New,
	}, 0)
	hash := auth.PayloadHash("application/json")
	hash.Write([]byte(buf))
	auth.SetHash(hash)
	req.Header.Set("Authorization", auth.RequestHeader())
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)

	// invalid hawk id
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/type/ip/192.168.0.1", nil)
	auth = hawk.NewRequestAuth(req, &hawk.Credentials{
		ID:   "invalid",
		Key:  "toor",
		Hash: sha256.New,
	}, 0)
	req.Header.Set("Authorization", auth.RequestHeader())
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)

	// invalid hawk secret
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/type/ip/192.168.0.1", nil)
	auth = hawk.NewRequestAuth(req, &hawk.Credentials{
		ID:   "root",
		Key:  "invalid",
		Hash: sha256.New,
	}, 0)
	req.Header.Set("Authorization", auth.RequestHeader())
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)

	// valid hawk credentials with a content-type set but no request body on GET,
	// verify the request is rejected
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/type/ip/192.168.0.1", nil)
	auth = hawk.NewRequestAuth(req, &hawk.Credentials{
		ID:   "root",
		Key:  "toor",
		Hash: sha256.New,
	}, 0)
	req.Header.Set("Authorization", auth.RequestHeader())
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)

	// valid hawk put with a request body
	recorder = httptest.NewRecorder()
	buf = "{\"ip\": \"192.168.0.1\", \"reputation\": 50}"
	req = httptest.NewRequest("PUT", "/type/ip/192.168.0.1", bytes.NewReader([]byte(buf)))
	auth = hawk.NewRequestAuth(req, &hawk.Credentials{
		ID:   "root",
		Key:  "toor",
		Hash: sha256.New,
	}, 0)
	hash = auth.PayloadHash("application/json")
	hash.Write([]byte(buf))
	auth.SetHash(hash)
	req.Header.Set("Authorization", auth.RequestHeader())
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)

	// valid hawk creds in a put with a request body, but missing a content-type
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("PUT", "/type/ip/192.168.0.1", bytes.NewReader([]byte(buf)))
	auth = hawk.NewRequestAuth(req, &hawk.Credentials{
		ID:   "root",
		Key:  "toor",
		Hash: sha256.New,
	}, 0)
	hash = auth.PayloadHash("application/json")
	hash.Write([]byte(buf))
	auth.SetHash(hash)
	req.Header.Set("Authorization", auth.RequestHeader())
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)

	// valid hawk creds in a put with a request body, missing a payload hash
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("PUT", "/type/ip/192.168.0.1", bytes.NewReader([]byte(buf)))
	auth = hawk.NewRequestAuth(req, &hawk.Credentials{
		ID:   "root",
		Key:  "toor",
		Hash: sha256.New,
	}, 0)
	req.Header.Set("Authorization", auth.RequestHeader())
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)

	// invalid hawk header
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/type/ip/192.168.0.1", nil)
	req.Header.Set("Authorization", "Hawk invalid")
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)

	// zero length hawk header
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/type/ip/192.168.0.1", nil)
	req.Header.Set("Authorization", "Hawk ")
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)

	// empty authorization header
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/type/ip/192.168.0.1", nil)
	req.Header.Set("Authorization", " ")
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)

	// unauth'd endpoint, should succeed
	recorder = httptest.NewRecorder()
	h.ServeHTTP(recorder, httptest.NewRequest("GET", "/__heartbeat__", nil))
	assert.Equal(t, http.StatusOK, recorder.Code)
}
