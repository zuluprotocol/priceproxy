package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vegaprotocol/priceproxy/config"
	"golang.org/x/time/rate"
)

func coinmarketcapStartFetching(
	board priceBoard,
	sourcecfg config.SourceConfig,
) {
	var (
		fetchURL        = sourcecfg.URL
		oneRequestEvery = time.Duration(sourcecfg.SleepReal) * time.Second
		rateLimiter     = rate.NewLimiter(rate.Every(oneRequestEvery), 1)
		ctx             = context.Background()
		err             error
	)

	apiKey := ""
	if sourcecfg.AuthKeyEnvName != "" {
		apiKey = os.Getenv(sourcecfg.AuthKeyEnvName)
	}

	if apiKey == "" {
		log.WithFields(log.Fields{
			"sourceName":     coingeckoSourceName,
			"URL":            sourcecfg.URL,
			"AuthKeyEnvName": sourcecfg.AuthKeyEnvName,
		}).Warnf("The API key is empty. Use the `auth_key_env_name` config for the source and export corresponding environment name")
	}

	fetchURLQuery := fetchURL.Query()
	fetchURLQuery.Add("CMC_PRO_API_KEY", apiKey)
	fetchURL.RawQuery = fetchURLQuery.Encode()

	for {
		if err = rateLimiter.Wait(ctx); err != nil {
			log.WithFields(log.Fields{
				"error":             err.Error(),
				"sourceName":        sourcecfg.Name,
				"URL":               sourcecfg.URL.String(),
				"rateLimitDuration": oneRequestEvery,
			}).Errorln("Rate Limiter Failed. Falling back to Sleep.")
			// fallback
			time.Sleep(oneRequestEvery)
		}

		coinmarketcapData, err := coinmarketcapSingleFetch(fetchURL.String())
		if err != nil {
			log.WithFields(log.Fields{
				"error":             err.Error(),
				"sourceName":        sourcecfg.Name,
				"URL":               sourcecfg.URL.String(),
				"rateLimitDuration": oneRequestEvery,
			}).Errorln("failed to get trading data.")
		}

		for _, price := range board.PriceList(sourcecfg.Name) {
			fetchedCurrency := coinmarketcapData.GetCurrency(price.Base)
			if fetchedCurrency == nil {
				log.WithFields(log.Fields{
					"sourceName":     coingeckoSourceName,
					"base":           price.Base,
					"quote":          price.Quote,
					"quote_override": price.QuoteOverride,
				}).Errorln("price not returned from the API")
				continue
			}

			fetchedQuote := fetchedCurrency.QuoteByName(price.Quote)
			fetchedPrice := 0.0
			fetchedLastUpdate := fetchedCurrency.LastUpdated
			if fetchedQuote == nil {
				log.WithFields(log.Fields{
					"sourceName":     coingeckoSourceName,
					"base":           price.Base,
					"quote":          price.Quote,
					"quote_override": price.QuoteOverride,
				}).Warnf("collected price in the quote current is 0, consider selecting different quote and overwrite it with the `quote_override` parameter")
			} else {
				fetchedPrice = fetchedQuote.Price
				fetchedLastUpdate = fetchedQuote.LastUpdated
			}

			parsedTime, err := time.Parse(time.RFC3339, fetchedLastUpdate)
			if err != nil {
				log.WithFields(log.Fields{
					"error":             err.Error(),
					"sourceName":        coingeckoSourceName,
					"base":              price.Base,
					"quote":             price.Quote,
					"quote_override":    price.QuoteOverride,
					"last_updated_time": fetchedLastUpdate,
				}).Warnf("cannot parse fetched last_updated time with the ISO8601 format")
			}

			if fetchedPrice == 0 {
				log.WithFields(log.Fields{
					"sourceName":     coingeckoSourceName,
					"base":           price.Base,
					"quote":          price.Quote,
					"quote_override": price.QuoteOverride,
				}).Debug("Quote/Base rate not found directly, trying conversion")

				fetchedPrice = coinmarketcapData.ConvertPrice(price.Base, price.Quote)
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
					LastUpdatedReal:   parsedTime,
					LastUpdatedWander: time.Now().Round(0),
				},
			)
		}
	}
}

type coinmarketcapQuoteData struct {
	Price       float64 `json:"price"`
	LastUpdated string  `json:"last_updated"`
}

type coinmarketcapCurrencyData struct {
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	Slug        string `json:"slug"`
	LastUpdated string `json:"last_updated"`

	Quote map[string]coinmarketcapQuoteData `json:"quote"`
}

type coinmarketcapFetchData struct {
	Data []coinmarketcapCurrencyData `json:"data"`
}

func (data coinmarketcapFetchData) GetCurrency(name string) *coinmarketcapCurrencyData {
	for _, currencyData := range data.Data {
		if strings.EqualFold(currencyData.Name, name) || strings.EqualFold(currencyData.Slug, name) || strings.EqualFold(currencyData.Symbol, name) {
			return &currencyData
		}
	}

	return nil
}

func (currency coinmarketcapCurrencyData) QuoteByName(name string) *coinmarketcapQuoteData {
	for qName, qData := range currency.Quote {
		if strings.EqualFold(qName, name) {
			return &qData
		}
	}
	return nil
}

func (data coinmarketcapFetchData) ConvertPrice(base, quote string) float64 {
	baseCurrency := data.GetCurrency(base)
	quoteCurrency := data.GetCurrency(quote)

	if baseCurrency == nil || quoteCurrency == nil {
		return 0.0
	}

	for qName, qData := range baseCurrency.Quote {
		// try luck with direct conversion
		if strings.EqualFold(qName, baseCurrency.Name) || strings.EqualFold(qName, baseCurrency.Slug) || strings.EqualFold(qName, baseCurrency.Symbol) {
			return qData.Price
		}

		// conversion by common currency(e.g USD)
		if commonCurrency := quoteCurrency.QuoteByName(qName); commonCurrency != nil {
			return qData.Price / commonCurrency.Price
		}

		// todo: search by ConvertPrice(quote, qName), but may create deadlock...
		// 	    so we have to impleent timeout + error mechanism here
	}

	return data.ConvertPrice(quote, base)
}

func coinmarketcapSingleFetch(url string) (*coinmarketcapFetchData, error) {
	resp, err := http.Get(url) // nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("failed to get coinmarketcap data, %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get coinmarketcap data: expected status 200, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var prices coinmarketcapFetchData
	if err = json.NewDecoder(resp.Body).Decode(&prices); err != nil {
		return nil, fmt.Errorf("failed to parse coinmarketcap data, %w", err)
	}
	return &prices, nil
}
