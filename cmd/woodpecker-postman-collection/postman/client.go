package postman

import (
	"net/http"
	"sync"
	"time"
)

var (
	httpClient *http.Client
	once       sync.Once
)

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
			Transport: &http.Transport{
				MaxIdleConns:        100,              // Max idle connections
				IdleConnTimeout:     90 * time.Second, // Idle connection timeout
				TLSHandshakeTimeout: 10 * time.Second,
			},
		}
	})
	return httpClient
}
