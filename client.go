package iprepd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

// Client is the iprepd service client
type Client struct {
	hostURL    string
	authStr    string
	httpClient *http.Client
}

// in an effort to keep the client's error handling consistent and for ease of
// testing, all error messages should be listed below and reused where possible
const (
	// input validation errors
	clientErrURLEmpty            = "url cannot be empty"
	clientErrAuthEmpty           = "auth credentials cannot be empty"
	clientErrObjectEmpty         = "object cannot be empty"
	clientErrObjectTypeEmpty     = "object type cannot be empty"
	clientErrBadType             = "object provided does not match type"
	clientErrViolationEmpty      = "violation cannot be empty"
	clientErrReputationNil       = "reputation cannot be nil"
	clientErrViolationRequestNil = "violation request cannot be nil"
	clientErrMarshal             = "could not marshal payload"
	// http client errors
	clientErrBuildRequest = "could not build http request"
	clientErrSendRequest  = "could not send http request"
	// http reponse payload errors
	clientErrReadResponse = "could not read response body"
	clientErrNon200       = "non 200 status code received"
	clientErrUnmarshal    = "could not unmarshal response body"
)

// NewClient is the default constructor for the client
func NewClient(url, token string, httpClient *http.Client) (*Client, error) {
	if url == "" {
		return nil, errors.New(clientErrURLEmpty)
	}
	if token == "" {
		return nil, errors.New(clientErrAuthEmpty)
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		hostURL:    url,
		authStr:    token,
		httpClient: httpClient,
	}, nil
}

func (c *Client) addAuth(r *http.Request) {
	r.Header.Set("Authorization", c.authStr)
}

// Dump retrieves all reputation entries
func (c *Client) Dump() ([]Reputation, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/dump", c.hostURL), nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", clientErrBuildRequest, err)
	}
	c.addAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", clientErrSendRequest, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %d", clientErrNon200, resp.StatusCode)
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("%s: %s", clientErrReadResponse, err)
	}
	var ret []Reputation
	if err = json.Unmarshal(bodyBytes, &ret); err != nil {
		return nil, fmt.Errorf("%s: %s", clientErrUnmarshal, err)
	}
	return ret, nil
}

// Heartbeat checks whether an IPrepd deployment is healthy / reachable
func (c *Client) Heartbeat() (bool, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/__heartbeat__", c.hostURL), nil)
	if err != nil {
		return false, fmt.Errorf("%s: %s", clientErrBuildRequest, err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("%s: %s", clientErrSendRequest, err)
	}
	return (resp.StatusCode == http.StatusOK), nil
}

// LBHeartbeat checks whether an IPrepd LB is healthy / reachable
func (c *Client) LBHeartbeat() (bool, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/__lbheartbeat__", c.hostURL), nil)
	if err != nil {
		return false, fmt.Errorf("%s: %s", clientErrBuildRequest, err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("%s: %s", clientErrSendRequest, err)
	}
	return (resp.StatusCode == http.StatusOK), nil
}

// GetReputation fetches the reputation of a given object and type
func (c *Client) GetReputation(objectType, object string) (*Reputation, error) {
	if object == "" {
		return nil, errors.New(clientErrObjectEmpty)
	}
	if objectType == "" {
		return nil, errors.New(clientErrObjectTypeEmpty)
	}
	if err := validateType(objectType, object); err != nil {
		return nil, errors.New(clientErrBadType)
	}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/type/%s/%s", c.hostURL, objectType, object), nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", clientErrBuildRequest, err)
	}
	c.addAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", clientErrSendRequest, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %d", clientErrNon200, resp.StatusCode)
	}
	byt, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("%s: %s", clientErrReadResponse, err)
	}
	var r *Reputation
	if err := json.Unmarshal(byt, &r); err != nil {
		return nil, fmt.Errorf("%s: %s", clientErrUnmarshal, err)
	}
	return r, nil
}

// SetReputation updates the reputation of a given object and type to a given score
func (c *Client) SetReputation(r *Reputation) error {
	if r == nil {
		return errors.New(clientErrReputationNil)
	}
	if r.Object == "" {
		return errors.New(clientErrObjectEmpty)
	}
	if r.Type == "" {
		return errors.New(clientErrObjectTypeEmpty)
	}
	if err := validateType(r.Type, r.Object); err != nil {
		return errors.New(clientErrBadType)
	}
	byt, err := json.Marshal(&r)
	if err != nil {
		return fmt.Errorf("%s: %s", clientErrMarshal, err)
	}
	req, err := http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/type/%s/%s", c.hostURL, r.Type, r.Object), bytes.NewBuffer(byt))
	if err != nil {
		return fmt.Errorf("%s: %s", clientErrMarshal, err)
	}
	c.addAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %s", clientErrSendRequest, err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %d", clientErrNon200, resp.StatusCode)
	}
	return nil
}

// DeleteReputation deletes the reputation of a given object and type
func (c *Client) DeleteReputation(objectType, object string) error {
	if object == "" {
		return errors.New(clientErrObjectEmpty)
	}
	if objectType == "" {
		return errors.New(clientErrObjectTypeEmpty)
	}
	if err := validateType(objectType, object); err != nil {
		return errors.New(clientErrBadType)
	}
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/type/%s/%s", c.hostURL, objectType, object), nil)
	if err != nil {
		return fmt.Errorf("%s: %s", clientErrBuildRequest, err)
	}
	c.addAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %s", clientErrSendRequest, err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %d", clientErrNon200, resp.StatusCode)
	}
	return nil
}

// VersionResponse is the response payload from the /__version__ endpoint
type VersionResponse struct {
	Commit  string `json:"commit"`
	Version string `json:"version"`
	Source  string `json:"source"`
	Build   string `json:"build"`
}

// Version retrieves the version of the IPrepd deployment
func (c *Client) Version() (*VersionResponse, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/__version__", c.hostURL), nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", clientErrBuildRequest, err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", clientErrSendRequest, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %d", clientErrNon200, resp.StatusCode)
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("%s: %s", clientErrReadResponse, err)
	}
	var vr *VersionResponse
	if err = json.Unmarshal(bodyBytes, &vr); err != nil {
		return nil, fmt.Errorf("%s: %s", clientErrUnmarshal, err)
	}
	return vr, nil
}

// GetViolations gets all existing violations on the server
func (c *Client) GetViolations() ([]Violation, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/violations", c.hostURL), nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", clientErrBuildRequest, err)
	}
	c.addAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", clientErrSendRequest, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %d", clientErrNon200, resp.StatusCode)
	}
	bodyByt, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", clientErrReadResponse, err)
	}
	var v []Violation
	if err = json.Unmarshal(bodyByt, &v); err != nil {
		return nil, fmt.Errorf("%s: %s", clientErrUnmarshal, err)
	}
	return v, nil
}

// ApplyViolation submits a ViolationRequest to iprepd
func (c *Client) ApplyViolation(vr *ViolationRequest) error {
	if vr == nil {
		return errors.New(clientErrViolationRequestNil)
	}
	if vr.Object == "" {
		return errors.New(clientErrObjectEmpty)
	}
	if vr.Type == "" {
		return errors.New(clientErrObjectTypeEmpty)
	}
	if err := validateType(vr.Type, vr.Object); err != nil {
		return errors.New(clientErrBadType)
	}
	if vr.Violation == "" {
		return errors.New(clientErrViolationEmpty)
	}
	byt, err := json.Marshal(&vr)
	if err != nil {
		return fmt.Errorf("%s: %s", clientErrMarshal, err)
	}
	req, err := http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/violations/type/%s/%s", c.hostURL, vr.Type, vr.Object), bytes.NewBuffer(byt))
	if err != nil {
		return fmt.Errorf("%s: %s", clientErrBuildRequest, err)
	}
	c.addAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %s", clientErrSendRequest, err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %d", clientErrNon200, resp.StatusCode)
	}
	return nil
}

// BatchApplyViolation submits a batch of ViolationRequests to iprepd
func (c *Client) BatchApplyViolation(typ string, vrs []ViolationRequest) error {
	if typ == "" {
		return errors.New(clientErrObjectTypeEmpty)
	}
	if len(vrs) == 0 {
		return nil
	}
	byt, err := json.Marshal(&vrs)
	if err != nil {
		return fmt.Errorf("%s: %s", clientErrMarshal, err)
	}
	req, err := http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/violations/type/%s", c.hostURL, typ), bytes.NewBuffer(byt))
	if err != nil {
		return fmt.Errorf("%s: %s", clientErrMarshal, err)
	}
	c.addAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %s", clientErrSendRequest, err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %d", clientErrNon200, resp.StatusCode)
	}
	return nil
}
