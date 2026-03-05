// Package vschema provides the logic to validate the MCP tools input
// schema and be able to provide a test payload in accordance to it
package vschema

import (
	"fmt"

	"github.com/operantai/woodpecker/cmd/woodpecker-mcp-verifier/utils"
)

// BasicParametersCheck implements IvSchema.
func (v *VSchema) BasicParametersCheck(schema any, mPayload utils.PayloadContent) (map[string]any, error) {
	resp, err := checkToolTypeParams(schema, mPayload)
	return *resp, err
}

// ValidateWithAI implements IvSchema.
func (v *VSchema) ValidateWithAI(schema any, mPayload utils.PayloadContent, aiFormatter IAIFormatter) (map[string]any, error) {
	respSchema, err := aiFormatter.AnalyzeSchema(schema)

	if err != nil {
		return nil, err
	}
	params := addPayload(&respSchema, mPayload)

	return *params, err
}

// Takes the schema passed of each tool and parse it to find an input of type string to send the payload
// It also checks for the required fields and assigns default values
func checkToolTypeParams(schemaDef any, mPayload utils.PayloadContent) (*map[string]any, error) {
	// Assert the input is a map
	schema, ok := schemaDef.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("error formatting input schema from tool")
	}

	params := &map[string]any{}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("input schema does not have a properties section")
	}

	rawRequired, _ := schema["required"].([]any)
	required := map[string]bool{}

	for _, r := range rawRequired {
		if s, ok := r.(string); ok {
			required[s] = true
		}
	}

	// Loop to set default values for the required fields
	for field, rawProp := range properties {
		prop, ok := rawProp.(map[string]any)
		if !ok {
			continue
		}

		fieldType, _ := prop["type"].(string)

		switch {
		case required[field]:
			(*params)[field] = defaultForJSONType(fieldType)
		}
	}

	// Second loop over to make sure we find a string type to send through the payload
	for field, rawProp := range properties {
		prop, ok := rawProp.(map[string]any)
		if !ok {
			continue
		}

		fieldType, _ := prop["type"].(string)

		if fieldType == "string" {
			(*params)[field] = mPayload.Content
			return params, nil
		}
	}
	return params, nil
}

// addPayload loops over the json response and adds the payload we want to the first string type field
func addPayload(data *map[string]any, mPayload utils.PayloadContent) *map[string]any {
	for key, value := range *data {
		if keyVal, ok := value.(string); ok && key == "my_custom_field" {
			(*data)[keyVal] = mPayload.Content
			delete(*data, key)
			break
		}
	}
	return data
}

// defaultForJSONType sets default values to field types
func defaultForJSONType(t string) any {
	switch t {
	case "string":
		return ""
	case "number":
		return 0
	case "integer":
		return 0
	case "boolean":
		return false
	case "array":
		return []any{}
	case "object":
		return map[string]any{}
	default:
		return nil
	}
}
