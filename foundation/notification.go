package foundation

import (
	"strings"
	"time"

	"github.com/weitecit/pkg/log"
	"github.com/weitecit/pkg/utils"

	"encoding/json"
	"fmt"
)

type NotifyLevel2 utils.Enum

const (
	NotifyLevelInfo    NotifyLevel2 = "info"
	NotifyLevelWarning NotifyLevel2 = "warning"
	NotifyLevelError   NotifyLevel2 = "error"
)

type NotifyCode utils.Enum

const (
	NotifyCodeNone NotifyCode = ""
)

type (
	Message struct {
		Text  string
		Level NotifyLevel2
		Code  NotifyCode
	}

	Notification struct {
		Text   string       `json:"text" bson:"text,omitempty"`
		Level  NotifyLevel2 `json:"level" bson:"level,omitempty"`
		Count  int          `json:"count" bson:"count,omitempty"`
		Users  []string     `json:"users" bson:"users"`
		Code   NotifyCode   `json:"code" bson:"code,omitempty"`
		Source string       `json:"source_id" bson:"source_id,omitempty"`
		Viewed *time.Time   `json:"viewed" bson:"viewed,omitempty"`
	}
)

// Para poder modificar el array debe ser un puntero
type Notifiable interface {
	GetNotifications() *[]*Notification
}

func (m *Notification) ToJSON() string {
	o, err := json.MarshalIndent(&m, "", "\t")
	if err != nil {
		log.Err(err)
		return "Error in conversion"
	}
	return string(o)
}

func Notify(i Notifiable, message *Message, users []string, source string) *[]*Notification {

	// TODO: dependiendo del nivel se puede enviar un mensaje al usuario actual o hacer un log
	m := &Notification{
		Text:   strings.ToLower(message.Text),
		Level:  message.Level,
		Users:  users,
		Code:   message.Code,
		Source: source,
	}

	notifications := i.GetNotifications()
	return m.addNotifications(notifications)
}

func (m *Notification) addNotifications(notifications *[]*Notification) *[]*Notification {
	alreadyAdded := false
	m.Count++
	for i, n := range *notifications {
		if n.Text == m.Text && n.Level == m.Level && n.Source == m.Source {
			alreadyAdded = true
			(*notifications)[i].Count++
			break
		}
	}

	if !alreadyAdded {
		*notifications = append(*notifications, m)
	}

	return notifications
}

func (m *Notification) CloneTo(i Notifiable, users []string, source string) *[]*Notification {
	msg := Message{
		Text:  m.Text,
		Level: m.Level,
	}
	return Notify(i, &msg, users, source)
}

func NewWarningWithCode(i Notifiable, message string, code NotifyCode, users []string, source string) *[]*Notification {
	msg := Message{
		Code:  code,
		Text:  message,
		Level: NotifyLevelWarning,
	}
	return Notify(i, &msg, users, source)
}

func NewWarning(i Notifiable, message string, users []string, source string) *[]*Notification {

	return NewWarningWithCode(i, message, NotifyCodeNone, users, source)
}

func NewInfo(i Notifiable, message string, users []string, source string) *[]*Notification {
	msg := Message{
		Text:  message,
		Level: NotifyLevelInfo,
	}
	return Notify(i, &msg, users, source)
}

func NewError(i Notifiable, message string, users []string, source string) *[]*Notification {
	msg := Message{
		Text:  message,
		Level: NotifyLevelError,
	}
	return Notify(i, &msg, users, source)
}

func NewNotification(notifications []*Notification, message *Message) *[]*Notification {

	m := &Notification{
		Text:  strings.ToLower(message.Text),
		Level: message.Level,
	}

	return m.addNotifications(&notifications)
}

func AddNotificationByText(notifications []*Notification, destination Notifiable, message string, users []string, source string) {
	for _, notification := range notifications {
		if notification.Text != message {
			continue
		}
		msg := Message{
			Text:  notification.Text,
			Level: notification.Level,
		}
		Notify(destination, &msg, users, source)
	}
}

func AddNotifications(notifications []*Notification, destination Notifiable, users []string, source string) {
	for _, notification := range notifications {
		msg := Message{
			Text:  notification.Text,
			Level: notification.Level,
		}
		Notify(destination, &msg, users, source)
	}
}

func CountNotifications(notifications []*Notification, message string) int {
	for _, notification := range notifications {
		if notification.Text == message {
			return notification.Count
		}
	}
	return 0
}

func DeleteNotification(notifications []*Notification, message string) []*Notification {
	list := []*Notification{}
	for _, notification := range notifications {
		if notification.Text == message {
			continue
		}
		list = append(list, notification)
	}
	return list
}

func DeleteNotificationIfContains(notifications []*Notification, message string) []*Notification {
	list := []*Notification{}
	for _, notification := range notifications {
		if strings.Contains(notification.Text, message) {
			continue
		}
		list = append(list, notification)
	}
	return list
}

func DeleteNotificationByCode(notifications []*Notification, code NotifyCode) []*Notification {
	list := []*Notification{}
	for _, notification := range notifications {
		if notification.Code == code {
			continue
		}
		list = append(list, notification)
	}
	return list
}

// NotificationService defines the interface for sending notifications
type NotificationService interface {
	SendNotification(userID string, count int) error
}

var defaultNotificationService NotificationService

// SetNotificationService sets the global notification service
func SetNotificationService(service NotificationService) {
	defaultNotificationService = service
}

// SendNotificationToUser sends a notification using the default service
func SendNotificationToUser(userID string, count int) error {
	if userID == "" {
		return fmt.Errorf("user id is empty")
	}

	if defaultNotificationService == nil {
		return fmt.Errorf("notification service not initialized")
	}
	return defaultNotificationService.SendNotification(userID, count)
}
