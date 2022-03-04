// Command priceproxy runs a REST webserver that serves prices that are periodically updated from other sources.
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jinzhu/configor"
	log "github.com/sirupsen/logrus"

	"code.vegaprotocol.io/priceproxy/config"
	"code.vegaprotocol.io/priceproxy/service"
)

var (
	// Version is set at build time using: -ldflags "-X main.Version=someversion"
	Version = "no_version_set"

	// VersionHash is set at build time using: -ldflags "-X main.VersionHash=somehash"
	VersionHash = "no_hash_set"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	var configName string
	var configVersion bool
	flag.StringVar(&configName, "config", "", "Configuration YAML file")
	flag.BoolVar(&configVersion, "version", false, "Show version")
	flag.Parse()

	if configVersion {
		fmt.Printf("version %v (%v)\n", Version, VersionHash)
		return
	}

	var cfg config.Config
	err := configor.Load(&cfg, configName)
	// https://github.com/jinzhu/configor/issues/40
	if err != nil && !strings.Contains(err.Error(), "should be struct") {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Fatal("Failed to read config")
	}
	err = config.CheckConfig(&cfg)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Fatal("Config checks failed")
	}

	err = config.ConfigureLogging(cfg.Server)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Fatal("Failed to load config")
	}

	log.WithFields(log.Fields{
		"version": Version,
		"hash":    VersionHash,
	}).Info("Version")

	var s *service.Service
	s, err = service.NewService(cfg)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Fatal("Failed to create service")
	}

	go func() {
		err := s.Start()
		if err != nil {
			log.WithFields(log.Fields{
				"listen": cfg.Server.Listen,
				"extra":  err.Error(),
			}).Fatal("Could not listen")
		}
	}()
	c := make(chan os.Signal, 2)
	signal.Notify(c, syscall.SIGINT)
	signal.Notify(c, syscall.SIGTERM)
	<-c
	s.Stop()
}
