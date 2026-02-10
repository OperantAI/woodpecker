// Package mcpverifier defines the creation of an MCP client that will connect to an MCP server, discover their tools
// and send a bulk of payload requests defined in a json config file.
package mcpverifier

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/operantai/woodpecker/cmd/woodpecker-mcp-verifier/mcp-verifier/oauth"
	"github.com/operantai/woodpecker/internal/output"
)

var (
	httpClient *http.Client
	once       sync.Once
)

type HeaderTransport struct {
	Base          http.RoundTripper
	CustomHeaders map[string]string
}

func (t *HeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Loop over the custom headers and set them
	for key, val := range t.CustomHeaders {
		req.Header.Add(key, val)
	}
	return t.Base.RoundTrip(req)
}

// GetHTTPClient returns a singleton instance of http.Client.
func GetHTTPClient(opts *oauth.HTTPTransportOptions) *http.Client {

	once.Do(func() {
		transport, err := oauth.NewHTTPTransport(oauth.OauthHandler, opts)
		if err != nil {
			output.WriteFatal("An error performing the Oauth flow happened: %v", err)
		}
		httpClient = &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		}
	})
	return httpClient
}

type MCPConfig struct {
	Config MCPConfigConnection `json:"config"`
}

type MCPConfigConnection struct {
	CustomHeaders map[string]string `json:"customHeaders"`
	Payloads      []PayloadContent  `json:"payloads"`
}

type PayloadContent struct {
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}

type IMCPClient interface {
	// Calls an MCP tool with a set of crafted payloads if it has any string input in its inputschema
	ToolCallWithPayload(ctx context.Context, cs IMCPClientSession, tool mcp.Tool, mPayload PayloadContent) error
	// Gets a MCP config from a json file
	GetMCPConfig(jsonPayloadPath string) (*MCPConfig, error)
}

type mcpClient struct{}

func NewMCPClient() IMCPClient {
	return &mcpClient{}
}

type IMCPClientSession interface {
	// CallTool calls the tool with the given parameters.
	//
	// The params.Arguments can be any value that marshals into a JSON object.
	CallTool(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error)
}
