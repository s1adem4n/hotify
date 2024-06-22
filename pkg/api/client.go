package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"hotify/pkg/config"
	"hotify/pkg/services"
	"io"
	"net/http"
)

func ResponseOK(resp *http.Response) bool {
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

type Client struct {
	Address string
	Secret  string
	client  *http.Client
}

// custom round tripper for computing the HMAC signature
type roundTripper struct {
	secret string
	rt     http.RoundTripper
}

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body == nil {
		req.Body = io.NopCloser(bytes.NewReader([]byte{}))
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	req.Body = io.NopCloser(bytes.NewReader(body))

	signature := hmac.New(sha256.New, []byte(rt.secret))
	signature.Write(body)

	req.Header.Set("X-Signature-256", fmt.Sprintf("sha256=%x", signature.Sum(nil)))

	return rt.rt.RoundTrip(req)
}

func NewClient(address, secret string) *Client {
	client := &http.Client{
		Transport: &roundTripper{
			secret: secret,
			rt:     http.DefaultTransport,
		},
	}

	return &Client{
		Address: address,
		Secret:  secret,
		client:  client,
	}
}

func (c *Client) Fetch(method string, path string) (*http.Response, error) {
	req, err := http.NewRequest(method, fmt.Sprintf("%s/%s", c.Address, path), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if !ResponseOK(resp) {
		var body []byte
		if resp.Body != nil {
			defer resp.Body.Close()
			body, _ = io.ReadAll(resp.Body)
		}
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, body)
	}

	return resp, nil
}

func (c *Client) GetConfig() (*config.Config, error) {
	resp, err := c.Fetch(http.MethodGet, "api/config")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var config config.Config
	err = json.NewDecoder(resp.Body).Decode(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *Client) Services() ([]services.Service, error) {
	resp, err := c.Fetch(http.MethodGet, "api/services")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var services []services.Service
	err = json.NewDecoder(resp.Body).Decode(&services)
	if err != nil {
		return nil, err
	}

	return services, nil
}

func (c *Client) Service(name string) (*services.Service, error) {
	resp, err := c.Fetch(http.MethodGet, fmt.Sprintf("api/services/%s", name))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var service services.Service
	err = json.NewDecoder(resp.Body).Decode(&service)
	if err != nil {
		return nil, err
	}

	return &service, nil
}

func (c *Client) StartService(name string) error {
	resp, err := c.Fetch(http.MethodGet, fmt.Sprintf("api/services/%s/start", name))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *Client) StopService(name string) error {
	resp, err := c.Fetch(http.MethodGet, fmt.Sprintf("api/services/%s/stop", name))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *Client) UpdateService(name string) error {
	resp, err := c.Fetch(http.MethodGet, fmt.Sprintf("api/services/%s/update", name))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *Client) CreateService(config *config.ServiceConfig) error {
	marshaled, err := json.Marshal(config)
	if err != nil {
		return err
	}

	resp, err := c.client.Post(fmt.Sprintf("%s/api/services", c.Address), "application/json", bytes.NewReader(marshaled))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if !ResponseOK(resp) {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) DeleteService(name string) error {
	resp, err := c.Fetch(http.MethodDelete, fmt.Sprintf("api/services/%s", name))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *Client) RestartService(name string) error {
	resp, err := c.Fetch(http.MethodGet, fmt.Sprintf("api/services/%s/restart", name))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
