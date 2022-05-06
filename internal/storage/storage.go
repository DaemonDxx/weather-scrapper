package storage

import (
	"temperature/pkg/weather"
	"time"
)

type Storage interface {
	SaveTemperatureByLastDay(temperature []*weather.Temperature) error
	GetAllTemperature() []*TemperatureEntity
}

type DBStorage struct {
	repo TemperatureRepository
}

func NewDBStorage(repo TemperatureRepository) Storage {
	return &DBStorage{repo: repo}
}

func (storage *DBStorage) GetAllTemperature() []*TemperatureEntity {
	return storage.repo.FindAll()
}

func (storage *DBStorage) SaveTemperatureByLastDay(temperature []*weather.Temperature) error {
	entities := createEntities(temperature)
	return storage.repo.SaveAll(entities)
}

func createEntities(t []*weather.Temperature) []*TemperatureEntity {
	entities := make([]*TemperatureEntity, 0, len(t))
	for _, temperature := range t {
		date := time.Now().AddDate(0, 0, -1)
		item := TemperatureEntity{
			Temperature: temperature.Value,
			Department:  temperature.Location.Description,
			Day:         date.Day(),
			Month:       int(date.Month()),
			Year:        date.Year(),
		}
		entities = append(entities, &item)
	}
	return entities
}
