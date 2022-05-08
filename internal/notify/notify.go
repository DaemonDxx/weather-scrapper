package notify

import "fmt"

type NotifierError struct {
	notifierType string
	err          error
}

func (e *NotifierError) Error() string {
	return fmt.Sprintf("[%s] Не удалось отправить сообщение: %s", e.notifierType, e.err)
}

type Notifier interface {
	Emit(message string)
}
