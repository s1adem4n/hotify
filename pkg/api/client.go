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

func (c *Client) Services() ([]services.Service, error) {
	resp, err := c.client.Get(fmt.Sprintf("%s/api/services", c.Address))
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
	resp, err := c.client.Get(fmt.Sprintf("%s/api/services/%s", c.Address, name))
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
	resp, err := c.client.Get(fmt.Sprintf("%s/api/services/%s/start", c.Address, name))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *Client) StopService(name string) error {
	resp, err := c.client.Get(fmt.Sprintf("%s/api/services/%s/stop", c.Address, name))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *Client) UpdateService(name string) error {
	resp, err := c.client.Get(fmt.Sprintf("%s/api/services/%s/update", c.Address, name))
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

	return nil
}
