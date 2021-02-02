package pricing

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"code.vegaprotocol.io/priceproxy/config"
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
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("bitstamp returned HTTP %d", resp.StatusCode)
		return
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = ErrServerResponseReadFail
		return
	}

	var response bitstampResponse
	if err = json.Unmarshal(content, &response); err != nil {
		return
	}

	if response.Last == "" {
		err = errors.New("bitstamp returned an empty Last price")
		return
	}

	t := time.Now().Round(0)
	priceinfo.LastUpdatedReal = t
	priceinfo.LastUpdatedWander = t
	priceinfo.Price, err = strconv.ParseFloat(response.Last, 64)
	priceinfo.Price *= pricecfg.Factor
	return
}
