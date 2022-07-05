package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"code.vegaprotocol.io/priceproxy/config"
	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

var (
	coingeckoExtraPairs = []config.PriceConfig{}
	coingeckoSourceName = "coingecko"
)

func coingeckoAddExtraPriceConfig(priceconfig config.PriceConfig) error {
	coingeckoExtraPairs = append(coingeckoExtraPairs, priceconfig)
	return nil
}

type priceBoard interface {
	UpdatePrice(pricecfg config.PriceConfig, newPrice PriceInfo)
}

func coingeckoStartFetching(
	board priceBoard,
	sourcecfg config.SourceConfig,
) {
	var (
		fetchURL        = sourcecfg.URL.String()
		oneRequestEvery = time.Duration(sourcecfg.SleepReal) * time.Second
		rateLimiter     = rate.NewLimiter(rate.Every(oneRequestEvery), 1)
		ctx             = context.Background()
		err             error
	)
	log.WithFields(log.Fields{
		"sourceName":        coingeckoSourceName,
		"URL":               fetchURL,
		"rateLimitDuration": oneRequestEvery,
	}).Infof("Starting Coingecko Fetching\n")
	for {
		if err = rateLimiter.Wait(ctx); err != nil {
			log.WithFields(log.Fields{
				"error":             err.Error(),
				"sourceName":        coingeckoSourceName,
				"URL":               fetchURL,
				"rateLimitDuration": oneRequestEvery,
			}).Errorln("Rate Limiter Failed. Falling back to Sleep.")
			// fallback
			time.Sleep(oneRequestEvery)
		}

		prices, err := coingeckoSingleFetch(fetchURL)
		if err != nil {
			log.WithFields(log.Fields{
				"error":             err.Error(),
				"sourceName":        coingeckoSourceName,
				"URL":               fetchURL,
				"rateLimitDuration": oneRequestEvery,
			}).Errorf("Retry in %d sec.\n", oneRequestEvery)
			continue
		}

		for base, data := range *prices {
			board.UpdatePrice(
				config.PriceConfig{
					Source: coingeckoSourceName,
					Base:   base,
					Quote:  "ETH",
					Factor: 1.0,
					Wander: true,
				},
				PriceInfo{
					Price:             data.ETH,
					LastUpdatedReal:   time.Unix(int64(data.LastUpdatedAt), 0),
					LastUpdatedWander: time.Now().Round(0),
				},
			)
		}

		for _, extraPair := range coingeckoExtraPairs {
			base, ok := (*prices)[extraPair.Base]
			if !ok {
				log.WithFields(log.Fields{
					"sourceName":        coingeckoSourceName,
					"URL":               fetchURL,
					"rateLimitDuration": oneRequestEvery,
				}).Errorf("Failed to get base %s for extra pair %v\n", extraPair.Base, extraPair)
				continue
			}
			var price float64
			if strings.EqualFold(extraPair.Quote, "EUR") {
				price = base.EUR
			} else if strings.EqualFold(extraPair.Quote, "USD") {
				price = base.USD
			} else if strings.EqualFold(extraPair.Quote, "BTC") {
				price = base.BTC
			} else {
				quote, ok := (*prices)[extraPair.Quote]
				if !ok {
					log.WithFields(log.Fields{
						"sourceName":        coingeckoSourceName,
						"URL":               fetchURL,
						"rateLimitDuration": oneRequestEvery,
					}).Errorf("Failed to get quote %s for extra pair %v\n", extraPair.Source, extraPair)
					continue
				}
				price = base.USD / quote.USD
			}
			board.UpdatePrice(
				extraPair,
				PriceInfo{
					Price:             price,
					LastUpdatedReal:   time.Unix(int64(base.LastUpdatedAt), 0),
					LastUpdatedWander: time.Now().Round(0),
				},
			)
		}
	}
}

type coingeckoFetchData map[string]struct {
	USD           float64 `json:"usd"`
	EUR           float64 `json:"eur"`
	BTC           float64 `json:"btc"`
	ETH           float64 `json:"eth"`
	LastUpdatedAt uint64  `json:"last_updated_at"`
}

func coingeckoSingleFetch(url string) (*coingeckoFetchData, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get coingecko data, %w", err)
	}
	defer resp.Body.Close()
	var prices coingeckoFetchData
	if err = json.NewDecoder(resp.Body).Decode(&prices); err != nil {
		return nil, fmt.Errorf("failed to parse coingecko data, %w", err)
	}
	return &prices, nil
}
