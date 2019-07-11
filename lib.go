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

// NewClient is the default constructor for the client
func NewClient(url, token string, httpClient *http.Client) (*Client, error) {
	if url == "" {
		return nil, errors.New("url cannot be empty")
	}
	if token == "" {
		return nil, errors.New("auth credentials cannot be empty")
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
		return nil, fmt.Errorf("could not build http request: %s", err)
	}
	c.addAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not send http request: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("non 200 status code: %s")
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %s", err)
	}
	var ret []Reputation
	if err = json.Unmarshal(bodyBytes, &ret); err != nil {
		return nil, fmt.Errorf("could not unmarshal response body: %s", err)
	}
	return ret, nil
}

// Heartbeat checks whether an IPrepd deployment is healthy / reachable
func (c *Client) Heartbeat() (bool, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/__heartbeat__", c.hostURL), nil)
	if err != nil {
		return false, fmt.Errorf("could not build http request: %s", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("could not send http request: %s", err)
	}
	return (resp.StatusCode == http.StatusOK), nil
}

// LBHeartbeat checks whether an IPrepd LB is healthy / reachable
func (c *Client) LBHeartbeat() (bool, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/__lbheartbeat__", c.hostURL), nil)
	if err != nil {
		return false, fmt.Errorf("could not build http request: %s", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("could not send http request: %s", err)
	}
	return (resp.StatusCode == http.StatusOK), nil
}

// GetReputation fetches the reputation of a given object and type
func (c *Client) GetReputation(objectType, object string) (*Reputation, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/type/%s/%s", c.hostURL, objectType, object), nil)
	if err != nil {
		return nil, fmt.Errorf("could not build http request: %s", err)
	}
	c.addAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not send http request: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non 200 status code received: %d", resp.StatusCode)
	}
	byt, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %s", err)
	}
	var r *Reputation
	if err := json.Unmarshal(byt, &r); err != nil {
		return nil, fmt.Errorf("could not unmarshal response body: %s", err)
	}
	return r, nil
}

// SetReputation updates the reputation of a given object and type to a given score
func (c *Client) SetReputation(r *Reputation) error {
	byt, err := json.Marshal(&r)
	if err != nil {
		return fmt.Errorf("could not marshal payload: %s", err)
	}
	req, err := http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/type/%s/%s", c.hostURL, r.Type, r.Object), bytes.NewBuffer(byt))
	if err != nil {
		return fmt.Errorf("could not build http request: %s", err)
	}
	c.addAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("could not send http request: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non 200 status code received: %d", resp.StatusCode)
	}
	return nil
}

// DeleteReputation deletes the reputation of a given object and type
func (c *Client) DeleteReputation(objectType, object string) error {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/type/%s/%s", c.hostURL, objectType, object), nil)
	if err != nil {
		return fmt.Errorf("could not build http request: %s", err)
	}
	c.addAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("could not send http request: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non 200 status code received: %d", resp.StatusCode)
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
		return nil, fmt.Errorf("could not build http request: %s", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not send http request: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("non 200 status code: %s")
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %s", err)
	}
	var vr *VersionResponse
	if err = json.Unmarshal(bodyBytes, &vr); err != nil {
		return nil, fmt.Errorf("could not unmarshal response body: %s", err)
	}
	return vr, nil
}

// GetViolations gets all existing violations on the server
func (c *Client) GetViolations() ([]Violation, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/violations", c.hostURL), nil)
	if err != nil {
		return nil, fmt.Errorf("could not build http request: %s", err)
	}
	c.addAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not send http request: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non 200 status code received: %d", resp.StatusCode)
	}
	bodyByt, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %s", err)
	}
	var v []Violation
	if err = json.Unmarshal(bodyByt, &v); err != nil {
		return nil, fmt.Errorf("could not unmarshal response body: %s", err)
	}
	return v, nil
}

// ApplyViolation submits a ViolationRequest to iprepd
func (c *Client) ApplyViolation(vr *ViolationRequest) error {
	byt, err := json.Marshal(&vr)
	if err != nil {
		return fmt.Errorf("could not marshal payload: %s", err)
	}
	req, err := http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/violations/type/%s/%s", c.hostURL, vr.Type, vr.Object), bytes.NewBuffer(byt))
	if err != nil {
		return fmt.Errorf("could not build http request: %s", err)
	}
	c.addAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("could not send http request: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non 200 status code received: %d", resp.StatusCode)
	}
	return nil
}

// BatchApplyViolation submits a batch of ViolationRequests to iprepd
func (c *Client) BatchApplyViolation(typ string, vrs []ViolationRequest) error {
	byt, err := json.Marshal(&vrs)
	if err != nil {
		return fmt.Errorf("could not marshal payload: %s", err)
	}
	req, err := http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/violations/type/%s", c.hostURL, typ), bytes.NewBuffer(byt))
	if err != nil {
		return fmt.Errorf("could not build http request: %s", err)
	}
	c.addAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("could not send http request: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non 200 status code received: %d", resp.StatusCode)
	}
	return nil
}
