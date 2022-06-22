package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/kaigoh/loggo/database"
	"github.com/kaigoh/loggo/graph/generated"
	"github.com/kaigoh/loggo/models"
	"github.com/kaigoh/loggo/storage"
)

func (r *eventResolver) Data(ctx context.Context, obj *models.Event) (*string, error) {
	return obj.GetDataURL(r.Config, r.DB)
}

func (r *queryResolver) GetChannels(ctx context.Context) ([]*models.Channel, error) {
	var channels []*models.Channel
	result := r.DB.Order("name ASC").Find(&channels)
	if result.Error != nil {
		return nil, result.Error
	}
	return channels, nil
}

func (r *queryResolver) GetChannel(ctx context.Context, id uint) (*models.Channel, error) {
	return storage.GetChannel(ctx, id)
}

func (r *queryResolver) GetEvent(ctx context.Context, id uint) (*models.Event, error) {
	return storage.GetEvent(ctx, id)
}

func (r *queryResolver) GetChannelEvents(ctx context.Context, channelID uint, page *uint, pageSize *uint) ([]*models.Event, error) {
	var events []*models.Event
	result := r.DB.Where("channel_id = ?", channelID).Scopes(database.Paginate(int(*page), int(*pageSize))).Order("timestamp DESC").Find(&events)
	if result.Error != nil {
		return nil, result.Error
	}
	return events, nil
}

func (r *queryResolver) GetSourceEvents(ctx context.Context, channelID uint, source string, page *uint, pageSize *uint) ([]*models.Event, error) {
	var events []*models.Event
	result := r.DB.Where("channel_id = ? AND source = ?", channelID, source).Scopes(database.Paginate(int(*page), int(*pageSize))).Order("timestamp DESC").Find(&events)
	if result.Error != nil {
		return nil, result.Error
	}
	return events, nil
}

// Event returns generated.EventResolver implementation.
func (r *Resolver) Event() generated.EventResolver { return &eventResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type eventResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
