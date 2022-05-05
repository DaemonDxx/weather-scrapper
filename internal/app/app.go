package app

import (
	"context"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	"sync"
	"temperature/internal/repository"
	"temperature/pkg/weather"
	"temperature/pkg/weather/sources"
)

type WeatherApplication struct {
	config     *Config
	weatherAPI *weather.API
	signal     chan<- int
	repository repository.TemperatureRepository
	cron       *cron.Cron
}

func New(c *Config) *WeatherApplication {
	app := WeatherApplication{
		config: c,
		cron:   cron.New(),
	}
	app.initWeatherAPI()
	app.initDB()
	app.initCron()
	return &app
}

func (app *WeatherApplication) initWeatherAPI() {
	log.WithFields(log.Fields{"module": "weather"}).Info("Init weather API...")
	source := sources.NewOpenWeatherAPI(&app.config.Token)
	//source := sources.FakeSource{}
	api, err := weather.New(source)
	if err != nil {
		log.WithFields(log.Fields{"module": "weather"}).Fatalf("Weather API init error: %s", err)
	}
	app.weatherAPI = api
	log.WithFields(log.Fields{"module": "weather"}).Info("Weather API init done")
}

func (app *WeatherApplication) initDB() {
	log.WithFields(log.Fields{"module": "repository"}).Info("Init repository...")
	repo := repository.New(&app.config.DatabasePath)
	err := repo.Init()
	if err != nil {
		log.WithFields(log.Fields{"module": "repository"}).Fatalf("Repository init error: %s", err)
	}
	app.repository = repo
	log.WithFields(log.Fields{"module": "repository"}).Info("Repository init done")
}

func (app *WeatherApplication) initCron() {
	log.WithFields(log.Fields{"module": "cron"}).Info("Init cron...")
	_, err := app.cron.AddFunc(app.config.Schedule, func() {
		log.Info("Update...")
		err := app.update()
		if err != nil {
			log.Errorf("Update error: %s", err)
		} else {
			log.Info("Update done")
		}
	})
	if err != nil {
		log.WithFields(log.Fields{"module": "cron"}).Fatalf("Crin init error: %s", err)
	}
	log.WithFields(log.Fields{"module": "cron"}).Info("Cron init done")
}

func (app *WeatherApplication) Run() <-chan int {
	log.Info("Run application")
	signal := make(chan int)
	app.signal = signal
	app.cron.Start()
	return signal
}

func (app *WeatherApplication) update() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errorChan := make(chan error)
	defer close(errorChan)
	outChan := app.getTemperatures(ctx, errorChan)
	results := make([]*weather.Temperature, 0, 10)

EXIT:
	for {
		select {
		case r, ok := <-outChan:
			if !ok {
				break EXIT
			}
			results = append(results, r)
			log.WithFields(log.Fields{
				"department":  r.Location.Description,
				"temperature": r.Value,
			}).Info("Get temperature")
		case err := <-errorChan:
			return err
		}
	}
	if err := app.repository.Save(results); err != nil {
		return err
	}
	return nil
}

func (app *WeatherApplication) getTemperatures(ctx context.Context, errors chan<- error) chan *weather.Temperature {
	results := make(chan *weather.Temperature)

	wg := sync.WaitGroup{}

	go func() {
		for _, location := range app.config.Locations {
			wg.Add(1)
			go func(location weather.Location) {
				temp, err := app.weatherAPI.TemperatureOfDayByLocation(ctx, &location)
				if err != nil {
					errors <- err
				} else {
					results <- &weather.Temperature{
						Location: &location,
						Value:    temp,
					}
				}
				wg.Done()
			}(location)
		}
		wg.Wait()
		close(results)
	}()

	return results
}
