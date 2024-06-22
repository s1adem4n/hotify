package caddy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"strings"
)

type Upstream struct {
	Dial string `json:"dial"`
}

type Handle struct {
	Handler   string     `json:"handler"`
	Upstreams []Upstream `json:"upstreams"`
}

type Match struct {
	Host []string `json:"host"`
}

type Route struct {
	ID     string   `json:"@id"`
	Handle []Handle `json:"handle"`
	Match  []Match  `json:"match"`
}

func NewProxy(id string, match string, upstream string) Route {
	return Route{
		ID: id,
		Handle: []Handle{
			{
				Handler: "reverse_proxy",
				Upstreams: []Upstream{
					{
						Dial: upstream,
					},
				},
			},
		},
		Match: []Match{
			{
				Host: []string{match},
			},
		},
	}
}

type Server struct {
	Listen []string `json:"listen"`
	Routes []Route  `json:"routes"`
}

type Client struct {
	ServerName string
	Address    string
	Server     Server
}

var BaseConfig = `{
	"apps": {
		"http": {
			"servers": {}
		}
	}
}`

func (c *Client) LoadBaseConfig() error {
	resp, err := http.Get(fmt.Sprintf("%s/config/apps/http/servers", c.Address))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		resp, err = http.Post(fmt.Sprintf("%s/config/", c.Address), "application/json", bytes.NewBuffer([]byte(BaseConfig)))
		if err != nil {
			return err
		}
		defer resp.Body.Close()
	}

	return nil
}

func (c *Client) LoadServer() error {
	path := fmt.Sprintf("config/apps/http/servers/%s", c.ServerName)

	resp, err := http.Get(fmt.Sprintf("%s/%s", c.Address, path))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if strings.HasPrefix(string(body), "null") || resp.StatusCode != http.StatusOK {
		server := Server{
			Listen: []string{":443"},
			Routes: []Route{},
		}

		err := c.SetObject("POST", path, server)
		if err != nil {
			return err
		}

		c.Server = server
	} else {
		err := json.Unmarshal(body, &c.Server)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) Init() error {
	err := c.LoadBaseConfig()
	if err != nil {
		return err
	}
	err = c.LoadServer()
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) ObjectExists(path string) bool {
	resp, err := http.Get(fmt.Sprintf("%s/%s", c.Address, path))
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	return string(body) != "null\n" && resp.StatusCode == http.StatusOK
}

func (c *Client) DeleteObject(path string) error {
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/%s", c.Address, path), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *Client) SetObject(method string, path string, object any) error {
	marshaled, err := json.Marshal(object)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(method, fmt.Sprintf("%s/%s", c.Address, path), bytes.NewBuffer(marshaled))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *Client) AddRoute(route Route) error {
	if c.ObjectExists(fmt.Sprintf("id/%s", route.ID)) {
		return errors.New("route already exists")
	}

	path := fmt.Sprintf("config/apps/http/servers/%s/routes", c.ServerName)

	err := c.SetObject("POST", path, route)
	if err != nil {
		return err
	}

	c.Server.Routes = append(c.Server.Routes, route)
	return nil
}

func GenerateID(match string) string {
	h := fnv.New64a()
	h.Write([]byte(match))
	return fmt.Sprintf("%x", h.Sum64())
}
