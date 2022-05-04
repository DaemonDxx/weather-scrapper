package app

import (
	"context"
	"fmt"
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
	source := sources.NewOpenWeatherAPI(&app.config.Token)
	//source := sources.FakeSource{}
	api, err := weather.New(source)
	if err != nil {
		panic(err)
	}
	app.weatherAPI = api
}

func (app *WeatherApplication) initDB() {
	db, err := gorm.Open(sqlite.Open(app.config.DatabasePath), &gorm.Config{
		CreateBatchSize: 20,
	})
	if err != nil {
		panic(err)
	}
	err = db.AutoMigrate(&TemperatureEntity{})
	if err != nil {
		panic(err)
	}
	app.db = db
}

func (app *WeatherApplication) Run() <-chan int {
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	outChan, errorsChan := app.getTemperatures(ctx)
	results := make([]*Temperature, 0, 10)

EXIT:
	for {
		select {
		case r, ok := <-outChan:
			if !ok {
				break EXIT
			}
			results = append(results, r)
			fmt.Printf("Средняя температруа в %s равно %0.1f \n\r", r.Location.Description, r.Value)
		case err := <-errorsChan:
			return err
		}
	}
	entities := createEntities(results)
	if err := app.db.Create(&entities).Error; err != nil {
		return err
	}
	return nil
}

func (app *WeatherApplication) getTemperatures(ctx context.Context) (chan *Temperature, chan error) {
	results := make(chan *Temperature, len(app.config.Locations))
	errors := make(chan error, len(app.config.Locations))

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
		close(errors)
	}()

	return results, errors
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
