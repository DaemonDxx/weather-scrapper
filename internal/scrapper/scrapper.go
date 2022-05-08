package scrapper

import (
	"context"
	"fmt"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	"strings"
	"sync"
	"temperature/internal/notify"
	"temperature/internal/storage"
	weather "temperature/internal/weather"
)

var logger = log.New()

type Scrapper struct {
	config     *Config
	weatherAPI *weather.API
	storage    storage.Storage
	cron       *cron.Cron
	notifier   notify.Notifier
}

func New(c *Config) *Scrapper {
	app := Scrapper{
		config: c,
		cron:   cron.New(),
	}
	app.initWeatherAPI()
	app.initStorage()
	app.initCron()
	app.initNotifier()
	return &app
}

func (scrapper *Scrapper) initWeatherAPI() {
	logger.Info("Инициализация модуля погоды")
	source := weather.NewOpenWeatherAPI(&scrapper.config.Token)
	//source := sources.FakeSource{}
	api, err := weather.New(source)
	if err != nil {
		logger.Fatalf("Ошибка инициализации модуля погоды : %s", err)
	}
	scrapper.weatherAPI = api
	logger.Info("Модуль погоды инициализирован")
}

func (scrapper *Scrapper) initStorage() {
	logger.Info("Инициализация модуля storage")
	repo := storage.New(&scrapper.config.DatabasePath)
	err := repo.Init()
	if err != nil {
		logger.Fatalf("Repository init error: %s", err)
	}
	storage := storage.NewDBStorage(repo)
	scrapper.storage = storage
	logger.Info("Модуль storage инициализирован")
}

func (scrapper *Scrapper) initCron() {
	logger.Info("Инициализация модуля cron")
	_, err := scrapper.cron.AddFunc(scrapper.config.Schedule, func() {
		logger.Info("Старт получения данных о погоде")
		results, err := scrapper.update()
		if err != nil {
			logger.Errorf("Не удалось получить данные: %s", err)
			scrapper.notifier.Emit(newErrorUpdateMessage())
		} else {
			logger.Info("Данные о погоде получены")
			scrapper.notifier.Emit(*newSuccessUpdateMessage(results))
		}
	})
	if err != nil {
		logger.Fatalf("Модуль cron завершился с ошибкой: %s", err)
	}
	logger.Info("Модуль cron инициализирован")
}

func (scrapper *Scrapper) initNotifier() {
	logger.Info("Инициализация модуля уведомления")
	notifier, err := notify.NewTelegramNotifier(scrapper.config.Notifier.Telegram)
	if err != nil {
		logger.Fatalf("Модуль уведомления выдал ошибку: %e", err)
	}
	scrapper.notifier = notifier
	logger.Info("Модуль уведомления унициализирован")
}

func (scrapper *Scrapper) Run() {
	logger.Info("Run application")
	scrapper.cron.Start()
}

func (scrapper *Scrapper) update() ([]*weather.Temperature, error) {
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
			logger.WithFields(log.Fields{
				"филиал":      r.Location.Description,
				"температура": r.Value,
			}).Info("Получено значение")
		case err := <-errorChan:
			return nil, err
		}
	}
	if err := scrapper.storage.SaveTemperatureByLastDay(results); err != nil {
		return nil, err
	}
	return results, nil
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

func newSuccessUpdateMessage(results []*weather.Temperature) *string {
	builder := strings.Builder{}
	builder.WriteString("Обновление прошло успешно! \n\r")
	for _, item := range results {
		builder.WriteString(fmt.Sprintf("%s - %0.1fC; \n\r", item.Location.Description, item.Value))
	}
	message := builder.String()
	return &message
}

func newErrorUpdateMessage() string {
	return "Обновление прошло не удачно"
}
