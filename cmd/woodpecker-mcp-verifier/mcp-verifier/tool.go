package mcpverifier

import (
	"fmt"
)

// Takes the schema passed of each tool and parse it to find an input of type string to send the payload
// It also checks for the required fields and assigns default values
func checkToolTypeParams(schemaDef any, mPayload PayloadContent) (*map[string]any, error) {
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

// setParamsSchema checks the current input schema of the tool and sets the default fields needed
// some paramters are required, setting those and using one string field to send the payload we want
func setParamsSchema(inputSchema any, mPayload PayloadContent) (map[string]any, error) {

	schema, err := checkToolTypeParams(inputSchema, mPayload)
	if err != nil {
		return nil, err
	}
	return *schema, nil
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
