package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
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

			// Parse the topic name is if it were a URL...
			u, err := url.Parse(pk.TopicName)
			if err != nil {
				return pkx, err
			}
			p := strings.Split(u.Path, "/")
			var s []string
			for _, e := range p {
				if len(e) > 0 {
					s = append(s, e)
				}
			}
			if len(s) < 2 || s[0] != "channel" {
				return pkx, fmt.Errorf("topic '" + s[1] + "' is not a valid channel")
			}

			// Get the channel from the topic...
			channel, err := models.ChannelByMQTTTopic(db, s[1])
			if err != nil {
				return pkx, err
			}

			// Our new event...
			var newEvent models.NewEvent
			newEvent.ChannelID = channel.ID

			params := u.Query()

			// If we want data stored with the event, the new event data comes from "URL" parameters in the topic
			// This means if we are just firing events with no payload, we can upload that data in multiple formats,
			// otherwise if we have query parameters in the topic, we have to assume that there is a data payload...
			if len(params) > 0 {

				// Ensure all the parameter keys are lower case...
				for k, v := range params {
					params[strings.ToLower(k)] = v
				}

				newEvent.Source = params.Get("source")
				newEvent.Level = models.EventLevel(params.Get("level"))
				ts := params.Get("timestamp")
				if len(ts) > 0 {
					newEvent.Timestamp = &ts
				}
				title := params.Get("title")
				if len(title) > 0 {
					newEvent.Title = &title
				}
				newEvent.Message = params.Get("message")
				newEvent.Data = &pk.Payload

			} else {

				// Try and decode the payload...
				err = newEvent.FromData(pk.Payload)
				if err != nil {
					log.Println(err)
					return pkx, err
				}

			}

			event, err := newEvent.ToEvent()
			if err != nil {
				return pkx, err
			}

			pkx.WillRetain = true
			db.Create(&event)

			out, err := event.ToJSON()
			if err != nil {
				return pk, err
			}

			pkx.Payload = out

			// Put the event on the wire...
			defer publishEvent(channel, &event, &out)

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

	r.GET("/channel/:channelName/event", func(c *gin.Context) {

		// Get the channel...
		channel, err := models.ChannelByName(db, c.Param("channelName"))
		if err != nil {
			c.AbortWithError(404, err)
			return
		}

		// Create a new event...
		var n models.NewEvent
		n.ChannelID = channel.ID

		// Try headers first, then fall back to a standard binding...
		err = c.BindHeader(&n)
		if err != nil {
			err = c.Bind(&n)
			if err != nil {
				c.AbortWithError(400, err)
				return
			}
		}

		event, err := n.ToEvent()
		if err != nil {
			c.AbortWithError(500, err)
			return
		}

		// Save it...
		db.Create(&event)

		// Put the event on the wire...
		defer publishEvent(channel, &event, nil)

		c.JSON(200, event)

	})

	r.POST("/channel/:channelName/event", func(c *gin.Context) {

		// Get the channel...
		channel, err := models.ChannelByName(db, c.Param("channelName"))
		if err != nil {
			c.AbortWithError(404, err)
			return
		}

		// Create a new event...
		var n models.NewEvent
		n.ChannelID = channel.ID

		// Try headers first, then fall back to a standard binding...
		err = c.BindHeader(&n)
		if err != nil {
			err = c.Bind(&n)
			if err != nil {
				c.AbortWithError(400, err)
				return
			}
		}

		// Append the body to it...
		body := c.Request.Body
		d, err := ioutil.ReadAll(body)
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
		n.Data = &d
		event, err := n.ToEvent()
		if err != nil {
			c.AbortWithError(500, err)
			return
		}

		// Save it...
		db.Create(&event)

		// Put the event on the wire...
		defer publishEvent(channel, &event, nil)

		c.JSON(200, event)
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
		err := mqttServer.Publish("/channel/"+c.Param("topic"), []byte(c.Param("message")), false)
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

func publishEvent(channel *models.Channel, event *models.Event, json *[]byte) (err error) {
	var out []byte
	if json == nil {
		out, err = event.ToJSON()
		if err != nil {
			return err
		}
	} else {
		out = *json
	}
	return mqttServer.Publish("/channel/"+*channel.MQTTTopic, out, false)
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
