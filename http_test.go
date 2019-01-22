package iprepd

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHandlers(t *testing.T) {
	assert.Nil(t, baseTest())
	sruntime.cfg.Auth.DisableAuth = true
	h := mwHandler(newRouter())

	// heartbeat
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/__heartbeat__", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)

	// lb heartbeat
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/__lbheartbeat__", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)

	// request version
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/__version__", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)

	// request violation list
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/violations", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	res := recorder.Result()
	assert.Equal(t, "application/json", res.Header.Get("Content-Type"))
	buf, err := ioutil.ReadAll(res.Body)
	assert.Nil(t, err)
	assert.Equal(t, string(buf),
		"[{\"name\":\"violation1\",\"penalty\":5,\"decreaselimit\":25},"+
			"{\"name\":\"violation2\",\"penalty\":50,\"decreaselimit\":50},"+
			"{\"name\":\"violation3\",\"penalty\":0,\"decreaselimit\":0}]")

	// request reputation for a stored ip
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/192.168.0.1", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	res = recorder.Result()
	assert.Equal(t, "application/json", res.Header.Get("Content-Type"))
	buf, err = ioutil.ReadAll(res.Body)
	assert.Nil(t, err)
	var r Reputation
	err = json.Unmarshal(buf, &r)
	assert.Nil(t, err)
	assert.Equal(t, "192.168.0.1", r.IP)
	assert.Equal(t, 50, r.Reputation)
	assert.Equal(t, false, r.Reviewed)

	// request reputation for an unknown ip
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/192.168.0.2", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusNotFound, recorder.Code)

	// request reputation for invalid ip, should get a 404 as it will not match
	// the handler regex
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/255.2555.255.255", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusNotFound, recorder.Code)

	// store reputation for an ip
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/192.168.2.20", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
	recorder = httptest.NewRecorder()
	buf2 := "{\"ip\": \"192.168.2.20\", \"reputation\": 25}"
	req = httptest.NewRequest("PUT", "/192.168.2.20", bytes.NewReader([]byte(buf2)))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/192.168.2.20", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	res = recorder.Result()
	buf, err = ioutil.ReadAll(res.Body)
	assert.Nil(t, err)
	err = json.Unmarshal(buf, &r)
	assert.Nil(t, err)
	assert.Equal(t, "192.168.2.20", r.IP)
	assert.Equal(t, 25, r.Reputation)
	assert.Equal(t, false, r.Reviewed)

	// try to store invalid reputation score
	recorder = httptest.NewRecorder()
	buf2 = "{\"ip\": \"192.168.2.20\", \"reputation\": 500}"
	req = httptest.NewRequest("PUT", "/192.168.2.20", bytes.NewReader([]byte(buf2)))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	// dump reputation
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/dump", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	res = recorder.Result()
	assert.Equal(t, "application/json", res.Header.Get("Content-Type"))
	buf, err = ioutil.ReadAll(res.Body)
	assert.Nil(t, err)
	var reputations []Reputation
	err = json.Unmarshal(buf, &reputations)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(reputations))
	c := 0
	for _, rep := range reputations {
		if rep.IP == "192.168.2.20" {
			c += 1
			assert.Equal(t, "192.168.2.20", rep.IP)
			assert.Equal(t, 25, rep.Reputation)
		}
		if rep.IP == "192.168.0.1" {
			c += 1
			assert.Equal(t, "192.168.0.1", rep.IP)
			assert.Equal(t, 50, rep.Reputation)
		}
		assert.Equal(t, false, rep.Reviewed)
	}
	assert.Equal(t, 2, c)

	// delete entry
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("DELETE", "/192.168.2.20", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/192.168.2.20", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusNotFound, recorder.Code)

	// put violation
	recorder = httptest.NewRecorder()
	buf2 = "{\"ip\": \"192.168.3.1\", \"violation\": \"violation1\"}"
	req = httptest.NewRequest("PUT", "/violations/192.168.3.1", bytes.NewReader([]byte(buf2)))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/192.168.3.1", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	res = recorder.Result()
	buf, err = ioutil.ReadAll(res.Body)
	assert.Nil(t, err)
	err = json.Unmarshal(buf, &r)
	assert.Nil(t, err)
	assert.Equal(t, "192.168.3.1", r.IP)
	assert.Equal(t, 95, r.Reputation)
	assert.Equal(t, false, r.Reviewed)

	// put violations
	recorder = httptest.NewRecorder()
	buf2 = "[{\"ip\": \"192.168.4.1\", \"violation\": \"violation1\"}," +
		"{\"ip\": \"192.168.5.1\", \"violation\": \"violation2\"}]"
	req = httptest.NewRequest("PUT", "/violations", bytes.NewReader([]byte(buf2)))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/192.168.4.1", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	res = recorder.Result()
	buf, err = ioutil.ReadAll(res.Body)
	assert.Nil(t, err)
	err = json.Unmarshal(buf, &r)
	assert.Nil(t, err)
	assert.Equal(t, "192.168.4.1", r.IP)
	assert.Equal(t, 95, r.Reputation)
	assert.Equal(t, false, r.Reviewed)
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/192.168.5.1", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	res = recorder.Result()
	buf, err = ioutil.ReadAll(res.Body)
	assert.Nil(t, err)
	err = json.Unmarshal(buf, &r)
	assert.Nil(t, err)
	assert.Equal(t, "192.168.5.1", r.IP)
	assert.Equal(t, 50, r.Reputation)
	assert.Equal(t, false, r.Reviewed)
}

func TestViolationDecreaseLimit(t *testing.T) {
	assert.Nil(t, baseTest())
	sruntime.cfg.Auth.DisableAuth = true
	h := mwHandler(newRouter())

	for i := 0; i < 5; i++ {
		recorder := httptest.NewRecorder()
		buf := "{\"ip\": \"192.168.3.1\", \"violation\": \"violation1\"}"
		req := httptest.NewRequest("PUT", "/violations/192.168.3.1", bytes.NewReader([]byte(buf)))
		req.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
	}
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/192.168.3.1", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	res := recorder.Result()
	buf, err := ioutil.ReadAll(res.Body)
	assert.Nil(t, err)
	var r Reputation
	err = json.Unmarshal(buf, &r)
	assert.Nil(t, err)
	assert.Equal(t, "192.168.3.1", r.IP)
	assert.Equal(t, 75, r.Reputation)
	assert.Equal(t, false, r.Reviewed)

	for i := 0; i < 100; i++ {
		recorder = httptest.NewRecorder()
		buf2 := "{\"ip\": \"192.168.3.1\", \"violation\": \"violation1\"}"
		req = httptest.NewRequest("PUT", "/violations/192.168.3.1", bytes.NewReader([]byte(buf2)))
		req.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
	}
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/192.168.3.1", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	res = recorder.Result()
	buf, err = ioutil.ReadAll(res.Body)
	assert.Nil(t, err)
	err = json.Unmarshal(buf, &r)
	assert.Nil(t, err)
	assert.Equal(t, "192.168.3.1", r.IP)
	assert.Equal(t, 25, r.Reputation)
	assert.Equal(t, false, r.Reviewed)

	recorder = httptest.NewRecorder()
	buf2 := "{\"ip\": \"192.168.4.1\", \"violation\": \"violation2\"}"
	req = httptest.NewRequest("PUT", "/violations/192.168.4.1", bytes.NewReader([]byte(buf2)))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/192.168.4.1", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	res = recorder.Result()
	buf, err = ioutil.ReadAll(res.Body)
	assert.Nil(t, err)
	err = json.Unmarshal(buf, &r)
	assert.Nil(t, err)
	assert.Equal(t, "192.168.4.1", r.IP)
	assert.Equal(t, 50, r.Reputation)
	assert.Equal(t, false, r.Reviewed)

	// send the same violation again, which shouldn't change the reputation
	recorder = httptest.NewRecorder()
	buf2 = "{\"ip\": \"192.168.4.1\", \"violation\": \"violation2\"}"
	req = httptest.NewRequest("PUT", "/violations/192.168.4.1", bytes.NewReader([]byte(buf2)))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/192.168.4.1", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	res = recorder.Result()
	buf, err = ioutil.ReadAll(res.Body)
	assert.Nil(t, err)
	err = json.Unmarshal(buf, &r)
	assert.Nil(t, err)
	assert.Equal(t, "192.168.4.1", r.IP)
	assert.Equal(t, 50, r.Reputation)
	assert.Equal(t, false, r.Reviewed)

	for i := 0; i < 5; i++ {
		recorder := httptest.NewRecorder()
		buf := "{\"ip\": \"192.168.5.1\", \"violation\": \"violation1\"}"
		req := httptest.NewRequest("PUT", "/violations/192.168.5.1", bytes.NewReader([]byte(buf)))
		req.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
	}
	recorder = httptest.NewRecorder()
	buf2 = "{\"ip\": \"192.168.5.1\", \"violation\": \"violation2\"}"
	req = httptest.NewRequest("PUT", "/violations/192.168.5.1", bytes.NewReader([]byte(buf2)))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/192.168.5.1", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	res = recorder.Result()
	buf, err = ioutil.ReadAll(res.Body)
	assert.Nil(t, err)
	err = json.Unmarshal(buf, &r)
	assert.Nil(t, err)
	assert.Equal(t, "192.168.5.1", r.IP)
	assert.Equal(t, 50, r.Reputation)
	assert.Equal(t, false, r.Reviewed)
}

func TestExceptions(t *testing.T) {
	assert.Nil(t, baseTest())
	sruntime.cfg.Auth.DisableAuth = true
	h := mwHandler(newRouter())

	recorder := httptest.NewRecorder()
	buf := "{\"ip\": \"192.168.1.1\", \"violation\": \"violation2\"}"
	req := httptest.NewRequest("PUT", "/violations/192.168.1.1", bytes.NewReader([]byte(buf)))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/192.168.1.1", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestDecay(t *testing.T) {
	assert.Nil(t, baseTest())
	sruntime.cfg.Auth.DisableAuth = true
	h := mwHandler(newRouter())

	r := Reputation{
		IP:          "192.168.2.1",
		Reputation:  50,
		LastUpdated: time.Now().Add(-1 * (time.Second * 10)).UTC(),
	}
	buf, err := json.Marshal(r)
	assert.Nil(t, err)
	err = sruntime.redis.set(r.IP, buf, 0).Err()

	// initial request with default (no) decay
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/192.168.2.1", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	res := recorder.Result()
	buf, err = ioutil.ReadAll(res.Body)
	assert.Nil(t, err)
	err = json.Unmarshal(buf, &r)
	assert.Nil(t, err)
	assert.Equal(t, "192.168.2.1", r.IP)
	assert.Equal(t, 50, r.Reputation)
	assert.Equal(t, false, r.Reviewed)

	// adjust the decay and verify it is being applied
	sruntime.cfg.Decay.Points = 1
	sruntime.cfg.Decay.Interval = time.Second
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/192.168.2.1", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	res = recorder.Result()
	buf, err = ioutil.ReadAll(res.Body)
	assert.Nil(t, err)
	err = json.Unmarshal(buf, &r)
	assert.Nil(t, err)
	assert.Equal(t, "192.168.2.1", r.IP)
	assert.InDelta(t, 60, r.Reputation, 2)
	assert.Equal(t, false, r.Reviewed)

	sruntime.cfg.Decay.Points = 50
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/192.168.2.1", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	res = recorder.Result()
	buf, err = ioutil.ReadAll(res.Body)
	assert.Nil(t, err)
	err = json.Unmarshal(buf, &r)
	assert.Nil(t, err)
	assert.Equal(t, "192.168.2.1", r.IP)
	assert.Equal(t, 100, r.Reputation)
	assert.Equal(t, false, r.Reviewed)
}

func TestUnknownViolation(t *testing.T) {
	assert.Nil(t, baseTest())
	sruntime.cfg.Auth.DisableAuth = true
	h := mwHandler(newRouter())

	// Verify that submitting an unknown violation isn't treated as an error, since
	// we just want to log cases where this occurs
	recorder := httptest.NewRecorder()
	buf2 := "[{\"ip\": \"192.168.4.1\", \"violation\": \"unknownviolation\"}," +
		"{\"ip\": \"192.168.5.1\", \"violation\": \"violation2\"}]"
	req := httptest.NewRequest("PUT", "/violations", bytes.NewReader([]byte(buf2)))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/192.168.4.1", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/192.168.5.1", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	res := recorder.Result()
	buf, err := ioutil.ReadAll(res.Body)
	assert.Nil(t, err)
	var r Reputation
	err = json.Unmarshal(buf, &r)
	assert.Nil(t, err)
	assert.Equal(t, "192.168.5.1", r.IP)
	assert.Equal(t, 50, r.Reputation)
	assert.Equal(t, false, r.Reviewed)
}

func TestReviewedReset(t *testing.T) {
	assert.Nil(t, baseTest())
	sruntime.cfg.Auth.DisableAuth = true
	h := mwHandler(newRouter())

	// Verify that the reviewed flag is correctly toggled off if a reputation decays
	// to 100.
	recorder := httptest.NewRecorder()
	buf := "{\"ip\": \"192.168.4.1\", \"violation\": \"violation1\"}"
	req := httptest.NewRequest("PUT", "/violations/192.168.4.1", bytes.NewReader([]byte(buf)))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/192.168.4.1", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	res := recorder.Result()
	buf2, err := ioutil.ReadAll(res.Body)
	assert.Nil(t, err)
	var r Reputation
	err = json.Unmarshal(buf2, &r)
	assert.Nil(t, err)
	assert.Equal(t, "192.168.4.1", r.IP)
	assert.Equal(t, 95, r.Reputation)
	assert.Equal(t, false, r.Reviewed)

	// Toggle the reviewed flag to true
	recorder = httptest.NewRecorder()
	r.Reviewed = true
	buf2, err = json.Marshal(r)
	assert.Nil(t, err)
	req = httptest.NewRequest("PUT", "/192.168.4.1", bytes.NewReader([]byte(buf2)))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/192.168.4.1", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	res = recorder.Result()
	buf2, err = ioutil.ReadAll(res.Body)
	assert.Nil(t, err)
	err = json.Unmarshal(buf2, &r)
	assert.Nil(t, err)
	assert.Equal(t, "192.168.4.1", r.IP)
	assert.Equal(t, 95, r.Reputation)
	assert.Equal(t, true, r.Reviewed)

	// Adjust decay rate to decay the reputation up to 100, and manually readd the value
	// and force the updated time so the decay takes effect
	sruntime.cfg.Decay.Points = 50
	sruntime.cfg.Decay.Interval = time.Second
	r.LastUpdated = time.Now().Add(-1 * (time.Second * 10)).UTC()
	buf2, err = json.Marshal(r)
	assert.Nil(t, err)
	err = sruntime.redis.set(r.IP, buf, 0).Err()
	assert.Nil(t, err)
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/192.168.4.1", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	res = recorder.Result()
	buf2, err = ioutil.ReadAll(res.Body)
	assert.Nil(t, err)
	err = json.Unmarshal(buf2, &r)
	assert.Nil(t, err)
	assert.Equal(t, "192.168.4.1", r.IP)
	assert.Equal(t, 100, r.Reputation)
	assert.Equal(t, false, r.Reviewed)

	// Apply the violation again now that the IP has decayed to 100
	recorder = httptest.NewRecorder()
	buf = "{\"ip\": \"192.168.4.1\", \"violation\": \"violation1\"}"
	req = httptest.NewRequest("PUT", "/violations/192.168.4.1", bytes.NewReader([]byte(buf)))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)

	// Reset the decay rate, and verify the flag is off
	sruntime.cfg.Decay.Points = 0
	sruntime.cfg.Decay.Interval = time.Minute
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/192.168.4.1", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	res = recorder.Result()
	buf2, err = ioutil.ReadAll(res.Body)
	assert.Nil(t, err)
	err = json.Unmarshal(buf2, &r)
	assert.Nil(t, err)
	assert.Equal(t, "192.168.4.1", r.IP)
	assert.Equal(t, 95, r.Reputation)
	assert.Equal(t, false, r.Reviewed)
}

func TestZeroViolation(t *testing.T) {
	assert.Nil(t, baseTest())
	sruntime.cfg.Auth.DisableAuth = true
	h := mwHandler(newRouter())

	recorder := httptest.NewRecorder()
	buf := "{\"ip\": \"192.168.4.1\", \"violation\": \"violation3\"}"
	req := httptest.NewRequest("PUT", "/violations/192.168.4.1", bytes.NewReader([]byte(buf)))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/192.168.4.1", nil)
	h.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	res := recorder.Result()
	buf2, err := ioutil.ReadAll(res.Body)
	assert.Nil(t, err)
	var r Reputation
	err = json.Unmarshal(buf2, &r)
	assert.Nil(t, err)
	assert.Equal(t, "192.168.4.1", r.IP)
	assert.Equal(t, 100, r.Reputation)
	assert.Equal(t, false, r.Reviewed)
}
