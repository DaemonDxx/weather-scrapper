package scrapper

import (
	"context"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	"sync"
	"temperature/internal/storage"
	"temperature/pkg/weather"
	"temperature/pkg/weather/sources"
)

type Scrapper struct {
	config     *Config
	weatherAPI *weather.API
	storage    storage.Storage
	cron       *cron.Cron
}

func New(c *Config) *Scrapper {
	app := Scrapper{
		config: c,
		cron:   cron.New(),
	}
	app.initWeatherAPI()
	app.initStorage()
	app.initCron()
	return &app
}

func (scrapper *Scrapper) initWeatherAPI() {
	log.WithFields(log.Fields{"module": "weather"}).Info("Init weather API...")
	source := sources.NewOpenWeatherAPI(&scrapper.config.Token)
	//source := sources.FakeSource{}
	api, err := weather.New(source)
	if err != nil {
		log.WithFields(log.Fields{"module": "weather"}).Fatalf("Weather API init error: %s", err)
	}
	scrapper.weatherAPI = api
	log.WithFields(log.Fields{"module": "weather"}).Info("Weather API init done")
}

func (scrapper *Scrapper) initStorage() {
	log.WithFields(log.Fields{"module": "storage"}).Info("Storage storage...")
	repo := storage.New(&scrapper.config.DatabasePath)
	err := repo.Init()
	if err != nil {
		log.WithFields(log.Fields{"module": "storage"}).Fatalf("Repository init error: %s", err)
	}
	storage := storage.NewDBStorage(repo)
	scrapper.storage = storage
	log.WithFields(log.Fields{"module": "storage"}).Info("Storage init done")
}

func (scrapper *Scrapper) initCron() {
	log.WithFields(log.Fields{"module": "cron"}).Info("Init cron...")
	_, err := scrapper.cron.AddFunc(scrapper.config.Schedule, func() {
		log.Info("Update...")
		err := scrapper.update()
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

func (scrapper *Scrapper) Run() {
	log.Info("Run application")
	scrapper.cron.Start()
}

func (scrapper *Scrapper) update() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errorChan := make(chan error)
	defer close(errorChan)
	outChan := scrapper.getTemperatures(ctx, errorChan)
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
	if err := scrapper.storage.SaveTemperatureByLastDay(results); err != nil {
		return err
	}
	return nil
}

func (scrapper *Scrapper) getTemperatures(ctx context.Context, errors chan<- error) chan *weather.Temperature {
	results := make(chan *weather.Temperature)

	wg := sync.WaitGroup{}

	go func() {
		for _, location := range scrapper.config.Locations {
			wg.Add(1)
			go func(location weather.Location) {
				temp, err := scrapper.weatherAPI.TemperatureOfDayByLocation(ctx, &location)
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
