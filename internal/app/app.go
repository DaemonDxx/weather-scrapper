package app

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"sync"
	"temperature/pkg/weather"
	"temperature/pkg/weather/sources"
)

type Config struct {
	Token     string             `yaml:"token"`
	Locations []weather.Location `yaml:"locations,flow"`
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

func Run(config *Config) <-chan int {
	signal := make(chan int)
	go func(config *Config) {
		source := sources.NewOpenWeatherAPI(&config.Token)
		//source := sources.FakeSource{}
		api := weather.New(source)
		wg := sync.WaitGroup{}
		for _, location := range config.Locations {
			wg.Add(1)
			go func(location weather.Location) {
				temp, err := api.TemperatureOfDayByLocation(&location)
				if err != nil {
					fmt.Println(err)
				} else {
					fmt.Printf("Средняя температура в %s = %0.1f \n\r", location.Description, temp)
				}
				wg.Done()
			}(location)
		}
		wg.Wait()
		signal <- 0
		close(signal)
	}(config)
	return signal
}
