package config

import (
	"os"

	"github.com/pelletier/go-toml/v2"
)

type ProxyConfig struct {
	// Address to listen on
	Match string `json:"match"`
	// Upstream address
	Upstream string `json:"upstream"`
}

type ServiceConfig struct {
	// Name of the service, used for logging and folder name, defaults to the key in the services map
	Name string `json:"name"`
	// Git repository URL
	Repo string `json:"repo"`
	// Command to execute to build the service, relative to the git repository
	Exec string `json:"exec"`
	// Command to execute to build the service, relative to the git repository
	Build string `json:"build"`
	// Restart the service when it exits
	Restart bool `json:"restart"`
	// Maximum number of restarts before giving up
	MaxRestarts int `json:"maxRestarts"`
	// Webhook secret to trigger updates
	Secret string `json:"secret"`
	// Proxy configuration for Caddy
	Proxy ProxyConfig `json:"proxy"`
}

type Config struct {
	// Path to the config file if loaded
	LoadPath string `json:"-"`
	// Services to manage
	Services map[string]ServiceConfig `json:"services"`
	// Address for management API and interface
	Address string `json:"address"`
	// Path to the services folder, where the services are cloned and built
	ServicesPath string `json:"servicesPath"`
	// Secret to verify API requests
	Secret string `json:"secret"`
}

func (c *Config) Load(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	err = toml.NewDecoder(file).Decode(c)
	c.LoadPath = path

	for key, service := range c.Services {
		if service.Name == "" {
			service.Name = key
			c.Services[key] = service
		}
	}

	return err
}

func (c *Config) Save(path string) error {
	if _, err := os.Stat(path); err == nil {
		err := os.Remove(path)
		if err != nil {
			return err
		}
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}

	encoder := toml.NewEncoder(file)
	err = encoder.Encode(c)
	if err != nil {
		return err
	}

	return nil
}
