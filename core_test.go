package client

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bitlum/macaroon-application-auth"
	"github.com/shopspring/decimal"
)

const macaroonHexEncoded = "0201066269746c756d0204811f79090002166469736f70732069737375655f6170695f746f6b656e00020f7573657220323136363332333436350000062023ffa8c3ba9fa8a8cda6171a313fcfdfc98b52410f03685c448583cf1be01d04"

func Test_graphQLCoreImplementsCore(t *testing.T) {
	// should error in compile time if not implements
	var _ core = &graphQLCore{}
}

func Test_responseBase_Error(t *testing.T) {
	tests := []struct {
		name       string
		rb         responseBase
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:    "nil errors",
			rb:      responseBase{Errors: nil},
			wantErr: false,
		},
		{
			name:    "empty errors",
			rb:      responseBase{Errors: []responseError{}},
			wantErr: false,
		},
		{
			name: "one error with one location",
			rb: responseBase{
				Errors: []responseError{
					{
						Message: "some error",
						Locations: []responseErrorLocation{
							{Line: 123, Column: 12},
						},
					},
				},
			},
			wantErr:    true,
			wantErrMsg: "some error, location: 123:12",
		},
		{
			name: "one error with multiple locations",
			rb: responseBase{
				Errors: []responseError{
					{
						Message: "some error",
						Locations: []responseErrorLocation{
							{Line: 123, Column: 12},
							{Line: 567, Column: 13},
							{Line: 890, Column: 14},
						},
					},
				},
			},
			wantErr:    true,
			wantErrMsg: "some error, locations: 123:12, 567:13, 890:14",
		},
		{
			name: "multiple errors with one location in first",
			rb: responseBase{
				Errors: []responseError{
					{
						Message: "some error",
						Locations: []responseErrorLocation{
							{Line: 123, Column: 12},
						},
					},
					{Message: "second error"},
				},
			},
			wantErr: true,
			wantErrMsg: "2 errors occurred, first one is: " +
				"some error, location: 123:12",
		},
		{
			name: "multiple errors with multiple locations in first",
			rb: responseBase{
				Errors: []responseError{
					{
						Message: "some error",
						Locations: []responseErrorLocation{
							{Line: 123, Column: 12},
							{Line: 567, Column: 13},
							{Line: 890, Column: 14},
						},
					},
					{Message: "second error"},
					{Message: "third error"},
				},
			},
			wantErr: true,
			wantErrMsg: "3 errors occurred, first one is: " +
				"some error, locations: 123:12, 567:13, 890:14",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rb.Error(); (err != nil) != tt.wantErr {
				t.Errorf("responseBase.Error() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.wantErr && tt.wantErrMsg != tt.rb.Error().Error() {
				t.Errorf("responseBase.Error() error message = `%s`, "+
					"wantErrMsg = `%s`", tt.rb.Error().Error(), tt.wantErrMsg)
			}
		})
	}
}

func Test_graphQLCore_do(t *testing.T) {

	const (
		path = "/query"
	)

	mac, err := auth.DecodeMacaroon(macaroonHexEncoded)
	if err != nil {
		t.Fatalf("failed to decode macaroon: %v", err)
	}

	checkMethod := func(t *testing.T, s *mockExchangeServer) {
		const want = "POST"
		got := s.request.method
		if got != want {
			t.Errorf("wrong method: got `%s` but want `%s`", got, want)
		}
	}

	checkURLPath := func(t *testing.T, s *mockExchangeServer) {
		got := s.request.urlPath
		want := path
		if got != want {
			t.Errorf("wrong URL: got `%s` but want `%s`", got, want)
		}
	}

	checkHeaders := func(t *testing.T, s *mockExchangeServer) {
		const wantContentType = "application/json"

		h := s.request.header

		gotContentType := h.Get("Content-Type")
		if gotContentType != wantContentType {
			t.Errorf("wrong Content-Type header: got `%s` but want `%s`",
				gotContentType, wantContentType)
		}

		wantAuthStartWith := "Macaroon "
		gotAuthorization := h.Get("Authorization")
		if strings.Index(gotAuthorization, wantAuthStartWith) != 0 {
			t.Errorf("wrong Authorization header: got `%s` but want"+
				" to start with `%s`",
				gotAuthorization, wantAuthStartWith)
		}
	}

	checkBody := func(t *testing.T, s *mockExchangeServer, wantReq request) {
		want, err := json.Marshal(wantReq)
		if err != nil {
			t.Fatalf("failed to json.Marshal request: " + err.Error())
		}
		got := s.request.body
		if string(want) != string(got) {
			t.Errorf("wrong body: got `%s` but want `%s`",
				got, want)
		}
	}

	t.Run("when exchange is down", func(t *testing.T) {
		s := newMockBackendServer()
		s.stop()
		c := &graphQLCore{
			url:      s.url() + path,
			macaroon: mac,
		}
		_, err := c.do(true, request{
			Query: "query",
			Variables: struct {
				Var1 string `json:"var1"`
			}{"value"},
		})
		if err == nil {
			t.Error("want error but got not error")
		}
	})
	t.Run("when 301 status code", func(t *testing.T) {
		s := newMockBackendServer()
		defer s.stop()
		s.response.code = 301
		c := &graphQLCore{
			url:      s.url() + path,
			macaroon: mac,
		}
		req := request{
			Query: "query",
			Variables: struct {
				Var1 string `json:"var1"`
			}{"value"},
		}
		_, err := c.do(true, req)
		checkMethod(t, s)
		checkURLPath(t, s)
		checkHeaders(t, s)
		checkBody(t, s, req)
		if err == nil {
			t.Error("want error but got no error")
		}
	})
	t.Run("when 200 status code", func(t *testing.T) {
		s := newMockBackendServer()
		defer s.stop()
		s.response.code = 200
		s.response.body = "response body"
		c := &graphQLCore{
			url:      s.url() + path,
			macaroon: mac,
		}
		req := request{
			Query: "query",
			Variables: struct {
				Var1  string          `json:"var1"`
				Asset string          `json:"asset"`
				Dec   decimal.Decimal `json:"dec"`
			}{"value", "BTC", dec(10)},
		}
		_, err := c.do(true, req)
		checkMethod(t, s)
		checkURLPath(t, s)
		checkHeaders(t, s)
		checkBody(t, s, req)
		if err != nil {
			t.Errorf("want no error but got error `%s`", err.Error())
		}
	})
}

// mockBackendRequest is a bitlum core mock service request data.
type mockBackendRequest struct {
	method  string
	urlPath string
	header  http.Header
	body    []byte
	error   error
}

// mockExchangeServer is mock of bitlum exchange server for testing
// purposes. It stores last request data and response with preset
// response data.
type mockExchangeServer struct {
	httpServer *httptest.Server
	// last request data, nil if no request recieved yet
	request *mockBackendRequest
	// response data to response with to next request
	response struct {
		code int
		body string
	}
}

// newMockBackendServer return new started bitlum core mock server.
func newMockBackendServer() *mockExchangeServer {
	s := &mockExchangeServer{}
	s.start()
	return s
}

// start starts test http server and returns its URL.
func (s *mockExchangeServer) start() {
	s.request = nil
	s.httpServer = httptest.NewServer(s)
}

// stop stops test http server. Indented to be called after start and
// before next start.
func (s *mockExchangeServer) stop() {
	s.httpServer.Close()
}

// url returns current http server URL.
func (s *mockExchangeServer) url() string {
	return s.httpServer.URL
}

// ServeHTTP is http.Handler implementation. Stores request data and
// responses with predefined data.
func (s *mockExchangeServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.request = &mockBackendRequest{}
	s.request.method = r.Method
	s.request.urlPath = r.URL.String()
	s.request.header = r.Header
	s.request.body, s.request.error = ioutil.ReadAll(r.Body)
	w.WriteHeader(s.response.code)
	w.Write([]byte(s.response.body))
}
