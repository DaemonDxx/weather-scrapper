package weather

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Coordinates struct {
	Lon float32 `yaml:"lon"`
	Lat float32 `yaml:"lat"`
}

type Source interface {
	Init() error
	GetWeatherByDate(ctx context.Context, date *time.Time, coordinate *Coordinates) (float32, error)
}

type Location struct {
	Description string        `yaml:"description"`
	Coordinates []Coordinates `yaml:"coordinates,flow"`
}

type Temperature struct {
	Location *Location
	Value    float32
}

type Result struct {
	Temperature float32
	Err         error
}

type API struct {
	source Source
}

func New(s Source) (*API, error) {
	weather := API{source: s}
	err := weather.checkConnection()
	return &weather, err
}

func (api *API) checkConnection() error {
	_, err := api.TemperatureOfDay(context.Background(), &Coordinates{
		Lon: 0,
		Lat: 0,
	})
	if err != nil {
		return fmt.Errorf("Не удалось установить соединенение с OpenWeather.com по прочине: %s", err.Error())
	}
	return nil
}

func (api *API) TemperatureOfDay(ctx context.Context, coordinates *Coordinates) (float32, error) {
	date := time.Now().AddDate(0, 0, -1)
	return api.source.GetWeatherByDate(ctx, &date, coordinates)
}

func (api *API) TemperatureOfDayByLocation(ctx context.Context, locations *Location) (float32, error) {
	temps := make([]float32, 0, 5)

	resultsChan := make(chan float32, len(locations.Coordinates))
	errorsChan := make(chan error)

	defer close(errorsChan)

	wg := sync.WaitGroup{}
	apiCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		for _, coordinates := range locations.Coordinates {
			wg.Add(1)
			go func(coordinates Coordinates) {
				temp, err := api.TemperatureOfDay(apiCtx, &coordinates)
				if err != nil {
					errorsChan <- err
				} else {
					resultsChan <- temp
				}
				wg.Done()
			}(coordinates)
		}
		wg.Wait()
		close(resultsChan)
	}()
	for {
		select {
		case result, ok := <-resultsChan:
			if !ok {
				return average(temps), nil
			}
			temps = append(temps, result)
		case err := <-errorsChan:
			return 0, err
		case <-ctx.Done():
			return 0, nil
		}
	}
}

func average(items []float32) float32 {
	var sum float32 = 0.0
	for _, i := range items {
		sum += i
	}
	return sum / float32(len(items))
}
