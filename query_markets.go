package client

import (
	"encoding/json"
	"errors"

	"github.com/shopspring/decimal"
)

// MarketsRequest is a query variables used in request
// markets statuses
type MarketsRequest struct {
	Markets []string `json:"markets"`
}

type MarketStatus struct {
	Market     string
	Stock      string
	Money      string
	Open       decimal.Decimal
	Close      decimal.Decimal
	High       decimal.Decimal
	Last       decimal.Decimal
	Low        decimal.Decimal
	Volume     decimal.Decimal
	ChangeLast decimal.Decimal
	ChangeHigh decimal.Decimal
	ChangeLow  decimal.Decimal
	BestAsk    decimal.Decimal
	BestBid    decimal.Decimal
}

// Markets shows markets statuse
func (c *Client) Markets(markets MarketsRequest) ([]MarketStatus, error) {

	var req request

	req.Query = `
		query Markets($markets: [Market!]!) {
			markets (markets: $markets){
				market
				stock
				money
				open
				close
				high
				last
				low
				volume
				changeLast
				changeHigh
				changeLow
				bestAsk
				bestBid
  			}
		}
	`

	req.Variables = markets

	respJSON, err := c.do(req)
	if err != nil {
		return []MarketStatus{},
			errors.New("failed to do request: " + err.Error())
	}

	resp := struct {
		responseBase
		Data struct {
			Markets []MarketStatus `json:"markets"`
		}
	}{}
	if err := json.Unmarshal(respJSON, &resp); err != nil {
		return []MarketStatus{},
			errors.New("failed to json.Unmarshal resp: " + err.Error())
	}

	if err := resp.Error(); err != nil {
		return resp.Data.Markets,
			errors.New("exchange error: " + err.Error())
	}

	return resp.Data.Markets, nil
}
