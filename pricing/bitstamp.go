package pricing

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"code.vegaprotocol.io/priceproxy/config"
	"github.com/pkg/errors"
)

type bitstampResponse struct {
	High      string `json:"high"`
	Last      string `json:"last"`
	Timestamp string `json:"timestamp"`
	Bid       string `json:"bid"`
	Vwap      string `json:"vwap"`
	Volume    string `json:"volume"`
	Low       string `json:"low"`
	Ask       string `json:"ask"`
	Open      string `json:"open"`
}

var _ fetchPriceFunc = getPriceBitStamp

func getPriceBitStamp(pricecfg config.PriceConfig, sourcecfg config.SourceConfig, client *http.Client, req *http.Request) (PriceInfo, error) {
	resp, err := client.Do(req)
	if err != nil {
		return PriceInfo{}, errors.Wrap(err, "failed to perform HTTP request")
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return PriceInfo{}, errors.Wrap(err, "failed to read HTTP response body")
	}

	if resp.StatusCode != http.StatusOK {
		return PriceInfo{}, fmt.Errorf("bitstamp returned HTTP %d (%s)", resp.StatusCode, string(content))
	}

	var response bitstampResponse
	if err = json.Unmarshal(content, &response); err != nil {
		return PriceInfo{}, errors.Wrap(err, "failed to parse HTTP response as JSON")
	}

	if response.Last == "" {
		return PriceInfo{}, errors.New("bitstamp returned an empty Last price")
	}

	var p float64
	p, err = strconv.ParseFloat(response.Last, 64)
	if err != nil {
		return PriceInfo{}, err
	}
	if p <= 0.0 {
		return PriceInfo{}, fmt.Errorf("bitstamp returned zero/negative price: %f", p)
	}
	t := time.Now().Round(0)
	return PriceInfo{
		LastUpdatedReal:   t,
		LastUpdatedWander: t,
		Price:             p * pricecfg.Factor,
	}, nil
}
