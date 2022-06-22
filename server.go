package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/kaigoh/loggo/configuration"
	"github.com/kaigoh/loggo/database"
	"github.com/kaigoh/loggo/graph"
	"github.com/kaigoh/loggo/graph/generated"
	"github.com/kaigoh/loggo/middleware"
	"github.com/kaigoh/loggo/models"
	mqtt "github.com/mochi-co/mqtt/server"
	"github.com/mochi-co/mqtt/server/events"
	"github.com/mochi-co/mqtt/server/listeners"
	"gorm.io/gorm"
)

var config configuration.Config
var db *gorm.DB
var mqttServer *mqtt.Server

func main() {

	fmt.Println("--------------")
	fmt.Println(" Loggo Server ")
	fmt.Println("--------------")

	// Load config...
	_, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Set the "PORT" variable...
	os.Setenv("PORT", strconv.Itoa(int(config.Server.HTTPPort)))

	// Connect to the database...
	db = database.Connect(&config)

	// Start cron tasks...
	go cronHandler()

	// MQTT
	mqttServer = mqtt.NewServer(nil)
	tcp := listeners.NewTCP("t1", ":"+strconv.Itoa(int(config.Server.MQTTPort)))
	err = mqttServer.AddListener(tcp, nil)
	if err != nil {
		log.Fatal(err)
	}
	mqttServer.Events.OnMessage = func(cl events.Client, pk events.Packet) (pkx events.Packet, err error) {
		if pk.FixedHeader.Type == byte(3) {
			// Try and decode the payload...
			var newEvent models.NewEvent
			err := newEvent.FromData(pk.Payload)
			if err != nil {
				log.Println(err)
				return pkx, err
			}
			// Get the channel from the topic...
			channel, err := models.ChannelByMQTTTopic(db, pk.TopicName)
			if err != nil {
				log.Println(err)
				return pkx, err
			}
			newEvent.ChannelID = channel.ID
			event, err := newEvent.ToEvent()
			if err != nil {
				log.Println(err)
				return pkx, err
			}
			db.Save(&event)
			return pkx, nil
		}

		return pk, nil
	}
	go mqttHandler()

	// HTTP
	r := gin.Default()
	r.Use(middleware.GinContextToContextMiddleware())
	r.Use(gzip.Gzip(gzip.DefaultCompression))
	r.POST("/api", graphqlHandler(db))
	r.GET("/playground", playgroundHandler())
	r.POST("/channel/:channelName/event", func(c *gin.Context) {

	})
	r.GET("/channel/:channelName/event/:eventId/data", func(c *gin.Context) {
		var data *models.EventData
		sub := db.Select("id").Where("name = ?", c.Param("channelName")).Limit(1).Model(&models.Channel{})
		result := db.Where("id = ? AND channel_id IN (?)", c.Param("eventId"), sub).Find(&data)
		if result.Error != nil {
			c.AbortWithError(500, result.Error)
			return
		}
		if result.RowsAffected == 0 {
			c.AbortWithStatus(404)
			return
		}
		c.Data(200, data.DataMIMEType, data.Data)
	})
	r.GET("/mqtt/:topic/:message", func(c *gin.Context) {
		err := mqttServer.Publish(c.Param("topic"), []byte(c.Param("message")), false)
		if err != nil {
			c.AbortWithError(500, err)
		}
	})
	r.Run()

}

func cronHandler() {
	log.Println("Starting cron tasks")

	ticker := time.NewTicker(1 * time.Hour)
	for range ticker.C {
		log.Println("Running cron tasks...")

		// Purge expired events...
		var channels []*models.Channel
		db.Find(channels)
		if len(channels) > 0 {
			for _, c := range channels {
				// Get the TTL for the channels events...
				ttl, err := c.GetTTL(&config)
				if err != nil {
					log.Println("Unable to process TTL for channel '" + c.Name + "' - events will NOT be purged!")
				} else {
					// ...and purge events which have expired
					window := time.Now().Add(-ttl)
					result := db.Where("channel_id = ? AND timestamp <= ?", c.ID, window).Delete(&models.Event{})
					if result.Error != nil {
						log.Println("Database error pruning events for channel '"+c.Name+"'", result.Error)
					} else {
						if result.RowsAffected > 0 {
							log.Println("Purged " + strconv.Itoa(int(result.RowsAffected)) + " events for channel '" + c.Name + "'")
						}
					}
				}
			}
		}

	}
}

func mqttHandler() {
	err := mqttServer.Serve()
	if err != nil {
		log.Fatal(err)
	}
}

func graphqlHandler(tx *gorm.DB) gin.HandlerFunc {
	// NewExecutableSchema and Config are in the generated.go file
	// Resolver is in the resolver.go file
	c := generated.Config{Resolvers: &graph.Resolver{
		DB:     tx,
		Config: &config,
	}}

	h := handler.New(generated.NewExecutableSchema(c))

	// Configure WebSocket with CORS
	h.AddTransport(&transport.Websocket{
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		KeepAlivePingInterval: 10 * time.Second,
	})

	h.AddTransport(transport.Options{})
	h.AddTransport(transport.POST{})

	h.SetQueryCache(lru.New(1000))
	h.Use(extension.FixedComplexityLimit(100))
	h.Use(extension.Introspection{})

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func playgroundHandler() gin.HandlerFunc {
	h := playground.Handler("Loggo GraphQL API", "/api")

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
