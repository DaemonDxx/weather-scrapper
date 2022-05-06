package main

import (
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/xuri/excelize/v2"
	"strconv"
	"strings"
	"temperature/internal/storage"
	"time"
)

const (
	startRow       = 8
	dateCol        = "A"
	temperatureCol = "B"
)

type Config struct {
	Filepath   *string
	Department *string
	DBPath     *string
}

func main() {
	log.Info("Старт парсинга архивов rp5.ru...")
	config := initApp()

	resultChan := parseFile(config.Filepath)
	success, errors := save(config.DBPath, addDepartmentField(config.Department, resultChan))

	log.WithFields(log.Fields{
		"Всего":     success + errors,
		"Сохранено": success,
		"Ошибок":    errors,
	}).Infof("Файл сохранен в БД")
}

func initApp() *Config {
	parsePath := flag.String("file", "", "Parse file path")
	department := flag.String("department", "", "Department")
	dbPath := flag.String("db", "", "Database path")
	flag.Parse()

	logFields := log.Fields{
		"path":       *parsePath,
		"department": *department,
		"dbPath":     *dbPath,
	}

	if *parsePath == "" || *department == "" || *dbPath == "" {
		log.WithFields(logFields).Fatalf("Указаны не все входные параметры")
	}

	log.WithFields(logFields).Fatalf("Конфигурация")

	return &Config{
		Filepath:   parsePath,
		Department: department,
		DBPath:     dbPath,
	}
}

func parseFile(path *string) <-chan *storage.TemperatureEntity {
	f, err := excelize.OpenFile(*path)
	if err != nil {
		log.Fatalln(err)
	}

	results := make(chan *storage.TemperatureEntity, 40)

	go func() {
		acc := make([]float32, 0, 8)
		var prevDate *time.Time
		row := startRow
		for {
			date, err := getDate(f, row)
			if err != nil {
				break
			}
			temp := getTemp(f, row)

			if prevDate == nil {
				prevDate = date
			}

			if prevDate.Day() != date.Day() {
				entity := storage.TemperatureEntity{
					Temperature: average(acc),
					Day:         prevDate.Day(),
					Month:       int(prevDate.Month()),
					Year:        prevDate.Year(),
				}
				results <- &entity
				acc = clearAcc(acc)
			}

			acc = append(acc, temp)
			prevDate = date
			row++
		}
		close(results)
	}()

	return results
}

func getDate(f *excelize.File, row int) (*time.Time, error) {
	dateCell, err := f.GetCellValue("Архив Погоды rp5", fmt.Sprintf("%s%d", dateCol, row))
	if err != nil {
		log.WithFields(log.Fields{
			"row": row,
		}).Fatalln("Ошибка чтения строки")
	}
	if dateCell == "" {
		return nil, fmt.Errorf("EOF")
	}
	return extractDate(dateCell), nil
}

func getTemp(f *excelize.File, row int) float32 {
	tempCell, err := f.GetCellValue("Архив Погоды rp5", fmt.Sprintf("%s%d", temperatureCol, row))
	if err != nil {
		log.WithFields(log.Fields{
			"row": row,
		}).Fatalln("Ошибка чтения строки")
	}
	temp, _ := strconv.ParseFloat(tempCell, 32)
	return float32(temp)
}

func extractDate(s string) *time.Time {
	s = strings.Fields(s)[0]
	arr := strings.Split(s, ".")
	day, _ := strconv.Atoi(arr[0])
	month, _ := strconv.Atoi(arr[1])
	year, _ := strconv.Atoi(arr[2])
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	return &date
}

func average(items []float32) float32 {
	var sum float32 = 0.0
	for _, i := range items {
		sum += i
	}
	return sum / float32(len(items))
}

func clearAcc(slice []float32) []float32 {
	return slice[:0]
}

func addDepartmentField(department *string, result <-chan *storage.TemperatureEntity) <-chan *storage.TemperatureEntity {
	transformChan := make(chan *storage.TemperatureEntity, cap(result))
	go func() {
		for entity := range result {
			entity.Department = *department
			transformChan <- entity
		}
		close(transformChan)
	}()
	return transformChan
}

func save(path *string, results <-chan *storage.TemperatureEntity) (success int, errors int) {
	repo := storage.New(path)
	err := repo.Init()
	if err != nil {
		log.Fatalf("Cannot open DB: %s", err)
	}

	success = 0
	errors = 0

	for entity := range results {
		err := repo.Save(entity)
		if err != nil {
			log.WithFields(log.Fields{
				"date": fmt.Sprintf("%s.%s.%s", entity.Day, entity.Month, entity.Year),
			}).Errorln("Не удалось сохранить значение")
			errors++
		} else {
			success++
		}
	}

	return success, errors
}
