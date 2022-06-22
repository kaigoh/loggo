package models

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"time"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

type NewEvent struct {
	ChannelID    uint
	Source       string     `binding:"required" header:"x-loggo-source" form:"source" json:"source" xml:"source" yaml:"source" toml:"source"`
	Level        EventLevel `binding:"required" header:"x-loggo-level" form:"level" json:"level" xml:"level" yaml:"level" toml:"level"`
	Timestamp    *string    `header:"x-loggo-timestamp" form:"timestamp" json:"timestamp" xml:"timestamp" yaml:"timestamp" toml:"timestamp"`
	Title        *string    `header:"x-loggo-title" form:"title" json:"title" xml:"title" yaml:"title" toml:"title"`
	Message      string     `binding:"required" header:"x-loggo-message" form:"message" json:"message" xml:"message" yaml:"message" toml:"message"`
	DataMIMEType *string
	Data         *[]byte `form:"data" json:"data" xml:"data" yaml:"data" toml:"data"`
}

func (e *NewEvent) GetTimestamp() (timestamp time.Time, err error) {
	if e.Timestamp != nil {
		return time.Parse(*e.Timestamp, "RFC3339Nano")
	}
	return time.Now(), nil
}

func (e *NewEvent) ToEvent() (event Event, err error) {
	event.ChannelID = e.ChannelID
	event.Timestamp, err = e.GetTimestamp()
	event.Source = e.Source
	event.Level = e.Level
	event.Title = e.Title
	event.Message = e.Message
	// ToDo: Handle data payload...
	return
}

func (e *NewEvent) FromJSON(data []byte) (err error) {
	err = json.Unmarshal(data, &e)
	if err == nil {
		mt := "application/json"
		e.DataMIMEType = &mt
	}
	return
}

func (e *NewEvent) FromXML(data []byte) (err error) {
	err = xml.Unmarshal(data, &e)
	if err == nil {
		mt := "application/xml"
		e.DataMIMEType = &mt
	}
	return
}

func (e *NewEvent) FromYAML(data []byte) (err error) {
	err = yaml.Unmarshal(data, &e)
	if err == nil {
		mt := "application/yaml"
		e.DataMIMEType = &mt
	}
	return
}

func (e *NewEvent) FromTOML(data []byte) (err error) {
	err = toml.Unmarshal(data, &e)
	if err == nil {
		mt := "application/toml"
		e.DataMIMEType = &mt
	}
	return
}

func (e *NewEvent) FromData(data []byte) (err error) {
	err = e.FromJSON(data)
	if err == nil {
		return nil
	}
	err = e.FromXML(data)
	if err == nil {
		return nil
	}
	err = e.FromYAML(data)
	if err == nil {
		return
	}
	err = e.FromTOML(data)
	if err == nil {
		return
	}
	return fmt.Errorf("unable to bind input data to NewError")
}
