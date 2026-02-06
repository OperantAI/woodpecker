package mcpverifier

import (
	"net/http"
	"sync"
	"time"
)

type MCPClient interface {

}

var (
	httpClient *http.Client
	once       sync.Once
)

type HeaderTransport struct {
	Base http.RoundTripper
	CustomHeaders map[string]string
}

func (t *HeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Loop over the custom headers and set them
	for key, val := range t.CustomHeaders {
		req.Header.Add(key,val)
	}
	return t.Base.RoundTrip(req)
}

// Base HTTP client interface
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Base HTTP client implementation
type BaseHTTPClient struct{}

func (c *BaseHTTPClient) Do(req *http.Request) (*http.Response, error) {
	client := GetHTTPClient()
	return client.Do(req)
}

func NewBaseHTTPClient() HTTPClient {
	return &BaseHTTPClient{}
}

// GetHTTPClient returns a singleton instance of http.Client.
func GetHTTPClient() *http.Client {
	once.Do(func() {
		httpClient = &http.Client{
			Timeout: 30 * time.Second, // Example timeout
			Transport: &HeaderTransport{
				Base: &http.Transport{
					MaxIdleConns:        100,              // Max idle connections
					IdleConnTimeout:     90 * time.Second, // Idle connection timeout
					TLSHandshakeTimeout: 10 * time.Second,
				},
				CustomHeaders: map[string]string{
					"o-mcp-client-name":"woodpecker-mcp-client",
				},
			},
		}
	})
	return httpClient
}


type MaliciousPayload struct{
	Payload string `json:"payload"`
}
