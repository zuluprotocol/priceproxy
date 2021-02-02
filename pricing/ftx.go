package pricing

// Source: https://ftx.com/
// Docs: https://docs.ftx.com/

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"code.vegaprotocol.io/priceproxy/config"
)

type ftxResultResponse struct {
	Ask                   float64 `json:"ask"`
	BaseCurrency          string  `json:"baseCurrency"`
	Bid                   float64 `json:"symbol"`
	Change1h              float64 `json:"change1h"`
	Change24h             float64 `json:"change24h"`
	ChangeBod             float64 `json:"changeBod"`
	Enabled               bool    `json:"enabled"`
	HighLeverageFeeExempt bool    `json:"highLeverageFeeExempt"`
	Last                  float64 `json:"last"`
	MinProvideSize        float64 `json:"minProvideSize"`
	Name                  string  `json:"name"`
	PostOnly              bool    `json:"postOnly"`
	Price                 float64 `json:"price"`
	PriceIncrement        float64 `json:"priceIncrement"`
	QuoteCurrency         string  `json:"quoteCurrency"`
	QuoteVolume24h        float64 `json:"quoteVolume24h"`
	Restricted            bool    `json:"restricted"`
	SizeIncrement         float64 `json:"sizeIncrement"`
	Type                  string  `json:"type"`
	Underlying            string  `json:"underlying"`
	VolumeUSD24h          float64 `json:"volumeUsd24h"`
}

type ftxResponse struct {
	Result  ftxResultResponse `json:"result"`
	Success bool              `json:"success"`
}

var _ fetchPriceFunc = getPriceFTX

func getPriceFTX(pricecfg config.PriceConfig, sourcecfg config.SourceConfig, client *http.Client, req *http.Request) (priceinfo PriceInfo, err error) {
	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("ftx.com returned HTTP %d", resp.StatusCode)
		return
	}

	var response ftxResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return
	}
	if !response.Success {
		err = fmt.Errorf("ftx.com returned success=false")
		return
	}

	priceinfo.Price = response.Result.Price
	if priceinfo.Price == 0 {
		// Sometimes null/zero.
		priceinfo.Price = response.Result.Last
	}

	if priceinfo.Price == 0 {
		err = fmt.Errorf("ftx.com returned zero for Price and Last")
		return
	}

	priceinfo.Price *= pricecfg.Factor
	t := time.Now().Round(0)
	priceinfo.LastUpdatedReal = t
	priceinfo.LastUpdatedWander = t
	return
}
