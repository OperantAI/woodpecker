package vschema

import (
	"fmt"

	"github.com/operantai/woodpecker/cmd/woodpecker-mcp-verifier/utils"
	"github.com/tmc/langchaingo/llms"
)

// IvSchema sets the methods to validate an input schema from an MCP tool with the expected input
type IvSchema interface {
	ValidateWithAI(schema any, mPayload utils.PayloadContent, aiFormatter IAIFormatter) (map[string]any, error)
	BasicParametersCheck(schema any, mPayload utils.PayloadContent) (map[string]any, error)
}

// IAIFormatter propose how to format a payload response based on an Input Schema using LLM
type IAIFormatter interface {
	AnalyzeSchema(inputSchema any) (map[string]any, error)
}

type VSchema struct{}

type AIFormatter struct {
	Model   llms.Model
	Options []llms.CallOption
	Enabled bool
}

func NewAIFormatter(
	model llms.Model,
	opts ...llms.CallOption,
) (IAIFormatter, error) {
	if model == nil {
		return nil, fmt.Errorf("LLM client is nil, probablly not initialized correctly")
	}
	return &AIFormatter{
		Model:   model,
		Options: opts,
	}, nil
}

func NewVSchema() IvSchema {
	return &VSchema{}
}
