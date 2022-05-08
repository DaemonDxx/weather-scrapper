package storage

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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
	Save(t *TemperatureEntity) error
	SaveAll(t []*TemperatureEntity) error
	FindAll() []*TemperatureEntity
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

func (repository *SQLiteRepository) Save(t *TemperatureEntity) error {
	return repository.db.Create(t).Error
}

func (repository *SQLiteRepository) SaveAll(temperature []*TemperatureEntity) error {
	return repository.db.Create(temperature).Error
}

func (repository *SQLiteRepository) FindAll() []*TemperatureEntity {
	var temps []*TemperatureEntity
	repository.db.Order("year, month, day, department").Find(&temps)
	return temps
}
