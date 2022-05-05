package app

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"temperature/pkg/weather"
)

type Config struct {
	Token        string             `yaml:"token"`
	Locations    []weather.Location `yaml:"locations,flow"`
	DatabasePath string             `yaml:"db"`
	Schedule     string             `yaml:"schedule"`
}

func (c *Config) LoadFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(data, c)
	if err != nil {
		return err
	}

	return nil
}
