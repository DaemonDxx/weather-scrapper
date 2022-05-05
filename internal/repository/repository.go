package repository

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"temperature/pkg/weather"
	"time"
)

type TemperatureEntity struct {
	Temperature float32
	Department  string `gorm:"primaryKey;autoIncrement:false"`
	Day         int    `gorm:"primaryKey;autoIncrement:false"`
	Month       int    `gorm:"primaryKey;autoIncrement:false"`
	Year        int    `gorm:"primaryKey;autoIncrement:false"`
}

type TemperatureRepository interface {
	Init() error
	Save(t []*weather.Temperature) error
}

type SQLiteRepository struct {
	db   *gorm.DB
	path string
}

func New(path *string) TemperatureRepository {
	return &SQLiteRepository{
		path: *path,
	}
}

func (repository *SQLiteRepository) Init() error {
	db, err := gorm.Open(sqlite.Open(repository.path), &gorm.Config{
		CreateBatchSize: 20,
	})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&TemperatureEntity{})
	if err != nil {
		return err
	}
	repository.db = db
	return nil
}

func (repository *SQLiteRepository) Save(temperature []*weather.Temperature) error {
	entities := createEntities(temperature)
	return repository.db.Create(&entities).Error
}

func createEntities(t []*weather.Temperature) []TemperatureEntity {
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
