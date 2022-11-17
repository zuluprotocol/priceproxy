package pricing

import (
	"net/url"
	"strings"
	"time"

	"github.com/vegaprotocol/priceproxy/config"
)

func httpStartFetching(
	board priceBoard,
	sourcecfg config.SourceConfig,
) {
	// TODO: implement when needed

	for {
		time.Sleep(time.Minute)
	}
}

func urlWithBaseQuote(u url.URL, pricecfg config.PriceConfig) *url.URL {
	result := u
	result.Path = strings.Replace(result.Path, "{base}", pricecfg.Base, 1)
	result.Path = strings.Replace(result.Path, "{quote}", pricecfg.Quote, 1)
	result.RawQuery = strings.Replace(result.RawQuery, "{base}", pricecfg.Base, 1)
	result.RawQuery = strings.Replace(result.RawQuery, "{quote}", pricecfg.Quote, 1)
	return &result
}
