package models

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"time"

	"github.com/kaigoh/loggo/configuration"
	"gorm.io/gorm"
)

type Event struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	ChannelID uint       `gorm:"index:idx_loggo_event_channel; index:idx_loggo_event,0; not null;" json:"channel_id"`
	Channel   Channel    `json:"-"`
	CreatedAt time.Time  `json:"created_at"`
	Source    string     `gorm:"index:idx_loggo_event,1; not null; size:128;" json:"source"`
	Level     EventLevel `gorm:"index:idx_loggo_event,2; not null;" json:"level"`
	Timestamp time.Time  `gorm:"index:idx_loggo_event,3,sort:desc; not null;" json:"timestamp"`
	Title     *string    `gorm:"size:128;" json:"title"`
	Message   string     `gorm:"size:512; not null;" json:"message"`
	HasData   bool       `gorm:"not null" json:"has_data"`
	EventData EventData  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}

func (e *Event) BeforeCreate(tx *gorm.DB) (err error) {
	if e.EventData.Data != nil {
		e.HasData = true
	}
	return
}

func (e *Event) BeforeSave(tx *gorm.DB) (err error) {
	if e.EventData.Data != nil {
		e.HasData = true
	}
	return
}

func (e *Event) GetDataURL(config *configuration.Config, tx *gorm.DB) (uri *string, err error) {
	var name string
	result := tx.Select("name").Where("id = ?", e.ChannelID).Model(&Channel{}).Find(&name)
	if result.Error != nil {
		return nil, result.Error
	}
	u, err := url.Parse(config.Server.URL)
	if err != nil {
		return nil, err
	}
	u.Path = fmt.Sprintf("/channel/%s/event/%d/data/", name, e.ID)
	compiled := u.String()
	return &compiled, nil
}

func (e *Event) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

type EventData struct {
	ID           uint `gorm:"primaryKey" json:"id"`
	EventID      uint
	DataMIMEType string `gorm:"size:128; column:data_mime_type; not null;" json:"data_mime_type"`
	Data         []byte `json:"data"`
}

type EventLevel string

const (
	EventLevelDebug   EventLevel = "debug"
	EventLevelInfo    EventLevel = "info"
	EventLevelWarning EventLevel = "warning"
	EventLevelError   EventLevel = "error"
	EventLevelFatal   EventLevel = "fatal"
)

var AllEventLevel = []EventLevel{
	EventLevelDebug,
	EventLevelInfo,
	EventLevelWarning,
	EventLevelError,
	EventLevelFatal,
}

func (e EventLevel) IsValid() bool {
	switch e {
	case EventLevelDebug, EventLevelInfo, EventLevelWarning, EventLevelError, EventLevelFatal:
		return true
	}
	return false
}

func (e EventLevel) String() string {
	return string(e)
}

func (e *EventLevel) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = EventLevel(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid EventLevel", str)
	}
	return nil
}

func (e EventLevel) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}
