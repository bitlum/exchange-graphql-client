package client

import (
	"encoding/json"
	"errors"
)

// Accounts shows balances for the assets owned by loggedin user
// using specified invoice.
func (c *Client) IssueApiToken() (string, error) {

	var req request

	req.Query = `
		query { issueApiToken }
	`

	resp := struct {
		responseBase
		Data struct {
			IssueApiToken string `json:"issueApiToken"`
		}
	}{}

	respJSON, err := c.do(req)
	if err != nil {
		return "",
			errors.New("failed to do request: " + err.Error())
	}

	if err := json.Unmarshal(respJSON, &resp); err != nil {
		return "",
			errors.New("failed to json.Unmarshal resp: " + err.Error())
	}

	if err := resp.Error(); err != nil {
		return resp.Data.IssueApiToken,
			errors.New("exchange error: " + err.Error())
	}

	return resp.Data.IssueApiToken, nil
}
