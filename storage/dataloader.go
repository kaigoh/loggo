package storage

import (
	"context"
	"net/http"

	"github.com/graph-gophers/dataloader"
	"gorm.io/gorm"
)

type ctxKey string

const (
	loadersKey = ctxKey("dataloaders")
)

type Loaders struct {
	ChannelLoader *dataloader.Loader
	EventLoader   *dataloader.Loader
}

func NewLoaders(tx *gorm.DB) *Loaders {
	channelReader := &ChannelReader{tx: tx}
	eventReader := &EventReader{tx: tx}
	loaders := &Loaders{
		ChannelLoader: dataloader.NewBatchedLoader(channelReader.GetChannels),
		EventLoader:   dataloader.NewBatchedLoader(eventReader.GetEvents),
	}
	return loaders
}

func Middleware(loaders *Loaders, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCtx := context.WithValue(r.Context(), loadersKey, loaders)
		r = r.WithContext(nextCtx)
		next.ServeHTTP(w, r)
	})
}

func For(ctx context.Context) *Loaders {
	return ctx.Value(loadersKey).(*Loaders)
}
