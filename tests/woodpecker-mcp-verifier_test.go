package tests

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/operantai/woodpecker/cmd/woodpecker-mcp-verifier/utils"
	"github.com/operantai/woodpecker/cmd/woodpecker-mcp-verifier/vschema"
	mcpverifier "github.com/operantai/woodpecker/internal/mcp-verifier"
)

type MCPClientSessionMock struct{}

type MockAIFormatter struct{}

func (a *MockAIFormatter) AnalyzeSchema(inputSchema any) (map[string]any, error) {
	return map[string]any{"response": "response"}, nil
}

type MockSchemaValidator struct{}

func (v *MockSchemaValidator) ValidateWithAI(schema any, mPayload utils.PayloadContent, aiFormatter vschema.IAIFormatter) (map[string]any, error) {
	return map[string]any{"response": "response"}, nil
}

func (v *MockSchemaValidator) BasicParametersCheck(schema any, mPayload utils.PayloadContent) (map[string]any, error) {
	return map[string]any{"response": "response"}, nil
}

func (m *MCPClientSessionMock) CallTool(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {

	return &mcp.CallToolResult{
		Meta: mcp.Meta{},
		Content: []mcp.Content{
			&mcp.TextContent{
				Text:        "Looking good",
				Meta:        mcp.Meta{},
				Annotations: &mcp.Annotations{},
			},
		},
		StructuredContent: nil,
		IsError:           false,
	}, nil
}

func NewMCPClientMock() *MCPClientSessionMock {
	return &MCPClientSessionMock{}
}

var _ = Describe("MCP Client verifier tests", func() {
	Context("Test Tool call with Payload", func() {
		It("should call tools with no error and stdout the respone", func() {
			ctx := context.Background()
			csMock := NewMCPClientMock()
			toolSchema := map[string]any{
				"$schema":     "https://json-schema.org/draft/2020-12/schema",
				"title":       "User",
				"description": "A simple user schema",
				"type":        "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type":      "string",
						"minLength": 1,
					},
					"age": map[string]any{
						"type":    "integer",
						"minimum": 0,
					},
					"email": map[string]any{
						"type":   "string",
						"format": "email",
					},
					"active": map[string]any{
						"type": "boolean",
					},
				},
				"required":             []string{"name", "age", "active"},
				"additionalProperties": false,
			}

			tool := &mcp.Tool{
				Meta:         mcp.Meta{},
				Annotations:  &mcp.ToolAnnotations{},
				Description:  "just testing",
				InputSchema:  toolSchema,
				Name:         "mcp-client-verifier",
				OutputSchema: nil,
				Title:        "",
				Icons:        []mcp.Icon{},
			}
			payload := &utils.PayloadContent{
				Content: "This is going to be fun",
				Tags:    []string{"LLM"},
			}

			validator := &MockSchemaValidator{}

			mcpv, err := mcpverifier.NewMCPClient(mcpverifier.WithValidator(validator))
			Expect(err).NotTo(HaveOccurred())

			err = mcpv.ToolCallWithPayload(ctx, csMock, *tool, *payload)

			Expect(err).NotTo(HaveOccurred())
		})
	})
})
