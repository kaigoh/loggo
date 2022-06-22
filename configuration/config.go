package configuration

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/creasty/defaults"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
)

type Config struct {
	defaultsLoaded  bool
	Timezone        string `default:"Etc/UTC" yaml:"timezone" envconfig:"TZ"`
	DefaultEntryTTL string `default:"672h" yaml:"default_entry_ttl" envconfig:"DEFAULT_ENTRY_TTL"`
	Database        struct {
		Type                string `default:"sqlite" yaml:"type" envconfig:"DATABASE_TYPE"`
		DSN                 string `default:"" yaml:"dsn" envconfig:"DATABASE_DSN"`
		SQLiteDataDirectory string `default:"data" yaml:"sqlite_directory" envconfig:"DATABASE_PATH"`
	} `yaml:"database"`
	Server struct {
		URL      string `default:"http://127.0.0.1:8080" yaml:"base_url" envconfig:"BASE_URL"`
		HTTPPort uint   `default:"8080" yaml:"http_port" envconfig:"HTTP_PORT"`
		MQTTPort uint   `default:"1883" yaml:"mqtt_port" envconfig:"MQTT_PORT"`
	} `yaml:"server"`
	Ntfy struct {
		Enabled  bool   `default:"false" yaml:"enabled" envconfig:"NTFY_ENABLED"`
		Endpoint string `default:"" yaml:"endpoint" envconfig:"NTFY_ENDPOINT"`
	} `yaml:"ntfy"`
}

const ConfigFile string = "config.yml"

func (c *Config) setDefaults() error {
	if !c.defaultsLoaded {
		err := defaults.Set(c)
		if err != nil {
			log.Fatal(err)
		}
		c.defaultsLoaded = true
		return err
	}
	return nil
}

// Load configuration (file first, overload with environment second...)
func (c *Config) LoadConfig() (bool, error) {
	c.setDefaults()
	fileResult, _ := c.LoadConfigFromFile()
	envResult, _ := c.LoadConfigFromEnvironment()
	if !fileResult && !envResult {
		return false, fmt.Errorf("unable to load configuration from any source")
	}
	return true, nil
}

// Load configuration from environment variables
func (c *Config) LoadConfigFromEnvironment() (bool, error) {
	c.setDefaults()
	err := envconfig.Process("", &c)
	if err != nil {
		return false, err
	}
	return true, nil
}

// Load configuration from file
func (c *Config) LoadConfigFromFile() (bool, error) {
	c.setDefaults()
	info, err := os.Stat(ConfigFile)
	if os.IsNotExist(err) {
		return false, fmt.Errorf(ConfigFile + " not found")
	}
	if info.IsDir() {
		return false, fmt.Errorf(ConfigFile + " is a directory!")
	}
	f, err := os.Open(ConfigFile)
	if err != nil {
		return false, err
	}
	defer f.Close()
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&c)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *Config) GetDefaultEntryTTL() (time.Duration, error) {
	return time.ParseDuration(c.DefaultEntryTTL)
}
