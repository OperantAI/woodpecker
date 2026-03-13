// Package mcpverifier defines the creation of an MCP client that will connect to an MCP server, discover their tools
// and send a bulk of payload requests defined in a json config file.
package mcpverifier

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/operantai/woodpecker/cmd/woodpecker-mcp-verifier/utils"
	"github.com/operantai/woodpecker/cmd/woodpecker-mcp-verifier/vschema"
	"github.com/operantai/woodpecker/internal/mcp-verifier/oauth"
	"github.com/operantai/woodpecker/internal/output"
	"github.com/spf13/viper"
	"github.com/tmc/langchaingo/llms/openai"
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

type IMCPClient interface {
	// Calls an MCP tool with a set of crafted payloads if it has any string input in its inputschema
	ToolCallWithPayload(ctx context.Context, cs IMCPClientSession, tool mcp.Tool, mPayload utils.PayloadContent) error
	// Gets a MCP config from a json file
	GetMCPConfig(jsonPayloadPath string) (*utils.MCPConfig, error)
}

type mcpClient struct {
	validator      vschema.IvSchema
	aiFormatter    vschema.IAIFormatter
	useAi          bool
	name           string
	experimentType string
}

type Option func(*mcpClient)

func WithValidator(validator vschema.IvSchema) Option {
	return func(mc *mcpClient) {
		mc.validator = validator
	}
}

func WithAIFormatter(useAI bool) Option {
	return func(mc *mcpClient) {
		mc.useAi = useAI
	}
}

func WithName(name string) Option {
	return func(mc *mcpClient) {
		mc.name = name
	}
}

func WithExperimentType(experimentType string) Option {
	return func(mc *mcpClient) {
		mc.experimentType = experimentType
	}
}

func NewMCPClient(options ...Option) (IMCPClient, error) {
	mc := &mcpClient{}

	// Apply optional configurations
	for _, option := range options {
		option(mc)
	}

	// Current module implementation of structure output is not that dynamic for the current needs where we dont
	// know the input schema format in advance and we want to leverage it to test the tool
	// Also you can pass the WOODPECKER_LLM_BASE_URL and should work with any OpenAI compatible APIs. You can use the
	// OPENAI_API_KEY env var, to auth the provider.
	if mc.useAi {
		llm, err := openai.New(openai.WithModel(viper.GetString("LLM_MODEL")), openai.WithBaseURL(viper.GetString("LLM_BASE_URL")), openai.WithToken(viper.GetString("LLM_AUTH_TOKEN")))
		if err != nil {
			return nil, fmt.Errorf("an error initializing the LLM client: %v", err)
		}
		mc.aiFormatter, err = vschema.NewAIFormatter(llm)
		if err != nil {
			return nil, err
		}

		output.WriteInfo("Validating schema using an AI formatter ...")
	}
	return mc, nil
}

type IMCPClientSession interface {
	// CallTool calls the tool with the given parameters.
	//
	// The params.Arguments can be any value that marshals into a JSON object.
	CallTool(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error)
}

type ToolResponses struct {
	ToolName   string   `json:"toolName"`
	Response   string   `json:"response"`
	Tags       []string `json:"tags"`
	Parameters any      `json:"parameters"`
}
