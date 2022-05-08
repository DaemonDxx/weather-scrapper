package notify

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	log "github.com/sirupsen/logrus"
	"sync"
)

const (
	subscribeCommand   = "subscribe"
	unSubscribeCommand = "unsubscribe"
)

var ErrListen = fmt.Errorf("Telegram сервис уже прослушивается")
var ErrHasSubscribing = fmt.Errorf("Пользователь уже подписан")
var ErrHasDeleting = fmt.Errorf("Пользователь не был ранее подписан")
var ErrEmptyConfig = fmt.Errorf("Не задана конфигурация Telegram Bot")

type TelegramNotifierConfig struct {
	Token string `yaml:"token"`
}

type TelegramSendError struct {
	Username string
	ChatID   int64
	Command  string
	err      error
}

func (e *TelegramSendError) Error() string {
	return e.err.Error()
}

func NewTgSendError(update *tgbotapi.Update, err error) *TelegramSendError {
	return &TelegramSendError{
		Username: update.Message.From.UserName,
		ChatID:   update.Message.Chat.ID,
		Command:  update.Message.Command(),
		err:      err,
	}
}

type UserID int64

type UserStorage interface {
	All() []*UserID
	Save(u *UserID) error
	Delete(u *UserID) error
}

type MemoryStorage struct {
	db    map[UserID]*UserID
	mutex sync.Mutex
}

func NewMemoryStorage() UserStorage {
	db := make(map[UserID]*UserID)
	return &MemoryStorage{db: db}
}

func (m *MemoryStorage) All() []*UserID {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	res := make([]*UserID, 0, len(m.db))
	for _, id := range m.db {
		res = append(res, id)
	}
	return res
}

func (m *MemoryStorage) Save(u *UserID) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	user := *u
	_, ok := m.db[user]
	if ok {
		return ErrHasSubscribing
	}

	m.db[user] = &user
	return nil
}

func (m *MemoryStorage) Delete(u *UserID) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	user := *u
	_, ok := m.db[user]
	if !ok {
		return ErrHasDeleting
	}
	delete(m.db, user)
	return nil
}

func NewEmitMessage(chatID int64, message string) tgbotapi.MessageConfig {
	return tgbotapi.NewMessage(chatID, message)
}

func NewAddSubscriberMessage(chatID int64, username string) tgbotapi.MessageConfig {
	message := fmt.Sprintf("%s, вы подписались на обновления!", username)
	return tgbotapi.NewMessage(chatID, message)
}

func NewDeleteSubscriberMessage(chatID int64) tgbotapi.MessageConfig {
	message := fmt.Sprintf("Выбольше не будете получать сообщения")
	return tgbotapi.NewMessage(chatID, message)
}

func NewErrorMessage(chatID int64, err error) tgbotapi.MessageConfig {
	message := fmt.Sprintf("Не удалось выполнить операцию - %s", err.Error())
	return tgbotapi.NewMessage(chatID, message)
}

type TelegramNotifier struct {
	tg          *tgbotapi.BotAPI
	db          UserStorage
	isListening bool
}

func NewTelegramNotifier(config *TelegramNotifierConfig) (Notifier, error) {
	if config == nil {
		return nil, ErrEmptyConfig
	}
	if config.Token == "" {
		return nil, ErrEmptyConfig
	}

	bot, err := tgbotapi.NewBotAPI(config.Token)
	if err != nil {
		return nil, err
	}
	db := NewMemoryStorage()

	notifier := TelegramNotifier{
		tg:          bot,
		db:          db,
		isListening: false,
	}

	err = notifier.listen()
	return &notifier, err
}

func (n *TelegramNotifier) listen() error {
	if n.isListening {
		return ErrListen
	}
	go func() {
		log.Info("Прослушивание Telegram бота запущено")
		updateConfig := tgbotapi.NewUpdate(0)
		updateConfig.Timeout = 60
		updates := n.tg.GetUpdatesChan(updateConfig)
		for update := range updates {
			if update.Message == nil {
				continue
			}

			if !update.Message.IsCommand() {
				continue
			}

			chatID := update.Message.Chat.ID
			userID := UserID(update.Message.Chat.ID)
			username := update.Message.From.UserName
			logFields := log.Fields{
				"module":   "telegram-notifier",
				"username": username,
			}

			var dbErr error
			var message tgbotapi.MessageConfig

			switch update.Message.Command() {
			case subscribeCommand:
				log.WithFields(logFields).Info("Запрос на подписку")

				dbErr = n.db.Save(&userID)
				if dbErr == nil {
					log.WithFields(logFields).Info("Подписан новый пользователь")
					message = NewAddSubscriberMessage(chatID, username)
				}

			case unSubscribeCommand:
				log.WithFields(logFields).Info("Запрос на отписку")
				dbErr = n.db.Delete(&userID)
				if dbErr == nil {
					log.WithFields(logFields).Info("Пользователь отписался")
					message = NewDeleteSubscriberMessage(chatID)
				}
			}

			if dbErr != nil {
				log.WithFields(logFields).Errorf("Неудачное выполнение команды - %s", dbErr)
				errMessage := NewErrorMessage(chatID, dbErr)
				_, err := n.tg.Send(errMessage)
				if err != nil {
					tgErrorHandler(NewTgSendError(&update, err))
				}
			} else {
				_, err := n.tg.Send(message)
				if err != nil {
					tgErrorHandler(NewTgSendError(&update, err))
				}
			}
		}
	}()
	return nil
}

func (n *TelegramNotifier) Emit(message string) {
	for _, userID := range n.db.All() {
		chatID := int64(*userID)
		tgMessage := NewEmitMessage(chatID, message)
		_, err := n.tg.Send(tgMessage)
		if err != nil {
			log.WithField("ChatID", chatID).Error("Не удалось отправить уведомление")
		}
	}
}

func tgErrorHandler(err *TelegramSendError) {
	if err != nil {
		log.WithFields(log.Fields{
			"From":    err.Username,
			"ChatID":  err.ChatID,
			"Command": err.Command,
		}).Errorf("Сообщение не отправлено - %s", err)
	}
}
