package pricing

import (
	"fmt"
	"sync"
	"time"

	"github.com/vegaprotocol/priceproxy/config"
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
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/engine_mock.go -package mocks github.com/vegaprotocol/priceproxy/pricing Engine
type Engine interface {
	AddSource(sourcecfg config.SourceConfig) error
	GetSource(name string) (config.SourceConfig, error)
	GetSources() ([]config.SourceConfig, error)

	PriceList(source string) config.PriceList
	GetPrice(pricecfg config.PriceConfig) (PriceInfo, error)
	GetPrices() map[config.PriceConfig]PriceInfo
	UpdatePrice(pricecfg config.PriceConfig, newPrice PriceInfo)

	StartFetching() error
}

type priceBoard interface {
	PriceList(source string) config.PriceList
	UpdatePrice(pricecfg config.PriceConfig, newPrice PriceInfo)
}

type engine struct {
	priceList config.PriceList
	prices    map[config.PriceConfig]PriceInfo
	pricesMu  sync.RWMutex

	sources   map[string]config.SourceConfig
	sourcesMu sync.Mutex
}

// NewEngine creates a new pricing engine.
func NewEngine(prices config.PriceList) Engine {
	e := engine{
		priceList: prices,
		prices:    make(map[config.PriceConfig]PriceInfo),
		sources:   make(map[string]config.SourceConfig),
	}
	return &e
}

func (e *engine) AddSource(sourcecfg config.SourceConfig) error {
	if sourcecfg.SleepReal == 0 {
		return fmt.Errorf("invalid source config: sleepReal is zero")
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

func (e *engine) GetPrice(pricecfg config.PriceConfig) (PriceInfo, error) {
	e.pricesMu.RLock()
	defer e.pricesMu.RUnlock()

	pi, found := e.prices[pricecfg]
	if !found {
		return PriceInfo{}, fmt.Errorf("price not found: %s", pricecfg.String())
	}
	return pi, nil
}

func (e *engine) GetPrices() map[config.PriceConfig]PriceInfo {
	e.pricesMu.RLock()
	defer e.pricesMu.RUnlock()
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

func (e *engine) PriceList(source string) config.PriceList {
	return e.priceList.GetBySource(source)
}

func (e *engine) StartFetching() error {
	for _, sourceConfig := range e.sources {
		if sourceConfig.IsCoinGecko() {
			go coingeckoStartFetching(e, sourceConfig)
			continue
		}
		if sourceConfig.IsCoinMarketCap() {
			go coinmarketcapStartFetching(e, sourceConfig)
			continue
		}
		if sourceConfig.IsBitstamp() {
			go bitstampStartFetching(e, sourceConfig)
			continue
		}

		go httpStartFetching(e, sourceConfig)
	}

	return nil
}

func (pi PriceInfo) String() string {
	return fmt.Sprintf("{PriceInfo Price:%f LastUpdatedReal:%s LastUpdatedWander:%s}",
		pi.Price, pi.LastUpdatedReal.String(), pi.LastUpdatedWander.String())
}
