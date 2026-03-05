// Package mcpverifier defines the creation of an MCP client that will connect to an MCP server, discover their tools
// and send a bulk of payload requests defined in a json config file.
package mcpverifier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/operantai/woodpecker/cmd/woodpecker-mcp-verifier/utils"
	"github.com/operantai/woodpecker/cmd/woodpecker-mcp-verifier/vschema"
	"github.com/operantai/woodpecker/internal/mcp-verifier/oauth"
	"github.com/operantai/woodpecker/internal/output"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

// RunClient entry point to start the MCP client connection
func RunClient(ctx context.Context, serverURL string, protocol utils.MCMCPprotocol, cmdArgs *[]string, payloadPath string) error {
	output.WriteInfo("Connecting to server: %s", serverURL)
	output.WriteInfo("Using protocol: %s", protocol)

	sValidator := vschema.NewVSchema()
	mcpClient, err := NewMCPClient(WithValidator(sValidator), WithAIFormatter(viper.GetBool("USE_AI_FORMATTER")))
	if err != nil {
		return err
	}
	mcpConfig, err := mcpClient.GetMCPConfig(payloadPath)
	if err != nil {
		return err
	}
	transport := getTransport(serverURL, protocol, cmdArgs, mcpConfig.Config)
	client := mcp.NewClient(&mcp.Implementation{Name: "woodpecker-mcp-verifier", Version: "v1.0.0"}, nil)
	cs, err := client.Connect(ctx, transport, nil)
	if err != nil {
		output.WriteFatal("Error initializing client: %v", err)
	}
	defer cs.Close()

	if cs.InitializeResult().Capabilities.Tools != nil {

		if err != nil {
			return err
		}

		// Collect tools and filter the allowed ones, by default all
		tools := cs.Tools(ctx, nil)
		allowedTools := mcpConfig.Config.AllowedTools
		var allTools []mcp.Tool
		for tool := range tools {
			if len(allowedTools) > 0 {
				if slices.Contains(allowedTools, tool.Name) {
					allTools = append(allTools, *tool)
				}
			} else {
				allTools = append(allTools, *tool)
			}
		}
		if err := setupBulkOperation(ctx, cs, &allTools, &mcpConfig.Config.Payloads, mcpClient); err != nil {
			return err
		}

	}
	return nil
}

// Setup concurrency to call multiple tools from the MCP server at a time with the tool payload
func setupBulkOperation(ctx context.Context, cs *mcp.ClientSession, allTools *[]mcp.Tool, mPayloads *[]utils.PayloadContent, mMCPClient IMCPClient) error {
	// Concurrent calls with error grouping and a concurrency limit
	maxConcurrency := 10
	if v := os.Getenv("MAX_CONCURRENCY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxConcurrency = n
		} else {
			output.WriteWarning("invalid WOODPECKER_MAX_CONCURRENCY=%q, using %d", v, maxConcurrency)
		}
	}
	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(maxConcurrency)

	for _, tool := range *allTools {
		for _, payload := range *mPayloads {
			// Copy of the parameters to avoid race conditions
			t := tool
			p := payload

			eg.Go(func() error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				if err := mMCPClient.ToolCallWithPayload(ctx, cs, t, p); err != nil {
					// just write the error, other goroutines still run
					output.WriteError("tool %s: %v", t.Name, err)
					switch {
					case errors.Is(err, mcp.ErrConnectionClosed):
						return err
						// Fatal condition
					case strings.Contains(err.Error(), "Internal Server Error"):
						return err // triggers cancellation
					default:
						return nil
					}
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
func getTransport(serverURL string, protocol utils.MCMCPprotocol, cmdArgs *[]string, mcpConfig utils.MCPConfigConnection) mcp.Transport {
	opts := &oauth.HTTPTransportOptions{
		Base: &http.Transport{
			MaxIdleConns:        100,              // Max idle connections
			IdleConnTimeout:     90 * time.Second, // Idle connection timeout
			TLSHandshakeTimeout: 10 * time.Second,
		},
		CustomHeaders: mcpConfig.CustomHeaders,
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

func (m *mcpClient) ToolCallWithPayload(ctx context.Context, cs IMCPClientSession, tool mcp.Tool, mPayload utils.PayloadContent) error {
	var params map[string]any
	var err error
	useAi := viper.GetBool("USE_AI_FORMATTER")
	if useAi {
		params, err = m.validator.ValidateWithAI(tool.InputSchema, mPayload, m.aiFormatter)
	} else {
		params, err = m.validator.BasicParametersCheck(tool.InputSchema, mPayload)
	}

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
		resp := map[string]any{
			"tool":     tool.Name,
			"response": string(data),
			"tags":     mPayload.Tags,
		}
		output.WriteInfo("Tool response ...")
		output.WriteJSON(resp)
	}
	return nil
}

func (m *mcpClient) GetMCPConfig(jsonPayloadPath string) (*utils.MCPConfig, error) {
	// Read the JSON file
	jsonData, err := os.ReadFile(jsonPayloadPath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %s, %v", jsonPayloadPath, err)
	}
	var collection utils.MCPConfig
	err = json.Unmarshal(jsonData, &collection)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %v", err)
	}
	// Add Auth headers
	auth := viper.GetString("AUTH_HEADER")
	if auth != "" {
		collection.Config.CustomHeaders["Authorization"] = auth
	}

	return &collection, nil
}
