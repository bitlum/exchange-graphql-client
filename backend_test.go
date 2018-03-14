package client

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shopspring/decimal"
)

func Test_graphqlBackendImplementsBackend(t *testing.T) {
	// should error in compile time if not implements
	var _ backend = &graphqlBackend{}
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

func Test_graphqlBackend_do(t *testing.T) {

	const (
		authToken = "test-auth-token"
		path      = "/query"
	)

	checkMethod := func(t *testing.T, s *mockBackendServer) {
		const want = "POST"
		got := s.request.method
		if got != want {
			t.Errorf("wrong method: got `%s` but want `%s`", got, want)
		}
	}

	checkURLPath := func(t *testing.T, s *mockBackendServer) {
		got := s.request.urlPath
		want := path
		if got != want {
			t.Errorf("wrong URL: got `%s` but want `%s`", got, want)
		}
	}

	checkHeaders := func(t *testing.T, s *mockBackendServer) {
		const wantContentType = "application/json"

		h := s.request.header

		gotContentType := h.Get("Content-Type")
		if gotContentType != wantContentType {
			t.Errorf("wrong Content-Type header: got `%s` but want `%s`",
				gotContentType, wantContentType)
		}

		gotAuthorization := h.Get("Authorization")
		if gotAuthorization != "Bearer "+authToken {
			t.Errorf("wrong Authorization header: got `%s` but want `%s`",
				gotContentType, wantContentType)
		}
	}

	checkBody := func(t *testing.T, s *mockBackendServer, wantReq request) {
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

	t.Run("when backend is down", func(t *testing.T) {
		s := newMockBackendServer()
		s.stop()
		c := &graphqlBackend{
			url:       s.url() + path,
			authToken: authToken,
		}
		_, err := c.do(request{
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
		c := &graphqlBackend{
			url:       s.url() + path,
			authToken: authToken,
		}
		req := request{
			Query: "query",
			Variables: struct {
				Var1 string `json:"var1"`
			}{"value"},
		}
		_, err := c.do(req)
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
		c := &graphqlBackend{
			url:       s.url() + path,
			authToken: authToken,
		}
		req := request{
			Query: "query",
			Variables: struct {
				Var1  string          `json:"var1"`
				Asset string          `json:"asset"`
				Dec   decimal.Decimal `json:"dec"`
			}{"value", "BTC", dec(10)},
		}
		_, err := c.do(req)
		checkMethod(t, s)
		checkURLPath(t, s)
		checkHeaders(t, s)
		checkBody(t, s, req)
		if err != nil {
			t.Errorf("want no error but got error `%s`", err.Error())
		}
	})
}

// mockBackendRequest is a bitlum backend mock service request data
type mockBackendRequest struct {
	method  string
	urlPath string
	header  http.Header
	body    []byte
	error   error
}

// mockBackendServer is mock of bitlum backend server for testing
// purposes. It stores last request data and response with preset
// response data
type mockBackendServer struct {
	httpServer *httptest.Server
	// last request data, nil if no request recieved yet
	request *mockBackendRequest
	// response data to response with to next request
	response struct {
		code int
		body string
	}
}

// newMockBackendServer return new started bitlum backend mock server
func newMockBackendServer() *mockBackendServer {
	s := &mockBackendServer{}
	s.start()
	return s
}

// start starts test http server and returns its URL
func (s *mockBackendServer) start() {
	s.request = nil
	s.httpServer = httptest.NewServer(s)
}

// stop stops test http server. Indented to be called after start and
// before next start.
func (s *mockBackendServer) stop() {
	s.httpServer.Close()
}

func (s *mockBackendServer) url() string {
	return s.httpServer.URL
}

// ServeHTTP is http.Handler implementation. Stores request data.
func (s *mockBackendServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.request = &mockBackendRequest{}
	s.request.method = r.Method
	s.request.urlPath = r.URL.String()
	s.request.header = r.Header
	s.request.body, s.request.error = ioutil.ReadAll(r.Body)
	w.WriteHeader(s.response.code)
	w.Write([]byte(s.response.body))
}
