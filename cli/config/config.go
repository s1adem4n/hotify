package config

import (
	"os"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Address string
	Secret  string
}

func (c *Config) Load(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	err = toml.NewDecoder(file).Decode(c)
	return err
}

func (c *Config) Save(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}

	err = toml.NewEncoder(file).Encode(c)
	return err
}
