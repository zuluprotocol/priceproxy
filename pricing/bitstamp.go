package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vegaprotocol/priceproxy/config"
	"golang.org/x/time/rate"
)

func bitstampStartFetching(
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

		prices, err := bitstampSingleFetch(fetchURL)
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
			var (
				fetchedTimestamp time.Time = time.Now()
				fetchedPrice     float64
			)

			if currency := prices.Currency(price.Base, price.Quote); currency != nil {
				fetchedTimestamp = currency.UnixTimestamp()
				fetchedPrice = currency.Price()
			}

			if fetchedPrice == 0 {
				log.WithFields(log.Fields{
					"sourceName":     sourcecfg.Name,
					"base":           price.Base,
					"quote":          price.Quote,
					"quote_override": price.QuoteOverride,
				}).Debug("Quote/Base rate not found directly, trying conversion")

				if currency := prices.Convert(price.Base, price.Quote); currency != nil {
					fetchedTimestamp = currency.UnixTimestamp()
					fetchedPrice = currency.Price()
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
					LastUpdatedReal:   fetchedTimestamp,
					LastUpdatedWander: time.Now().Round(0),
				},
			)
		}
	}
}

type bitstampCurrencyData struct {
	Timestamp string `json:"timestamp"`
	Last      string `json:"last"`
	Pair      string `json:"pair"`
}

type bitstampFetchData []bitstampCurrencyData

func (fd bitstampCurrencyData) Base() string {
	pairSlice := strings.Split(fd.Pair, "/")

	return pairSlice[0]
}

func (fd bitstampCurrencyData) UnixTimestamp() time.Time {
	timestamp, err := strconv.ParseInt(fd.Timestamp, 10, 64)
	if err != nil {
		return time.Now()
	}

	return time.Unix(timestamp, 0)
}

func (fd bitstampCurrencyData) Price() float64 {
	price, err := strconv.ParseFloat(fd.Timestamp, 64)
	if err != nil {
		return 0.0
	}

	return price
}

func (fd bitstampCurrencyData) Quote() string {
	pairSlice := strings.Split(fd.Pair, "/")

	if len(pairSlice) < 2 {
		return pairSlice[0]
	}
	return pairSlice[1]
}

func (fd bitstampFetchData) Currency(base, quote string) *bitstampCurrencyData {
	for _, currency := range fd {
		if strings.EqualFold(currency.Base(), base) && strings.EqualFold(currency.Quote(), quote) {
			return &currency
		}
	}

	return nil
}

func (fd bitstampFetchData) Convert(base, quote string) *bitstampCurrencyData {
	basePrices := []bitstampCurrencyData{}
	quotePrices := []bitstampCurrencyData{}

	if currency := fd.Currency(base, quote); currency != nil {
		return currency
	}

	for _, price := range fd {
		if strings.EqualFold(price.Base(), base) {
			basePrices = append(basePrices, price)
		}

		if strings.EqualFold(price.Base(), quote) {
			quotePrices = append(quotePrices, price)
		}
	}

	// find common currency and calculate value
	for _, basePrice := range basePrices {
		for _, quotePrice := range quotePrices {
			if strings.EqualFold(basePrice.Quote(), quotePrice.Quote()) {
				return &bitstampCurrencyData{
					Timestamp: basePrice.Timestamp,
					Pair:      fmt.Sprintf("%s/%s", base, quote),
					Last:      fmt.Sprintf("%f", (basePrice.Price() / quotePrice.Price())),
				}
			}
		}
	}

	return nil
}

func bitstampSingleFetch(url string) (bitstampFetchData, error) {
	resp, err := http.Get(url) // nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("failed to get bitstamp data, %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get bitstamp data: expected status 200, got %d", resp.StatusCode)
	}

	defer resp.Body.Close()
	var prices bitstampFetchData
	if err = json.NewDecoder(resp.Body).Decode(&prices); err != nil {
		return nil, fmt.Errorf("failed to parse bitstamp data, %w", err)
	}
	return prices, nil
}

// https://www.bitstamp.net/api/v2/ticker/
