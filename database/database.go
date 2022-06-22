package database

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kaigoh/loggo/configuration"
	"github.com/kaigoh/loggo/models"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB
var config *configuration.Config
var tzLocation time.Location

func SetTimezone() {
	loc, tze := time.LoadLocation(config.Timezone)
	if tze != nil {
		log.Println(config.Timezone)
		panic("Unable to load requested timezone!")
	}
	tzLocation = *loc
}

// Connect to a database
func Connect(c *configuration.Config) *gorm.DB {

	config = c

	// Set the timezone...
	SetTimezone()

	// If the database has been overridden, try and connect...
	switch strings.ToLower(config.Database.Type) {
	case "mysql":
		DB = connectMysql()
	case "postgres":
		DB = connectPostgres()
	case "sqlserver":
		DB = connectSQLServer()
	default:
		DB = connectSQLite()
	}

	// Run migrations
	Migrate(DB)

	return DB

}

func getLogger() logger.Interface {
	return logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)
}

func connectMysql() *gorm.DB {
	db, err := gorm.Open(mysql.Open(config.Database.DSN), &gorm.Config{
		Logger:      getLogger(),
		PrepareStmt: true,
	})
	if err != nil {
		panic("MySQL Error: Unable to connect to database, please check your configuration and restart the server")
	}
	return db
}

func connectPostgres() *gorm.DB {
	db, err := gorm.Open(postgres.Open(config.Database.DSN), &gorm.Config{
		Logger:      getLogger(),
		PrepareStmt: true,
	})
	if err != nil {
		panic("Postgres Error: Unable to connect to database, please check your configuration and restart the server")
	}
	return db
}

func connectSQLServer() *gorm.DB {
	db, err := gorm.Open(sqlserver.Open(config.Database.DSN), &gorm.Config{
		Logger:      getLogger(),
		PrepareStmt: true,
	})
	if err != nil {
		panic("SQL Server Error: Unable to connect to database, please check your configuration and restart the server")
	}
	return db
}

func connectSQLite() *gorm.DB {

	path := config.Database.SQLiteDataDirectory
	// Create the database path if needed...
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, 0774)
	}
	filename := filepath.Join(path, "loggo.db")

	db, err := gorm.Open(sqlite.Open(filename), &gorm.Config{
		Logger:      getLogger(),
		PrepareStmt: true,
	})
	if err != nil {
		panic("SQLite Error: Unable to connect to database, please check your configuration and restart the server")
	}
	return db
}

// Migrate all models
func Migrate(db *gorm.DB) {
	if err := db.AutoMigrate(&models.Channel{}, &models.Event{}); err != nil {
		panic(err.Error())
	}
}

func Paginate(page int, pageSize int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page < 1 {
			page = 1
		}
		if pageSize < 1 {
			pageSize = 20
		}
		offset := (page - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}
