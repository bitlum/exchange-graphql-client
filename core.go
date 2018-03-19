package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/bitlum/macaroon-application-auth"
	"gopkg.in/macaroon.v2"
)

// core is client core which perform low level http request. Used for
// decouple client from real exchange backend.
// Has two implementations:
// 1. graphQLCore: used for real requests to exchange GraphQL server.
// 2. mockCore: used for testing purposes.
type core interface {
	do(r request) ([]byte, error)
}

// graphQLCore is client core implementation used to perform authorized
// http requests to exchange GraphQL server.
type graphQLCore struct {
	url      string
	macaroon *macaroon.Macaroon

	// nonce is nonce counter used to protect client from replay-attack.
	nonce int64

	jwt string
}

// do performs authorized GraphQL request to bitlum exchange service and
// returns response body.
func (c *graphQLCore) do(r request) ([]byte, error) {

	reqJSON, err := json.Marshal(r)
	if err != nil {
		return nil, errors.New("failed to json.Marshal request: " +
			err.Error())
	}

	httpReq, err := http.NewRequest("POST", c.url,
		bytes.NewBuffer(reqJSON))
	if err != nil {
		return nil, errors.New("failed to http.NewRequest: " +
			err.Error())
	}

	if c.jwt == "" {
		// Each request should have increased nonce to protect client from
		// replay-attack.
		c.nonce++

		// Adding nonce to protect client from replay-attack.
		m, err := auth.AddNonce(c.macaroon, c.nonce)
		if err != nil {
			return nil, errors.New(
				"failed to add nonce to macaroon: " + err.Error())
		}

		// Adding current time to protect client from replay-attack.
		m, err = auth.AddCurrentTime(m)
		if err != nil {
			return nil, errors.New(
				"failed to add current time to macaroon: " + err.Error())
		}

		token, err := auth.EncodeMacaroon(m)
		if err != nil {
			return nil, errors.New(
				"failed to encode macaroon: " + err.Error())
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Macaroon "+token)
	} else {
		httpReq.Header.Set("Authorization", "Bearer "+c.jwt)
	}

	httpResp, err := (&http.Client{}).Do(httpReq)
	if err != nil {
		return nil, errors.New("failed to do http request: " +
			err.Error())
	}

	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response status: %s",
			httpResp.Status)
	}

	body, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return nil, errors.New("failed to read response body: " +
			err.Error())
	}

	return body, nil
}

// request is the GraphQL request.
type request struct {
	Query     string      `json:"query"`
	Variables interface{} `json:"variables"`
}

// responseBase is the GraphQL response base, supposed to be embedded
// into specific responses.
type responseBase struct {
	Errors []responseError
}

type responseError struct {
	Message   string
	Locations []responseErrorLocation
}

type responseErrorLocation struct {
	Line   int
	Column int
}

func (rb responseBase) Error() error {
	if len(rb.Errors) == 0 {
		return nil
	}
	e := rb.Errors[0]
	msg := e.Message
	switch len(e.Locations) {
	case 0:
	case 1:
		l := e.Locations[0]
		msg = fmt.Sprintf("%s, location: %d:%d", msg,
			l.Line, l.Column)
	default:
		msg = msg + ", locations: "
		for i, l := range e.Locations {
			if i > 0 {
				msg += ", "
			}
			msg += fmt.Sprintf("%d:%d", l.Line, l.Column)
		}
	}
	if len(rb.Errors) > 1 {
		msg = fmt.Sprintf("%d errors occurred, first one is: %s",
			len(rb.Errors), msg)
	}
	return errors.New(msg)
}
