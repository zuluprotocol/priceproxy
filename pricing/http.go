package pricing

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vegaprotocol/priceproxy/config"
)

func httpStartFetching(
	board priceBoard,
	sourcecfg config.SourceConfig,
) {
	// TODO: implement when needed

	for {
		time.Sleep(time.Minute)

		log.WithFields(log.Fields{
			"sourceName": sourcecfg.Name,
		}).Errorf("You are trying to use the fetcher which is not implemented yet. Try to use different one: bitstamp, coingecko, coinmarketcap")
	}
}

// func urlWithBaseQuote(u url.URL, pricecfg config.PriceConfig) *url.URL {
// 	result := u
// 	result.Path = strings.Replace(result.Path, "{base}", pricecfg.Base, 1)
// 	result.Path = strings.Replace(result.Path, "{quote}", pricecfg.Quote, 1)
// 	result.RawQuery = strings.Replace(result.RawQuery, "{base}", pricecfg.Base, 1)
// 	result.RawQuery = strings.Replace(result.RawQuery, "{quote}", pricecfg.Quote, 1)
// 	return &result
// }
