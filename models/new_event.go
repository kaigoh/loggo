package models

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

type NewEvent struct {
	ChannelID    uint
	Source       string     `binding:"required" header:"Loggo-Source" form:"source" json:"source" xml:"source" yaml:"source" toml:"source"`
	Level        EventLevel `binding:"required" header:"Loggo-Level" form:"level" json:"level" xml:"level" yaml:"level" toml:"level"`
	Timestamp    *string    `header:"Loggo-Timestamp" form:"timestamp" json:"timestamp" xml:"timestamp" yaml:"timestamp" toml:"timestamp"`
	Title        *string    `header:"Loggo-Title" form:"title" json:"title" xml:"title" yaml:"title" toml:"title"`
	Message      string     `binding:"required" header:"Loggo-Message" form:"message" json:"message" xml:"message" yaml:"message" toml:"message"`
	DataMIMEType *string    `json:"-"`
	Data         *[]byte    `form:"data" json:"data" xml:"data" yaml:"data" toml:"data"`
}

func (e *NewEvent) GetTimestamp() (timestamp time.Time, err error) {
	if e.Timestamp != nil {
		ts := *e.Timestamp
		if len(strings.TrimSpace(ts)) > 0 {
			return time.Parse(*e.Timestamp, "RFC3339Nano")
		}
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
	if e.Data != nil {
		event.HasData = true
		event.EventData.Data = *e.Data
		// Try and detect the MIME type of the payload...
		mimetype.SetLimit(1024 * 1024)
		mtype := mimetype.Detect(event.EventData.Data)
		event.EventData.DataMIMEType = mtype.String()
	}
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
	return fmt.Errorf("unable to bind input data to NewEvent")
}
