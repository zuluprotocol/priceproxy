package pricing

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"code.vegaprotocol.io/priceproxy/config"
	"github.com/pkg/errors"
)

type cmcStatusResponse struct {
	Timestamp    string `json:"timestamp"`
	ErrorCode    int    `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	Elapsed      int    `json:"elapsed"`
	CreditCount  int    `json:"credit_count"`
	Notice       string `json:"notice"`
}

type cmcQuoteResponse struct {
	Price            float64 `json:"price"`
	Volume24h        float64 `json:"volume_24h"`
	PercentChange1h  float64 `json:"percent_change_1h"`
	PercentChange24h float64 `json:"percent_change_24h"`
	PercentChange7d  float64 `json:"percent_change_7d"`
	MarketCap        float64 `json:"market_cap"`
	LastUpdated      string  `json:"last_updated"`
}

type cmcDataResponse struct {
	ID          int                         `json:"id"`
	Name        string                      `json:"name"`
	Symbol      string                      `json:"symbol"`
	Slug        string                      `json:"slug"`
	IsActive    int                         `json:"is_active"`
	LastUpdated string                      `json:"last_updated"`
	Quote       map[string]cmcQuoteResponse `json:"quote"`
}

type cmcResponse struct {
	Status cmcStatusResponse          `json:"status"`
	Data   map[string]cmcDataResponse `json:"data"`
}

func headersCoinmarketcap() (map[string][]string, error) {
	headers := make(map[string][]string, 1)

	fn := os.ExpandEnv("$HOME/coinmarketcap-apikey.txt")
	apiKey, err := ioutil.ReadFile(fn) // #nosec G304
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to read API key file %s", fn))
	}
	headers["X-CMC_PRO_API_KEY"] = []string{string(apiKey)}
	return headers, nil
}

func getPriceCoinmarketcap(pricecfg config.PriceConfig, sourcecfg config.SourceConfig, client *http.Client, req *http.Request) (priceinfo PriceInfo, err error) {
	if strings.HasPrefix(pricecfg.Quote, "XYZ") {
		// Inject a hidden price config, for competitions.
		pricecfg.Base = ""
		pricecfg.Quote = ""
	}
	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		err = errors.Wrap(err, "failed to perform HTTP request")
		return priceinfo, err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = errors.Wrap(err, "failed to read HTTP response body")
		return priceinfo, err
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("got HTTP %d (%s)", resp.StatusCode, string(content))
		return priceinfo, err
	}

	var response cmcResponse
	if err = json.Unmarshal(content, &response); err != nil {
		err = errors.Wrap(err, "failed to parse HTTP response as JSON")
		return priceinfo, err
	}

	data, ok := response.Data[pricecfg.Base]
	if !ok {
		err = fmt.Errorf("failed to find Base in response: %s", pricecfg.Base)
		return priceinfo, err
	}
	quote, ok := data.Quote[pricecfg.Quote]
	if !ok {
		err = fmt.Errorf("failed to find Quote in response: %s", pricecfg.Quote)
		return priceinfo, err
	}
	t := time.Now().Round(0)
	priceinfo.LastUpdatedReal = t
	priceinfo.LastUpdatedWander = t
	priceinfo.Price = quote.Price * pricecfg.Factor
	return priceinfo, nil
}
