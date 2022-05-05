package app

import (
	"context"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"io/ioutil"
	"sync"
	"temperature/pkg/weather"
	"temperature/pkg/weather/sources"
	"time"
)

type Config struct {
	Token        string             `yaml:"token"`
	Locations    []weather.Location `yaml:"locations,flow"`
	DatabasePath string             `yaml:"db"`
}

type Temperature struct {
	Location *weather.Location
	Value    float32
}

type TemperatureEntity struct {
	Temperature float32
	Department  string `gorm:"primaryKey;autoIncrement:false"`
	Day         int    `gorm:"primaryKey;autoIncrement:false"`
	Month       int    `gorm:"primaryKey;autoIncrement:false"`
	Year        int    `gorm:"primaryKey;autoIncrement:false"`
}

type WeatherApplication struct {
	config     *Config
	weatherAPI *weather.API
	signal     chan<- int
	db         *gorm.DB
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

func New(c *Config) *WeatherApplication {
	app := WeatherApplication{
		config: c,
	}
	app.initWeatherAPI()
	app.initDB()
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
	log.WithFields(log.Fields{"module": "db"}).Info("Init db...")
	db, err := gorm.Open(sqlite.Open(app.config.DatabasePath), &gorm.Config{
		CreateBatchSize: 20,
	})
	if err != nil {
		log.WithFields(log.Fields{"module": "db"}).Fatalf("DB open error: %s", err)
	}
	err = db.AutoMigrate(&TemperatureEntity{})
	if err != nil {
		log.WithFields(log.Fields{"module": "db"}).Fatalf("Migrate db error: %s", err)
	}
	app.db = db
	log.WithFields(log.Fields{"module": "db"}).Info("Db init done")
}

func (app *WeatherApplication) Run() <-chan int {
	log.Info("Run application")
	signal := make(chan int)
	app.signal = signal
	go func() {
		err := app.update()
		if err != nil {
			fmt.Println(err)
			signal <- 2
		}
		signal <- 0
	}()
	return signal
}

func (app *WeatherApplication) update() error {
	log.Info("Update...")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errorChan := make(chan error)
	defer close(errorChan)
	outChan := app.getTemperatures(ctx, errorChan)
	results := make([]*Temperature, 0, 10)

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
	entities := createEntities(results)
	if err := app.db.Create(&entities).Error; err != nil {
		return err
	}
	log.Info("Update done")
	return nil
}

func (app *WeatherApplication) getTemperatures(ctx context.Context, errors chan<- error) chan *Temperature {
	results := make(chan *Temperature)

	wg := sync.WaitGroup{}

	go func() {
		for _, location := range app.config.Locations {
			wg.Add(1)
			go func(location weather.Location) {
				temp, err := app.weatherAPI.TemperatureOfDayByLocation(ctx, &location)
				if err != nil {
					errors <- err
				} else {
					results <- &Temperature{
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

func createEntities(t []*Temperature) []TemperatureEntity {
	entities := make([]TemperatureEntity, 0, len(t))
	for _, temperature := range t {
		date := time.Now().AddDate(0, 0, -1)
		item := TemperatureEntity{
			Temperature: temperature.Value,
			Department:  temperature.Location.Description,
			Day:         date.Day(),
			Month:       int(date.Month()),
			Year:        date.Year(),
		}
		entities = append(entities, item)
	}
	return entities
}
