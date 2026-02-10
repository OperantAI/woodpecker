// Package mcpverifier defines the creation of an MCP client that will connect to an MCP server, discover their tools
// and send a bulk of payload requests defined in a json config file.
package mcpverifier

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/operantai/woodpecker/cmd/woodpecker-mcp-verifier/mcp-verifier/oauth"
	"github.com/operantai/woodpecker/cmd/woodpecker-mcp-verifier/utils"
	"github.com/operantai/woodpecker/internal/output"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

// RunClient entry point to start the MCP client connection
func RunClient(ctx context.Context, serverURL string, protocol utils.MCMCPprotocol, cmdArgs *[]string, payloadPath string) error {
	output.WriteInfo("Connecting to server: %s", serverURL)
	output.WriteInfo("Using protocol: %s", protocol)

	mcpClient := NewMCPClient()
	mcpConfig, err := mcpClient.GetMCPConfig(payloadPath)
	if err != nil {
		return err
	}
	transport := getTransport(serverURL, protocol, cmdArgs, mcpConfig.Config)
	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-client", Version: "v1.0.0"}, nil)
	cs, err := client.Connect(ctx, transport, nil)
	if err != nil {
		output.WriteFatal("Error initializing client: %v", err)
	}
	defer cs.Close()

	if cs.InitializeResult().Capabilities.Tools != nil {

		if err != nil {
			return err
		}

		tools := cs.Tools(ctx, nil)
		// Collect all tools first (unchanged)
		var allTools []mcp.Tool
		for tool := range tools {
			allTools = append(allTools, *tool)
		}
		if err := setupBulkOperation(ctx, cs, &allTools, &mcpConfig.Config.Payloads, mcpClient); err != nil {
			return err
		}

	}
	return nil
}

// Setup concurrency to call multiple tools from the MCP server at a time with the tool payload
func setupBulkOperation(ctx context.Context, cs *mcp.ClientSession, allTools *[]mcp.Tool, mPayloads *[]PayloadContent, mMCPClient IMCPClient) error {
	// Concurrent calls with error grouping and a concurrency limit
	var eg errgroup.Group
	maxConcurrency := 10

	if v := os.Getenv("WOODPECKER_MAX_CONCURRENCY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxConcurrency = n
		} else {
			output.WriteWarning("invalid WOODPECKER_MAX_CONCURRENCY=%q, using %d", v, maxConcurrency)
		}
	}
	sem := make(chan struct{}, maxConcurrency)

	for _, tool := range *allTools {
		for _, payload := range *mPayloads {
			// Copy of the parameters to avoid race conditions
			t := tool
			p := payload

			sem <- struct{}{} // acquire
			eg.Go(func() error {
				defer func() { <-sem }() // ensure release after tool call completion

				if err := mMCPClient.ToolCallWithPayload(ctx, cs, t, p); err != nil {
					// return error to errgroup, other goroutines still run
					return fmt.Errorf("tool %s: %w", t.Name, err)
				}
				return nil
			})
		}
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}

// Configures the MCP protocol to use based on the user selection
func getTransport(serverURL string, protocol utils.MCMCPprotocol, cmdArgs *[]string, mcpConfig MCPConfigConnection) mcp.Transport {
	var opts *oauth.HTTPTransportOptions
	woodPeckerEnabled := strings.ToLower(viper.GetString("WOODPECKER_OAUTH_CLIENT_ID"))
	if woodPeckerEnabled == "true" {
		opts = &oauth.HTTPTransportOptions{
			Base: &http.Transport{
				MaxIdleConns:        100,              // Max idle connections
				IdleConnTimeout:     90 * time.Second, // Idle connection timeout
				TLSHandshakeTimeout: 10 * time.Second,
			},
			CustomHeaders: mcpConfig.CustomHeaders,
		}
	} else {
		opts = nil
	}
	switch protocol {
	case utils.STREAMABLEHTTP:
		output.WriteInfo("Setting streamabale-http transport connection.")
		hClient := GetHTTPClient(opts)
		transport := &mcp.StreamableClientTransport{
			Endpoint:   serverURL,
			HTTPClient: hClient,
		}
		return transport
	case utils.SSE:
		output.WriteWarning("Setting SSE transport connection. It will be deprecated soon")
		hClient := GetHTTPClient(opts)
		transport := &mcp.SSEClientTransport{
			Endpoint:   serverURL,
			HTTPClient: hClient,
		}
		return transport
	default:
		output.WriteInfo("Setting a local STDIO transport connection.")
		cmd := exec.Command((*cmdArgs)[0], (*cmdArgs)[1:]...)
		transport := &mcp.CommandTransport{Command: cmd}
		return transport
	}
}

func (m *mcpClient) ToolCallWithPayload(ctx context.Context, cs IMCPClientSession, tool mcp.Tool, mPayload PayloadContent) error {

	params, err := setParamsSchema(tool.InputSchema, mPayload)
	if err != nil {
		return err
	}

	result, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      tool.Name,
		Arguments: params,
	})

	if err != nil {
		return err
	}
	for _, content := range result.Content {
		data, err := content.MarshalJSON()
		if err != nil {
			return err
		}
		output.WriteInfo("Tool {%s}: %s, tags: %s", tool.Name, string(data), mPayload.Tags)
	}
	return nil
}

func (m *mcpClient) GetMCPConfig(jsonPayloadPath string) (*MCPConfig, error) {
	// Read the JSON file
	jsonData, err := os.ReadFile(jsonPayloadPath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %s, %v", jsonPayloadPath, err)
	}
	var collection MCPConfig
	err = json.Unmarshal(jsonData, &collection)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %v", err)
	}
	return &collection, nil
}
