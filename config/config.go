// Package config contains structures used in retrieving app configuration
// from disk.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// ServerConfig describes the settings for running the price proxy.
type ServerConfig struct {
	Env       string
	Listen    string
	LogFormat string
	LogLevel  string
}

// PriceConfig describes one price setting, which uses one source.
type PriceConfig struct {
	Source        string  `yaml:"source"`
	Base          string  `yaml:"base"`
	BaseOverride  string  `yaml:"base_override"`
	Quote         string  `yaml:"quote"`
	QuoteOverride string  `yaml:"quote_override"`
	Factor        float64 `yaml:"factor"`
	Wander        bool    `yaml:"wander"`
}

// SourceConfig describes one source setting (e.g. one API endpoint).
// The URL has "{base}" and "{quote}" replaced at runtime with entries from PriceConfig.
type SourceConfig struct {
	Name           string  `yaml:"name"`
	URL            url.URL `yaml:"url"`
	AuthKeyEnvName string  `yaml:"auth_key_env_name"`
	SleepReal      int     `yaml:"sleepReal"`
}

type PriceList []PriceConfig

// Config describes the top level config file format.
type Config struct {
	Server  *ServerConfig   `yaml:"server"`
	Prices  PriceList       `yaml:"prices"`
	Sources []*SourceConfig `yaml:"sources"`
}

func (pl PriceList) GetBySource(source string) PriceList {
	result := PriceList{}

	for _, price := range pl {
		if price.Source == source {
			result = append(result, price)
		}
	}

	return result
}

var (
	// ErrNil indicates that a nil/null pointer was encountered.
	ErrNil = errors.New("nil pointer")

	// ErrMissingEmptyConfigSection indicates that a required config file section is missing (not present) or empty (zero-length).
	ErrMissingEmptyConfigSection = errors.New("config file section is missing/empty")

	// ErrInvalidValue indicates that a value was invalid.
	ErrInvalidValue = errors.New("invalid value")
)

// CheckConfig checks the config for valid structure and values.
func CheckConfig(cfg *Config) error {
	if cfg == nil {
		return ErrNil
	}

	if cfg.Server == nil {
		return fmt.Errorf("%s: %s", ErrMissingEmptyConfigSection.Error(), "server")
	}
	if cfg.Sources == nil {
		return fmt.Errorf("%s: %s", ErrMissingEmptyConfigSection.Error(), "sources")
	}
	if len(cfg.Sources) == 0 {
		return fmt.Errorf("%s: %s", ErrMissingEmptyConfigSection.Error(), "sources")
	}
	for _, sourcecfg := range cfg.Sources {
		if sourcecfg.SleepReal == 0 {
			return fmt.Errorf("%s: sleepReal", ErrInvalidValue.Error())
		}
	}

	if cfg.Prices == nil {
		return fmt.Errorf("%s: %s", ErrMissingEmptyConfigSection.Error(), "prices")
	}
	if len(cfg.Prices) == 0 {
		return fmt.Errorf("%s: %s", ErrMissingEmptyConfigSection.Error(), "prices")
	}
	for _, pricecfg := range cfg.Prices {
		if pricecfg.Factor == 0 {
			return fmt.Errorf("%s: factor", ErrInvalidValue.Error())
		}
	}

	return nil
}

// ConfigureLogging configures logging.
func ConfigureLogging(cfg *ServerConfig) error {
	if cfg == nil {
		return ErrNil
	}

	if cfg.Env != "prod" {
		// https://github.com/sirupsen/logrus#logging-method-name
		// This slows down logging (by a factor of 2).
		log.SetReportCaller(true)
	}

	switch cfg.LogFormat {
	case "json":
		log.SetFormatter(&log.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		})
	case "textcolour":
		log.SetFormatter(&log.TextFormatter{
			ForceColors:     true,
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339Nano,
		})
	case "textnocolour":
		log.SetFormatter(&log.TextFormatter{
			DisableColors:   true,
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339Nano,
		})
	default:
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339Nano,
		}) // with colour if TTY, without otherwise
	}

	if loglevel, err := log.ParseLevel(cfg.LogLevel); err == nil {
		log.SetLevel(loglevel)
	} else {
		log.SetLevel(log.WarnLevel)
	}
	return nil
}

func (pc PriceConfig) String() string {
	return fmt.Sprintf("{PriceConfig Base:%s Quote:%s Source:%s Wander:%v}",
		pc.Base, pc.Quote, pc.Source, pc.Wander)
}

func (ps SourceConfig) String() string {
	return fmt.Sprintf("{SourceConfig Name:%s URL:%s SleepReal:%ds}",
		ps.Name, ps.URL.String(), ps.SleepReal)
}

func (ps SourceConfig) IsCoinGecko() bool {
	return strings.Contains(ps.URL.Host, "coingecko.com")
}

func (ps SourceConfig) IsCoinMarketCap() bool {
	return strings.Contains(ps.URL.Host, "coinmarketcap.com")
}

func (ps SourceConfig) IsBitstamp() bool {
	return strings.Contains(ps.URL.Host, "bitstamp.net")
}
