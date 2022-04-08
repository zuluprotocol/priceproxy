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

func getPriceBitStamp(pricecfg config.PriceConfig, sourcecfg config.SourceConfig, client *http.Client, req *http.Request) (priceinfo PriceInfo, err error) {
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
		err = fmt.Errorf("bitstamp returned HTTP %d (%s)", resp.StatusCode, string(content))
		return priceinfo, err
	}

	var response bitstampResponse
	if err = json.Unmarshal(content, &response); err != nil {
		err = errors.Wrap(err, "failed to parse HTTP response as JSON")
		return priceinfo, err
	}

	if response.Last == "" {
		err = errors.New("bitstamp returned an empty Last price")
		return priceinfo, err
	}

	var p float64
	p, err = strconv.ParseFloat(response.Last, 64)
	if err != nil {
		return priceinfo, err
	}
	if p <= 0.0 {
		err = fmt.Errorf("bitstamp returned zero/negative price: %f", p)
		return priceinfo, err
	}
	t := time.Now().Round(0)
	priceinfo.LastUpdatedReal = t
	priceinfo.LastUpdatedWander = t
	priceinfo.Price = p * pricecfg.Factor
	return priceinfo, nil
}
