package report

import (
	"github.com/xuri/excelize/v2"
	"io"
	"os"
	"path"
	"temperature/internal/storage"
	"time"
)

const (
	indexColl     = 1
	departmentCol = 2
	dateCol       = 3
	valueCol      = 4
	filename      = "report.xlsx"
)

type Config struct {
	TempDir      string `yaml:"tempDir"`
	TemplatePath string `yaml:"template"`
}

type Report struct {
	Filename string
	Data     io.Reader
}

type Reporter interface {
	Get() (*Report, error)
}

type ExcelReporter struct {
	storage storage.Storage
	config  *Config
}

func NewExcelReporter(storage storage.Storage, config *Config) Reporter {
	return &ExcelReporter{
		storage: storage,
		config:  config,
	}
}

func (reporter *ExcelReporter) Get() (*Report, error) {
	//defer reporter.deleteTempFile()
	template, err := excelize.OpenFile(reporter.config.TemplatePath)
	if err != nil {
		return nil, err
	}
	records := reporter.storage.GetAllTemperature()
	fillTemplate(template, records)

	path := reporter.getFilePath(filename)

	err = template.SaveAs(path)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return &Report{
		Filename: filename,
		Data:     f,
	}, err
}

func (reporter *ExcelReporter) deleteTempFile() {
	os.Remove(reporter.getFilePath(filename))
}

func (reporter *ExcelReporter) getFilePath(filename string) string {
	return path.Join(reporter.config.TempDir, filename)
}

func fillTemplate(template *excelize.File, items []*storage.TemperatureEntity) *excelize.File {
	for i, record := range items {
		indexCoord, _ := excelize.CoordinatesToCellName(indexColl, i+2)
		departmentCoord, _ := excelize.CoordinatesToCellName(departmentCol, i+2)
		dateCoord, _ := excelize.CoordinatesToCellName(dateCol, i+2)
		valueCoord, _ := excelize.CoordinatesToCellName(valueCol, i+2)

		date := time.Date(record.Year, time.Month(record.Month), record.Day, 0, 0, 0, 0, time.UTC)
		template.SetCellValue("Источник", indexCoord, i)
		template.SetCellValue("Источник", departmentCoord, record.Department)
		template.SetCellValue("Источник", dateCoord, date)
		template.SetCellValue("Источник", valueCoord, record.Temperature)
	}
	return template
}
