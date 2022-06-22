package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/kaigoh/loggo/configuration"
	"gorm.io/gorm"
)

type Channel struct {
	ID        uint    `gorm:"primaryKey" json:"id"`
	UUID      string  `gorm:"index:idx_loggo_channel_uuid,unique; size:64; not null; column:uuid;" json:"uuid"`
	Name      string  `gorm:"index:idx_loggo_channel_name,unique; size:128; not null;" json:"name"`
	TTL       *string `gorm:"size:64;" json:"ttl"`
	MQTT      bool    `gorm:"default:true; column:mqtt_enabled; not null;" json:"mqtt"`
	MQTTTopic *string `gorm:"column:mqtt_topic;" json:"mqtt_topic"`
	Ntfy      bool    `gorm:"default:true; column:ntfy_enabled; not null;" json:"ntfy"`
	NtfyTopic *string `gorm:"column:ntfy_topic;" json:"ntfy_topic"`
	Events    []Event `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}

func (c *Channel) AfterFind(tx *gorm.DB) (err error) {

	// Make sure we have a topic name...
	topic := strings.ToLower(c.Name)
	if c.MQTTTopic == nil {
		c.MQTTTopic = &topic
	}
	if c.NtfyTopic == nil {
		c.NtfyTopic = &topic
	}

	return

}

func (c *Channel) GetTTL(config *configuration.Config) (time.Duration, error) {
	if c.TTL != nil {
		return time.ParseDuration(*c.TTL)
	}
	return time.ParseDuration(config.DefaultEntryTTL)
}

func ChannelByName(tx *gorm.DB, name string) (channel *Channel, err error) {
	result := tx.Where("name LIKE ?", name).Find(&channel)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("channel not found")
	}
	return
}

func ChannelByMQTTTopic(tx *gorm.DB, topic string) (channel *Channel, err error) {
	result := tx.Where("(mqtt_topic LIKE ?) OR (name LIKE ? AND mqtt_topic IS NULL)", topic, topic).Limit(1).Find(&channel)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("channel not found")
	}
	return
}

func ChannelByNtfyTopic(tx *gorm.DB, topic string) (channel *Channel, err error) {
	result := tx.Where("(ntfy_topic LIKE ?) OR (name LIKE ? AND ntfy_topic IS NULL)", topic, topic).Limit(1).Find(&channel)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("channel not found")
	}
	return
}
