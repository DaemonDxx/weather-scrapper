package scrapper

import (
	"github.com/spf13/viper"
	"temperature/internal/notify"
	"temperature/internal/weather"
)

var ConfigPath = "/etc/scrapper"

type Config struct {
	Locations []weather.Location `yaml:"locations,flow"`
	DBPath    string             `yaml:"dbPath"`
	Schedule  string             `yaml:"schedule"`
	Notifier  NotifierConfig     `yaml:"notifier"`
	Weather   WeatherConfig      `yaml:"weather"`
}

type NotifierConfig struct {
	Telegram *notify.TelegramNotifierConfig `yaml:"telegram"`
}

type WeatherConfig struct {
	Token string `yaml:"token"`
}

func (c *Config) Init() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath(ConfigPath)
	viper.AddConfigPath("./")
	viper.AddConfigPath("./configs")
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	err = viper.Unmarshal(c)
	return err
}
