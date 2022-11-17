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
	coingeckoSourceName = "coingecko"
	supportedQuotes     = []string{"ETH", "EUR", "USD", "BTC", "DAI"}
)

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

		for _, price := range board.PriceList(sourcecfg.Name) {
			priceUpdated := false
		CoinGeckoLoop:
			for coingeckoBase, coingeckoData := range *prices {
				if price.Base == coingeckoBase {
					var fetchedPrice float64

					switch strings.ToUpper(price.Quote) {
					case "ETH":
						fetchedPrice = coingeckoData.ETH
					case "BTC":
						fetchedPrice = coingeckoData.BTC
					case "USD":
						fetchedPrice = coingeckoData.USD
					case "EUR":
						fetchedPrice = coingeckoData.EUR
					case "DAI":
						fetchedPrice = coingeckoData.DAI
					default:
						log.WithFields(log.Fields{
							"sourceName":     coingeckoSourceName,
							"base":           price.Base,
							"quote":          price.Quote,
							"quote_override": price.QuoteOverride,
						}).Errorf("price quote is invalid, got %s, expecting one of %v", price.Quote, supportedQuotes)
						break CoinGeckoLoop
					}

					if fetchedPrice == 0 {
						log.WithFields(log.Fields{
							"sourceName":     coingeckoSourceName,
							"base":           price.Base,
							"quote":          price.Quote,
							"quote_override": price.QuoteOverride,
						}).Warnf("fetched price in the quote current is 0, consider selecting different quote and overwrite it with the `quote_override` parameter")
					}

					board.UpdatePrice(
						price,
						PriceInfo{
							Price:             fetchedPrice,
							LastUpdatedReal:   time.Unix(int64(coingeckoData.LastUpdatedAt), 0),
							LastUpdatedWander: time.Now().Round(0),
						},
					)
					priceUpdated = true
				}
			}

			if !priceUpdated {
				log.WithFields(log.Fields{
					"sourceName":     coingeckoSourceName,
					"base":           price.Base,
					"quote":          price.Quote,
					"quote_override": price.QuoteOverride,
				}).Errorf("price not found in the coingecko API")
			}
		}
	}
}

type coingeckoFetchData map[string]struct {
	USD           float64 `json:"usd"`
	EUR           float64 `json:"eur"`
	BTC           float64 `json:"btc"`
	ETH           float64 `json:"eth"`
	DAI           float64 `json:"dai"`
	LastUpdatedAt uint64  `json:"last_updated_at"`
}

func coingeckoSingleFetch(url string) (*coingeckoFetchData, error) {
	resp, err := http.Get(url) // nolint:noctx
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
