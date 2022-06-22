package storage

import (
	"context"
	"fmt"
	"strconv"

	"github.com/graph-gophers/dataloader"
	"github.com/kaigoh/loggo/models"
	"gorm.io/gorm"
)

type EventReader struct {
	tx *gorm.DB
}

func (c *EventReader) GetEvents(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	ids := make([]string, len(keys))
	for ix, key := range keys {
		ids[ix] = key.String()
	}
	var events []*models.Event
	c.tx.Where("id IN ?", ids).Find(&events)
	eventByID := map[string]*models.Event{}
	for _, v := range events {
		eventByID[strconv.Itoa(int(v.ID))] = v
	}
	output := make([]*dataloader.Result, len(keys))
	for index, eventKey := range keys {
		event, ok := eventByID[eventKey.String()]
		if ok {
			output[index] = &dataloader.Result{Data: event, Error: nil}
		} else {
			err := fmt.Errorf("user not found %s", eventKey.String())
			output[index] = &dataloader.Result{Data: nil, Error: err}
		}
	}
	return output
}

func GetEvent(ctx context.Context, eventID uint) (*models.Event, error) {
	loaders := For(ctx)
	thunk := loaders.EventLoader.Load(ctx, dataloader.StringKey(strconv.Itoa(int(eventID))))
	result, err := thunk()
	if err != nil {
		return nil, err
	}
	return result.(*models.Event), nil
}
