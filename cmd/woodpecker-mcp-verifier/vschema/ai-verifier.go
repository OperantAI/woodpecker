package vschema

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/operantai/woodpecker/internal/output"
	"github.com/tmc/langchaingo/llms"
)

// AnalyzeSchema implements IAIFormatter where it uses an LLM to generate a formatted response based on the tool input schema
func (a *AIFormatter) AnalyzeSchema(inputSchema any) (map[string]any, error) {
	ctx := context.Background()

	var result map[string]any

	// Marshal the map into a byte slice
	bSchema, err := json.Marshal(inputSchema)
	if err != nil {
		return nil, err
	}
	content := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, "You are an assistant that output JSON responses based on JSON schemas. Your purpose is to always response with a valid JSON object that satisfies the wanted schema . You just need to provide the minium fields and data so the schema expected is correct. Use default values based on the name of the fields. Always provide values for the \"required\" fields. IMPORTANT: From the JSON schema select one already present field, of type string/text with no validation or enums, the field must be of string type so the user can send free text. Add the name of that JSON field to the schema response with the name: \"my_custom_field\""),
		llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("Give me a json example data that satisfies the following input schema: %s", string(bSchema))),
	}
	a.Options = append(a.Options, llms.WithJSONMode())
	response, err := a.Model.GenerateContent(ctx, content, a.Options...)
	if err != nil {
		return nil, fmt.Errorf("an error generating response with the LLM: %v", err)
	}

	data := response.Choices[0].Content

	err = json.Unmarshal([]byte(data), &result)
	if err != nil {
		return nil, err
	}

	output.WriteInfo("AI response ...")
	output.WriteJSON(result)

	return result, nil
}
