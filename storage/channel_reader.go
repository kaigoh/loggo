package storage

import (
	"context"
	"fmt"
	"strconv"

	"github.com/graph-gophers/dataloader"
	"github.com/kaigoh/loggo/models"
	"gorm.io/gorm"
)

type ChannelReader struct {
	tx *gorm.DB
}

func (c *ChannelReader) GetChannels(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	ids := make([]string, len(keys))
	for ix, key := range keys {
		ids[ix] = key.String()
	}
	var channels []*models.Channel
	c.tx.Where("id IN ?", ids).Find(&channels)
	channelByID := map[string]*models.Channel{}
	for _, v := range channels {
		channelByID[strconv.Itoa(int(v.ID))] = v
	}
	output := make([]*dataloader.Result, len(keys))
	for index, channelKey := range keys {
		channel, ok := channelByID[channelKey.String()]
		if ok {
			output[index] = &dataloader.Result{Data: channel, Error: nil}
		} else {
			err := fmt.Errorf("user not found %s", channelKey.String())
			output[index] = &dataloader.Result{Data: nil, Error: err}
		}
	}
	return output
}

func GetChannel(ctx context.Context, channelID uint) (*models.Channel, error) {
	loaders := For(ctx)
	thunk := loaders.ChannelLoader.Load(ctx, dataloader.StringKey(strconv.Itoa(int(channelID))))
	result, err := thunk()
	if err != nil {
		return nil, err
	}
	return result.(*models.Channel), nil
}
