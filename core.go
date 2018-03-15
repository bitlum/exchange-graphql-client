package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
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
	url       string
	authToken string
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

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.authToken)

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
