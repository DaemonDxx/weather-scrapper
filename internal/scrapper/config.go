package scrapper

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"temperature/internal/notify"
	"temperature/internal/report"
	"temperature/internal/weather"
)

type NotifierConfig struct {
	Telegram *notify.TelegramNotifierConfig `yaml:"telegram"`
}

type Config struct {
	Token        string             `yaml:"token"`
	Locations    []weather.Location `yaml:"locations,flow"`
	DatabasePath string             `yaml:"db"`
	Schedule     string             `yaml:"schedule"`
	Reporter     report.Config      `yaml:"reporter"`
	Notifier     NotifierConfig     `yaml:"notifier"`
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
