package iprepd

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func getTestServer(t *testing.T) *httptest.Server {
	assert.Nil(t, baseTest())
	return httptest.NewServer(mwHandler(newRouter()))
}

func getTestClientUnauthorized(ts *httptest.Server) (*Client, error) {
	return NewClient(ts.URL, "APIKey badauth", nil)
}

func getTestClientAuthorized(ts *httptest.Server) (*Client, error) {
	return NewClient(ts.URL, "APIKey key1", nil)
}

func TestNewClient(t *testing.T) {
	goodAuth := "APIKey key1"
	goodURL := "http://127.0.0.1"

	c, err := NewClient("", goodAuth, nil)
	assert.Nil(t, c)
	assert.Equal(t, errors.New(clientErrURLEmpty), err)

	c, err = NewClient(goodURL, "", nil)
	assert.Nil(t, c)
	assert.Equal(t, errors.New(clientErrAuthEmpty), err)

	c, err = NewClient(goodURL, goodAuth, nil)
	assert.Nil(t, err)
	assert.Equal(t, goodURL, c.hostURL)
	assert.Equal(t, goodAuth, c.authStr)
}

func TestDump(t *testing.T) {
	srv := getTestServer(t)
	defer srv.Close()

	c, err := getTestClientAuthorized(srv)
	assert.Nil(t, err)
	reps, err := c.Dump()
	assert.Nil(t, err)
	assert.Equal(t, 4, len(reps))

	c, err = getTestClientUnauthorized(srv)
	assert.Nil(t, err)
	reps, err = c.Dump()
	assert.Equal(t, 0, len(reps))
	assert.Equal(t, fmt.Errorf("%s: %d", clientErrNon200, http.StatusUnauthorized), err)
}

func TestHeartbeat(t *testing.T) {
	srv := getTestServer(t)
	defer srv.Close()

	c, err := getTestClientAuthorized(srv)
	assert.Nil(t, err)
	ok, err := c.Heartbeat()
	assert.Nil(t, err)
	assert.True(t, ok)

	c, err = getTestClientUnauthorized(srv)
	assert.Nil(t, err)
	ok, err = c.Heartbeat()
	assert.Nil(t, err)
	assert.True(t, ok)
}

func TestLBHeartbeat(t *testing.T) {
	srv := getTestServer(t)
	defer srv.Close()

	c, err := getTestClientAuthorized(srv)
	assert.Nil(t, err)
	ok, err := c.LBHeartbeat()
	assert.Nil(t, err)
	assert.True(t, ok)

	c, err = getTestClientUnauthorized(srv)
	assert.Nil(t, err)
	ok, err = c.LBHeartbeat()
	assert.Nil(t, err)
	assert.True(t, ok)
}

func TestGetReputation(t *testing.T) {
	srv := getTestServer(t)
	defer srv.Close()

	goodClient, err := getTestClientAuthorized(srv)
	assert.Nil(t, err)
	badClient, err := getTestClientUnauthorized(srv)
	assert.Nil(t, err)

	tests := []struct {
		Name        string
		Object      string
		ObjectType  string
		ExpectedRep int
		ExpectErr   bool
		ExpectedErr error
		C           *Client
	}{
		// positive tests
		{
			Name:        "test: Good IP",
			Object:      "192.168.0.1",
			ObjectType:  TypeIP,
			ExpectErr:   false,
			ExpectedRep: 50,
			C:           goodClient,
		},
		{
			Name:        "test: Good Email",
			Object:      "usr@mozilla.com",
			ObjectType:  TypeEmail,
			ExpectErr:   false,
			ExpectedRep: 50,
			C:           goodClient,
		},
		// client side validation errors
		{
			Name:        "test: Missing Object",
			ObjectType:  TypeIP,
			ExpectErr:   true,
			ExpectedErr: errors.New(clientErrObjectEmpty),
			C:           goodClient,
		},
		{
			Name:        "test: Missing Type",
			Object:      "192.168.0.1",
			ExpectErr:   true,
			ExpectedErr: errors.New(clientErrObjectTypeEmpty),
			C:           goodClient,
		},
		{
			Name:        "test: Incorrect Object Type",
			Object:      "8.8.8.8",
			ObjectType:  TypeEmail,
			ExpectErr:   true,
			ExpectedErr: errors.New(clientErrBadType),
			C:           goodClient,
		},
		{
			Name:        "test: Bad Object Type",
			Object:      "I am not an IP",
			ObjectType:  TypeIP,
			ExpectErr:   true,
			ExpectedErr: errors.New(clientErrBadType),
			C:           goodClient,
		},
		// server error propagation
		{
			Name:        "test: Non Existent Email",
			Object:      "notinstore@mozilla.com",
			ObjectType:  TypeEmail,
			ExpectErr:   true,
			ExpectedErr: fmt.Errorf("%s: %d", clientErrNon200, http.StatusNotFound),
			C:           goodClient,
		},
		{
			Name:        "test: Non Existent IP",
			Object:      "8.8.8.8",
			ObjectType:  TypeIP,
			ExpectErr:   true,
			ExpectedErr: fmt.Errorf("%s: %d", clientErrNon200, http.StatusNotFound),
			C:           goodClient,
		},
		{
			Name:        "test: Unauthorized",
			Object:      "192.168.0.1",
			ObjectType:  TypeIP,
			ExpectErr:   true,
			ExpectedErr: fmt.Errorf("%s: %d", clientErrNon200, http.StatusUnauthorized),
			C:           badClient,
		},
	}

	for _, tst := range tests {
		rep, err := tst.C.GetReputation(tst.ObjectType, tst.Object)
		if tst.ExpectErr {
			assert.Nil(t, rep, tst.Name)
			assert.Equal(t, tst.ExpectedErr, err, tst.Name)
		} else {
			assert.Nil(t, err, tst.Name)
			assert.Equal(t, tst.Object, rep.Object, tst.Name)
			assert.Equal(t, tst.ObjectType, rep.Type, tst.Name)
			if tst.ObjectType == TypeIP {
				assert.Equal(t, tst.Object, rep.IP, tst.Name)
			}
			assert.Equal(t, rep.Reputation, tst.ExpectedRep)
			assert.Equal(t, false, rep.Reviewed, tst.Name)
			assert.Equal(t, "0001-01-01 00:00:00 +0000 UTC", rep.DecayAfter.String(), tst.Name)
			assert.Equal(t, true, rep.LastUpdated.Before(time.Now()), tst.Name)
		}
	}
}

func TestSetReputation(t *testing.T) {
	srv := getTestServer(t)
	defer srv.Close()

	goodClient, err := getTestClientAuthorized(srv)
	assert.Nil(t, err)
	badClient, err := getTestClientUnauthorized(srv)
	assert.Nil(t, err)

	tests := []struct {
		Name        string
		R           *Reputation
		ExpectErr   bool
		ExpectedErr error
		C           *Client
	}{
		// positive tests
		{
			Name: "test: Good IP",
			R: &Reputation{
				Type:       TypeIP,
				Object:     "208.28.28.28",
				Reputation: 10,
			},
			ExpectErr: false,
			C:         goodClient,
		},
		{
			Name: "test: Good Email",
			R: &Reputation{
				Type:       TypeEmail,
				Object:     "sstallone@mozilla.com",
				Reputation: 45,
			},
			ExpectErr: false,
			C:         goodClient,
		},
		// client side validation errors
		{
			Name:        "test: Nil Reputation",
			ExpectErr:   true,
			ExpectedErr: errors.New(clientErrReputationNil),
			C:           goodClient,
		},
		{
			Name: "test: Reputation No Object",
			R: &Reputation{
				Type:       TypeEmail,
				Reputation: 45,
			},
			ExpectErr:   true,
			ExpectedErr: errors.New(clientErrObjectEmpty),
			C:           goodClient,
		},
		{
			Name: "test: Reputation No Type",
			R: &Reputation{
				Object:     "sstallone@mozilla.com",
				Reputation: 45,
			},
			ExpectErr:   true,
			ExpectedErr: errors.New(clientErrObjectTypeEmpty),
			C:           goodClient,
		},
		{
			Name: "test: Bad IP",
			R: &Reputation{
				Type:       TypeIP,
				Object:     "safhkenfo",
				Reputation: 45,
			},
			ExpectErr:   true,
			ExpectedErr: errors.New(clientErrBadType),
			C:           goodClient,
		},
		{
			Name: "test: Bad Email",
			R: &Reputation{
				Type:       TypeEmail,
				Object:     "safhkenfo",
				Reputation: 45,
			},
			ExpectErr:   true,
			ExpectedErr: errors.New(clientErrBadType),
			C:           goodClient,
		},
		// server error propagation
		{
			Name: "test: Unauthorized",
			R: &Reputation{
				Type:       TypeIP,
				Object:     "208.28.28.28",
				Reputation: 10,
			},
			ExpectErr:   true,
			ExpectedErr: fmt.Errorf("%s: %d", clientErrNon200, http.StatusUnauthorized),
			C:           badClient,
		},
	}

	for _, tst := range tests {
		err := tst.C.SetReputation(tst.R)
		if tst.ExpectErr {
			assert.Equal(t, tst.ExpectedErr, err, tst.Name)
		} else {
			assert.Nil(t, err, tst.Name)
		}
	}
}

func TestDeleteReputation(t *testing.T) {
	srv := getTestServer(t)
	defer srv.Close()

	goodClient, err := getTestClientAuthorized(srv)
	assert.Nil(t, err)
	badClient, err := getTestClientUnauthorized(srv)
	assert.Nil(t, err)

	tests := []struct {
		Name        string
		Object      string
		ObjectType  string
		ExpectErr   bool
		ExpectedErr error
		C           *Client
	}{
		// positive tests
		{
			Name:       "test: Good IP",
			Object:     "192.168.0.1",
			ObjectType: TypeIP,
			ExpectErr:  false,
			C:          goodClient,
		},
		{
			Name:       "test: Good Email",
			Object:     "usr@mozilla.com",
			ObjectType: TypeEmail,
			ExpectErr:  false,
			C:          goodClient,
		},
		{
			Name:       "test: Non Existent Email",
			Object:     "notinstore@mozilla.com",
			ObjectType: TypeEmail,
			ExpectErr:  false,
			C:          goodClient,
		},
		{
			Name:       "test: Non Existent IP",
			Object:     "8.8.8.8",
			ObjectType: TypeIP,
			ExpectErr:  false,
			C:          goodClient,
		},
		// client side validation errors
		{
			Name:        "test: Missing Object",
			ObjectType:  TypeIP,
			ExpectErr:   true,
			ExpectedErr: errors.New(clientErrObjectEmpty),
			C:           goodClient,
		},
		{
			Name:        "test: Missing Type",
			Object:      "192.168.0.1",
			ExpectErr:   true,
			ExpectedErr: errors.New(clientErrObjectTypeEmpty),
			C:           goodClient,
		},
		{
			Name:        "test: Incorrect Object Type",
			Object:      "8.8.8.8",
			ObjectType:  TypeEmail,
			ExpectErr:   true,
			ExpectedErr: errors.New(clientErrBadType),
			C:           goodClient,
		},
		{
			Name:        "test: Bad Object Type",
			Object:      "I am not an IP",
			ObjectType:  TypeIP,
			ExpectErr:   true,
			ExpectedErr: errors.New(clientErrBadType),
			C:           goodClient,
		},
		// server error propagation
		{
			Name:        "test: Unauthorized",
			Object:      "192.168.0.1",
			ObjectType:  TypeIP,
			ExpectErr:   true,
			ExpectedErr: fmt.Errorf("%s: %d", clientErrNon200, http.StatusUnauthorized),
			C:           badClient,
		},
	}

	for _, tst := range tests {
		err := tst.C.DeleteReputation(tst.ObjectType, tst.Object)
		if tst.ExpectErr {
			assert.Equal(t, tst.ExpectedErr, err, tst.Name)
		} else {
			assert.Nil(t, err, tst.Name)
		}
	}
}

func TestVersion(t *testing.T) {
	srv := getTestServer(t)
	defer srv.Close()

	c, err := getTestClientAuthorized(srv)
	assert.Nil(t, err)
	vr, err := c.Version()
	assert.Nil(t, err)
	assert.Equal(t, testVersion, vr.Version)
	assert.Equal(t, testBuild, vr.Build)
	assert.Equal(t, testCommit, vr.Commit)
	assert.Equal(t, testSource, vr.Source)

	c, err = getTestClientUnauthorized(srv)
	assert.Nil(t, err)
	vr, err = c.Version()
	assert.Nil(t, err)
	assert.Equal(t, testVersion, vr.Version)
	assert.Equal(t, testBuild, vr.Build)
	assert.Equal(t, testCommit, vr.Commit)
	assert.Equal(t, testSource, vr.Source)
}

func TestGetViolations(t *testing.T) {
	srv := getTestServer(t)
	defer srv.Close()

	c, err := getTestClientAuthorized(srv)
	assert.Nil(t, err)
	vs, err := c.GetViolations()
	assert.Nil(t, err)
	assert.Equal(t, 3, len(vs))

	c, err = getTestClientUnauthorized(srv)
	assert.Nil(t, err)
	vs, err = c.GetViolations()
	assert.Nil(t, vs)
	assert.Equal(t, fmt.Errorf("%s: %d", clientErrNon200, http.StatusUnauthorized), err)
}

func TestApplyViolation(t *testing.T) {
	srv := getTestServer(t)
	defer srv.Close()

	goodClient, err := getTestClientAuthorized(srv)
	assert.Nil(t, err)
	badClient, err := getTestClientUnauthorized(srv)
	assert.Nil(t, err)

	tests := []struct {
		Name        string
		VR          *ViolationRequest
		ExpectErr   bool
		ExpectedErr error
		C           *Client
	}{
		// positive tests
		{
			Name: "test: Good IP",
			VR: &ViolationRequest{
				Type:      TypeIP,
				Object:    "208.28.28.28",
				Violation: "violation1",
			},
			ExpectErr: false,
			C:         goodClient,
		},
		{
			Name: "test: Good Email",
			VR: &ViolationRequest{
				Type:      TypeEmail,
				Object:    "sstallone@mozilla.com",
				Violation: "violation1",
			},
			ExpectErr: false,
			C:         goodClient,
		},
		// client side validation errors
		{
			Name:        "test: Nil Violation Request",
			ExpectErr:   true,
			ExpectedErr: errors.New(clientErrViolationRequestNil),
			C:           goodClient,
		},
		{
			Name:      "test: Violation Request No Object",
			ExpectErr: true,
			VR: &ViolationRequest{
				Type:      TypeEmail,
				Violation: "violation1",
			},
			ExpectedErr: errors.New(clientErrObjectEmpty),
			C:           goodClient,
		},
		{
			Name:      "test: Violation Request No Object Type",
			ExpectErr: true,
			VR: &ViolationRequest{
				Object:    "sstallone@mozilla.com",
				Violation: "violation1",
			},
			ExpectedErr: errors.New(clientErrObjectTypeEmpty),
			C:           goodClient,
		},
		{
			Name:      "test: Violation Request No Violation",
			ExpectErr: true,
			VR: &ViolationRequest{
				Type:   TypeEmail,
				Object: "sstallone@mozilla.com",
			},
			ExpectedErr: errors.New(clientErrViolationEmpty),
			C:           goodClient,
		},
		{
			Name: "test: Bad IP",
			VR: &ViolationRequest{
				Type:      TypeIP,
				Object:    "I'm not an IP",
				Violation: "violation1",
			},
			ExpectErr:   true,
			ExpectedErr: errors.New(clientErrBadType),
			C:           goodClient,
		},
		{
			Name: "test: Bad Email",
			VR: &ViolationRequest{
				Type:      TypeEmail,
				Object:    "I'm not an email",
				Violation: "violation1",
			},
			ExpectErr:   true,
			ExpectedErr: errors.New(clientErrBadType),
			C:           goodClient,
		},
		// server error propagation
		{
			Name: "test: Unauthorized",
			VR: &ViolationRequest{
				Type:      TypeIP,
				Object:    "208.28.28.28",
				Violation: "violation1",
			},
			ExpectErr:   true,
			ExpectedErr: fmt.Errorf("%s: %d", clientErrNon200, http.StatusUnauthorized),
			C:           badClient,
		},
	}

	for _, tst := range tests {
		err := tst.C.ApplyViolation(tst.VR)
		if tst.ExpectErr {
			assert.Equal(t, tst.ExpectedErr, err, tst.Name)
		} else {
			assert.Nil(t, err, tst.Name)
		}
	}
}

func TestBatchApplyViolations(t *testing.T) {
	srv := getTestServer(t)
	defer srv.Close()

	goodClient, err := getTestClientAuthorized(srv)
	assert.Nil(t, err)
	badClient, err := getTestClientUnauthorized(srv)
	assert.Nil(t, err)

	tests := []struct {
		Name        string
		Type        string
		VRS         []ViolationRequest
		ExpectErr   bool
		ExpectedErr error
		C           *Client
	}{
		//	positive tests
		{
			Name: "test: Good IPs",
			Type: TypeIP,
			VRS: []ViolationRequest{
				{
					Object:    "208.28.28.25",
					Violation: "violation1",
				},
				{
					Object:    "208.28.28.26",
					Violation: "violation1",
				},
				{
					Object:    "208.28.28.28",
					Violation: "violation1",
				},
			},
			ExpectErr: false,
			C:         goodClient,
		},
		{
			Name: "test: Good Emails",
			Type: TypeEmail,
			VRS: []ViolationRequest{
				{
					Object:    "lfine@mozilla.com",
					Violation: "violation1",
				},
				{
					Object:    "choward@mozilla.com",
					Violation: "violation1",
				},
				{
					Object:    "mhoward@mozilla.com",
					Violation: "violation1",
				},
			},
			ExpectErr: false,
			C:         goodClient,
		},
		// client side validation errors
		{
			Name:      "test: Empty Slice",
			Type:      TypeIP,
			VRS:       []ViolationRequest{},
			ExpectErr: false,
			C:         goodClient,
		},
		{
			Name:      "test: Nil Slice",
			Type:      TypeIP,
			ExpectErr: false,
			C:         goodClient,
		},
		{
			Name:        "test: Empty Type",
			ExpectErr:   true,
			ExpectedErr: errors.New(clientErrObjectTypeEmpty),
			C:           goodClient,
		},
		// server error propagation
		{
			Name: "test: Malformed IP in Violation Request",
			Type: TypeIP,
			VRS: []ViolationRequest{
				{
					Object:    "asd8.28.26",
					Violation: "violation1",
				},
				{
					Object:    "208.28.28.26",
					Violation: "violation1",
				},
			},
			ExpectErr:   true,
			ExpectedErr: fmt.Errorf("%s: %d", clientErrNon200, http.StatusBadRequest),
			C:           goodClient,
		},
		{
			Name: "test: Unauthorized",
			Type: TypeIP,
			VRS: []ViolationRequest{
				{
					Type:      TypeIP,
					Object:    "208.28.28.28",
					Violation: "violation1",
				},
			},
			ExpectErr:   true,
			ExpectedErr: fmt.Errorf("%s: %d", clientErrNon200, http.StatusUnauthorized),
			C:           badClient,
		},
	}

	for _, tst := range tests {
		err := tst.C.BatchApplyViolation(tst.Type, tst.VRS)
		if tst.ExpectErr {
			assert.Equal(t, tst.ExpectedErr, err, tst.Name)
		} else {
			assert.Nil(t, err, tst.Name)
		}
	}
}
