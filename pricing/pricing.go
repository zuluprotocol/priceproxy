package pricing

import (
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"code.vegaprotocol.io/priceproxy/config"
	log "github.com/sirupsen/logrus"
)

const minPrice = 0.00001

// PriceInfo describes a price from a source.
// The price may be a real updated from an upstream source, or one that has been wandered.
// The LastUpdated timstamps indicate when the price was last fetched for real and when (if at all) it was last wandered.
type PriceInfo struct {
	Price             float64
	LastUpdatedReal   time.Time
	LastUpdatedWander time.Time
}

// Engine is the source of price information from multiple external/internal/fake sources.
//go:generate go run github.com/golang/mock/mockgen -destination mocks/engine_mock.go -package mocks code.vegaprotocol.io/priceproxy/pricing Engine
type Engine interface {
	AddSource(sourcecfg config.SourceConfig) error
	GetSource(name string) (config.SourceConfig, error)
	GetSources() ([]config.SourceConfig, error)

	AddPrice(pricecfg config.PriceConfig) (pi PriceInfo, err error)
	GetPrice(pricecfg config.PriceConfig) (PriceInfo, error)
	GetPrices() map[config.PriceConfig]PriceInfo
	UpdatePrice(pricecfg config.PriceConfig, newPrice PriceInfo)
}

type engine struct {
	prices   map[config.PriceConfig]PriceInfo
	pricesMu sync.Mutex

	sources   map[string]config.SourceConfig
	sourcesMu sync.Mutex
}

type fetchPriceFunc func(pricecfg config.PriceConfig, sourcecfg config.SourceConfig, client *http.Client, req *http.Request) (PriceInfo, error)

// NewEngine creates a new pricing engine
func NewEngine() Engine {
	e := engine{
		prices:  make(map[config.PriceConfig]PriceInfo),
		sources: make(map[string]config.SourceConfig),
	}
	return &e
}

func (e *engine) AddSource(sourcecfg config.SourceConfig) error {
	if sourcecfg.SleepReal == 0 {
		return fmt.Errorf("invalid source config: sleepReal is zero")
	}
	if sourcecfg.SleepWander == 0 {
		return fmt.Errorf("invalid source config: sleepWander is zero")
	}

	e.sourcesMu.Lock()
	defer e.sourcesMu.Unlock()

	_, found := e.sources[sourcecfg.Name]
	if found {
		return fmt.Errorf("source already exists: %s", sourcecfg.Name)
	}

	e.sources[sourcecfg.Name] = sourcecfg
	return nil
}

func (e *engine) GetSource(name string) (config.SourceConfig, error) {
	e.sourcesMu.Lock()
	defer e.sourcesMu.Unlock()

	source, found := e.sources[name]

	if !found {
		return config.SourceConfig{}, fmt.Errorf("price source not found: %s", name)
	}
	return source, nil
}

func (e *engine) GetSources() ([]config.SourceConfig, error) {
	e.sourcesMu.Lock()
	defer e.sourcesMu.Unlock()

	response := make([]config.SourceConfig, len(e.sources))
	i := 0
	for _, source := range e.sources {
		response[i] = source
		i++
	}
	return response, nil
}

func (e *engine) AddPrice(pricecfg config.PriceConfig) (pi PriceInfo, err error) {
	e.pricesMu.Lock()
	_, found := e.prices[pricecfg]
	e.pricesMu.Unlock()
	if found {
		err = fmt.Errorf("price already exists: %s", pricecfg.String())
		return
	}

	source, err := e.GetSource(pricecfg.Source)
	if err != nil {
		return
	}

	headers := map[string][]string{}

	if source.Name == "bitstamp" {
		go e.stream(pricecfg, source, nil, headers, getPriceBitStamp)
	} else if strings.HasPrefix(source.Name, "ftx-") {
		go e.stream(pricecfg, source, nil, headers, getPriceFTX)
	} else {
		err = fmt.Errorf("no source for %s", source.Name)
		return
	}

	sublog := log.WithFields(log.Fields{
		"priceConfig": pricecfg.String(),
	})

	sublog.Debug("Waiting for first price")
	s := 10 // milliseconds
	for {
		pi, err = e.GetPrice(pricecfg)
		if err != nil {
			sublog.WithFields(log.Fields{"err": err.Error()}).Debug("Waiting for first price")
		} else {
			sublog.WithFields(log.Fields{"price": pi.Price}).Debug("Got first price")
			if pi.Price > 0.0 {
				break
			}
		}
		time.Sleep(time.Duration(s) * time.Millisecond)
		s *= 2
	}
	return
}

func (e *engine) GetPrice(pricecfg config.PriceConfig) (PriceInfo, error) {
	e.pricesMu.Lock()
	defer e.pricesMu.Unlock()

	pi, found := e.prices[pricecfg]
	if !found {
		return PriceInfo{}, fmt.Errorf("price not found: %s", pricecfg.String())
	}
	return pi, nil
}

func (e *engine) GetPrices() map[config.PriceConfig]PriceInfo {
	e.pricesMu.Lock()
	defer e.pricesMu.Unlock()
	results := map[config.PriceConfig]PriceInfo{}

	for k, v := range e.prices {
		results[k] = v
	}
	return results
}

func (e *engine) UpdatePrice(pricecfg config.PriceConfig, newPrice PriceInfo) {
	e.pricesMu.Lock()
	e.prices[pricecfg] = newPrice
	e.pricesMu.Unlock()
}

func (e *engine) stream(pricecfg config.PriceConfig, sourcecfg config.SourceConfig, u *url.URL, headers map[string][]string, fetchPrice fetchPriceFunc) {
	sublog := log.WithFields(log.Fields{
		"base":       pricecfg.Base,
		"quote":      pricecfg.Quote,
		"source":     sourcecfg.Name,
		"source-url": u.String(),
	})

	annualisedSleepReal := float64(sourcecfg.SleepReal) / 365.25 / 86400.0
	kappa := 1.0 / annualisedSleepReal

	client := http.Client{}
	if u == nil {
		u = urlWithBaseQuote(sourcecfg.URL, pricecfg)
	}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		sublog.WithFields(log.Fields{
			"error": err.Error()},
		).Fatal("Failed to create HTTP request")
	}
	for headerName, headerValueList := range headers {
		for _, headerValue := range headerValueList {
			req.Header.Add(headerName, headerValue)
		}
	}

	var rpi, realPriceInfo, priceInfo PriceInfo

	// Get price for the first time
	for err != nil || realPriceInfo.Price == 0 {
		realPriceInfo, err = fetchPrice(pricecfg, sourcecfg, &client, req)
		if err != nil {
			sublog.WithFields(log.Fields{
				"error": err.Error(),
			}).Debug("Failed to fetch real price for the first time")
		}
		time.Sleep(time.Duration(sourcecfg.SleepWander) * time.Second)
	}
	e.UpdatePrice(pricecfg, realPriceInfo)
	sublog.WithFields(log.Fields{
		"realPriceInfo": realPriceInfo.String(),
	}).Debug("Fetched real price for the first time")

	for {
		cutoff := time.Now().Round(0).Add(time.Duration(-sourcecfg.SleepReal) * time.Second)
		if realPriceInfo.LastUpdatedReal.Before(cutoff) {
			rpi, err = fetchPrice(pricecfg, sourcecfg, &client, req)
			if err == nil {
				realPriceInfo = rpi
				e.UpdatePrice(pricecfg, realPriceInfo)
				sublog.WithFields(log.Fields{
					"realPriceInfo": realPriceInfo.String(),
				}).Debug("Fetched real price")
			} else {
				sublog.WithFields(log.Fields{
					"error": err.Error(),
				}).Warning("Failed to fetch real price")
			}
		}

		if pricecfg.Wander {
			priceInfo, err = e.GetPrice(pricecfg)
			if err == nil {
				// make the price wander
				sigma := 1.0
				wander := kappa*(realPriceInfo.Price-priceInfo.Price)*annualisedSleepReal + sigma*priceInfo.Price*math.Sqrt(annualisedSleepReal)*rand.NormFloat64()
				priceInfo.Price += wander
				if priceInfo.Price < minPrice {
					priceInfo.Price = minPrice
				}
				priceInfo.LastUpdatedWander = time.Now().Round(0)
				e.UpdatePrice(pricecfg, priceInfo)
				sublog.WithFields(log.Fields{
					"kappa":    kappa,
					"sigma":    sigma,
					"wander":   wander,
					"newPrice": priceInfo.String(),
				}).Debug("Wandered price")
			} else {
				sublog.WithFields(log.Fields{
					"error": err.Error(),
				}).Warning("Failed to fetch price")
			}
		}
		time.Sleep(time.Duration(sourcecfg.SleepWander) * time.Second)
	}
}

func (pi PriceInfo) String() string {
	return fmt.Sprintf("{PriceInfo Price:%f LastUpdatedReal:%s LastUpdatedWander:%s}",
		pi.Price, pi.LastUpdatedReal.String(), pi.LastUpdatedWander.String())
}

func urlWithBaseQuote(u url.URL, pricecfg config.PriceConfig) *url.URL {
	result := u
	result.Path = strings.Replace(result.Path, "{base}", pricecfg.Base, 1)
	result.Path = strings.Replace(result.Path, "{quote}", pricecfg.Quote, 1)
	result.RawQuery = strings.Replace(result.RawQuery, "{base}", pricecfg.Base, 1)
	result.RawQuery = strings.Replace(result.RawQuery, "{quote}", pricecfg.Quote, 1)
	return &result
}
