// Package config contains structures used in retrieving app configuration
// from disk.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"
)

type ServerConfig struct {
	Env       string
	Listen    string
	LogFormat string
	LogLevel  string
}

type PriceConfig struct {
	Source string  `yaml:"source"`
	Base   string  `yaml:"base"`
	Quote  string  `yaml:"quote"`
	Factor float64 `yaml:"factor"`
	Wander bool    `yaml:"wander"`
}

type SourceConfig struct {
	Name        string  `yaml:"name"`
	URL         url.URL `yaml:"url"`
	SleepReal   int     `yaml:"sleepReal"`
	SleepWander int     `yaml:"sleepWander"`
}

type Config struct {
	Server  *ServerConfig   `yaml:"server"`
	Prices  []*PriceConfig  `yaml:"prices"`
	Sources []*SourceConfig `yaml:"sources"`
}

var (
	ErrNil                  = errors.New("nil pointer")
	ErrEmptyConfigSection   = errors.New("empty config file section")
	ErrMissingConfigSection = errors.New("missing config file section")
	ErrInvalidValue         = errors.New("invalid value")
)

func CheckConfig(cfg *Config) error {
	if cfg == nil {
		return ErrNil
	}

	if cfg.Sources == nil {
		return fmt.Errorf("%s: %s", ErrMissingConfigSection.Error(), "sources")
	}
	if len(cfg.Sources) == 0 {
		return fmt.Errorf("%s: %s", ErrEmptyConfigSection.Error(), "sources")
	}
	for _, sourcecfg := range cfg.Sources {
		if sourcecfg.SleepReal == 0 {
			return fmt.Errorf("%s: sleepReal", ErrInvalidValue.Error())
		}
		if sourcecfg.SleepWander == 0 {
			return fmt.Errorf("%s: sleepWander", ErrInvalidValue.Error())
		}
	}

	if cfg.Prices == nil {
		return fmt.Errorf("%s: %s", ErrMissingConfigSection.Error(), "prices")
	}
	if len(cfg.Prices) == 0 {
		return fmt.Errorf("%s: %s", ErrEmptyConfigSection.Error(), "prices")
	}
	for _, pricecfg := range cfg.Prices {
		if pricecfg.Factor == 0 {
			return fmt.Errorf("%s: factor", ErrInvalidValue.Error())
		}
	}

	return nil
}

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
	return fmt.Sprintf("{SourceConfig Name:%s URL:%s SleepReal:%ds SleepWander:%ds}",
		ps.Name, ps.URL.String(), ps.SleepReal, ps.SleepWander)
}
