package weather

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

type Coordinates struct {
	Lon float32 `yaml:"lon"`
	Lat float32 `yaml:"lat"`
}

type Location struct {
	Description string        `yaml:"description"`
	Coordinates []Coordinates `yaml:"coordinates,flow"`
}

type Result struct {
	Temperature float32
	Err         error
}

type Source interface {
	GetWeatherByDate(date *time.Time, coordinate *Coordinates) (float32, error)
}

type API struct {
	source Source
}

func New(s Source) *API {
	weather := API{source: s}
	return &weather
}

func (m *API) TemperatureOfDay(coordinates *Coordinates) (float32, error) {
	date := time.Now().AddDate(0, 0, -1)
	return m.source.GetWeatherByDate(&date, coordinates)
}

func (m *API) TemperatureOfDayByLocation(locations *Location) (float32, error) {
	temps := make([]float32, 0, 5)
	errs := make([]error, 0)
	resultsChan := make(chan *Result, 5)
	defer close(resultsChan)
	wg := sync.WaitGroup{}

	for _, coordinates := range locations.Coordinates {
		wg.Add(1)
		go func(coordinates Coordinates) {
			temp, err := m.TemperatureOfDay(&coordinates)
			if err != nil {
				errs = append(errs, err)
			} else {
				temps = append(temps, temp)
			}
			wg.Done()
		}(coordinates)
	}
	wg.Wait()
	if len(errs) != 0 {
		return 0, errors.New(generateErrorMessage(errs))
	}
	return average(temps), nil
}

func average(items []float32) float32 {
	var sum float32 = 0.0
	for _, i := range items {
		sum += i
	}
	return sum / float32(len(items))
}

func generateErrorMessage(errs []error) string {
	message := strings.Builder{}
	message.WriteString(fmt.Sprintf("При выполнении запроса возникли ошибки (%d): ", len(errs)))
	for _, e := range errs {
		message.WriteString(e.Error())
		message.WriteString("; ")
	}
	return message.String()
}
