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

var supportedQuotes = []string{"ETH", "EUR", "USD", "BTC", "DAI"}

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
		"sourceName":        sourcecfg.Name,
		"URL":               fetchURL,
		"rateLimitDuration": oneRequestEvery,
	}).Infof("Starting Coingecko Fetching\n")

	for {
		if err = rateLimiter.Wait(ctx); err != nil {
			log.WithFields(log.Fields{
				"error":             err.Error(),
				"sourceName":        sourcecfg.Name,
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
				"sourceName":        sourcecfg.Name,
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
							"sourceName":     sourcecfg.Name,
							"base":           price.Base,
							"quote":          price.Quote,
							"quote_override": price.QuoteOverride,
						}).Errorf("price quote is invalid, got %s, expecting one of %v", price.Quote, supportedQuotes)
						break CoinGeckoLoop
					}

					if fetchedPrice == 0 {
						log.WithFields(log.Fields{
							"sourceName":     sourcecfg.Name,
							"base":           price.Base,
							"quote":          price.Quote,
							"quote_override": price.QuoteOverride,
						}).Debug("Quote/Base rate not found directly, trying conversion")

						if convertedPrice := prices.Convert(price.Base, price.Quote); convertedPrice > 0 {
							fetchedPrice = convertedPrice
						}
					}

					if fetchedPrice == 0 {
						log.WithFields(log.Fields{
							"sourceName":     sourcecfg.Name,
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
					"sourceName":     sourcecfg.Name,
					"base":           price.Base,
					"quote":          price.Quote,
					"quote_override": price.QuoteOverride,
				}).Errorf("price not found in the coingecko API")
			}
		}
	}
}

type coingeckoCurrencyData struct {
	USD           float64 `json:"usd"`
	EUR           float64 `json:"eur"`
	BTC           float64 `json:"btc"`
	ETH           float64 `json:"eth"`
	DAI           float64 `json:"dai"`
	LastUpdatedAt uint64  `json:"last_updated_at"`
}

type coingeckoFetchData map[string]coingeckoCurrencyData

func (fd coingeckoFetchData) Convert(base, quote string) float64 {
	var baseData *coingeckoCurrencyData
	var quoteData *coingeckoCurrencyData
	for name, data := range fd {
		data := data
		if strings.EqualFold(base, name) {
			baseData = &data
		}

		if strings.EqualFold(quote, name) {
			quoteData = &data
		}

		if baseData != nil && quoteData != nil {
			break
		}
	}

	if baseData == nil || quoteData == nil {
		return 0.0
	}

	if baseData.USD > 0 && quoteData.USD > 0 {
		return baseData.USD / quoteData.USD
	}

	if baseData.EUR > 0 && quoteData.EUR > 0 {
		return baseData.EUR / quoteData.EUR
	}

	if baseData.DAI > 0 && quoteData.DAI > 0 {
		return baseData.DAI / quoteData.DAI
	}

	if baseData.BTC > 0 && quoteData.BTC > 0 {
		return baseData.BTC / quoteData.BTC
	}

	if baseData.ETH > 0 && quoteData.ETH > 0 {
		return baseData.ETH / quoteData.ETH
	}

	return 0.0
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
