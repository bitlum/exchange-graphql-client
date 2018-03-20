package client

import (
	"encoding/json"
	"errors"

	"github.com/shopspring/decimal"
)

// AccountsRequest is a query variables used in request user's account
// balances
type AccountsRequest struct {
	Assets []string `json:"assets"`
}

type Account struct {
	Asset      string
	Address    string
	Available  decimal.Decimal
	Estimation decimal.Decimal
	Freezed    decimal.Decimal
	Pending    decimal.Decimal
}

// Accounts shows balances for the assets owned by loggedin user
// using specified invoice.
func (c *Client) Accounts(assets []string) ([]Account, error) {

	var req request

	req.Query = `
		query Accounts($assets: [Asset!]!) {
  			accounts( assets: $assets) {
				asset, address, available, estimation, freezed
  			}
		}
	`

	req.Variables = AccountsRequest{
		Assets: assets,
	}

	resp := struct {
		responseBase
		Data struct {
			Accounts []Account `json:"accounts"`
		}
	}{}

	respJSON, err := c.do(req)
	if err != nil {
		return []Account{},
			errors.New("failed to do request: " + err.Error())
	}

	if err := json.Unmarshal(respJSON, &resp); err != nil {
		return []Account{},
			errors.New("failed to json.Unmarshal resp: " + err.Error())
	}

	if err := resp.Error(); err != nil {
		return resp.Data.Accounts,
			errors.New("exchange error: " + err.Error())
	}

	return resp.Data.Accounts, nil
}
