package postman

import (
	"fmt"
	"os"
)

type PostmanCollection struct {
	Info      PostmanInfo   `json:"info"`
	Items     []PostmanItem `json:"item"`
	Variables []Variable    `json:"variable"`
}

func (pc *PostmanCollection) GetVariableMap() map[string]string {
	varMap := make(map[string]string)
	for _, v := range pc.Variables {
		// Check if an environment variable exists for the variable key
		if envVal, exists := os.LookupEnv(v.Key); exists {
			varMap[v.Key] = envVal
		} else {
			varMap[v.Key] = v.Value
		}
	}
	return varMap
}

func (pc *PostmanCollection) GetSummary() string {
	summary := "Postman Collection: " + pc.Info.Name + "\n"
	summary += "Description: " + pc.Info.Description + "\n\n"
	summary += "Number of Items: " + fmt.Sprintf("%d", len(pc.Items)) + "\n\n"
	summary += "Variables:\n"
	for _, v := range pc.Variables {
		summary += "- " + v.Key + "\n"
	}
	return summary
}

func (pc *PostmanCollection) PrintAll() string {
	details := "Postman Collection Details:\n"
	details += "Name: " + pc.Info.Name + "\n"
	details += "Description: " + pc.Info.Description + "\n"
	details += "Schema: " + pc.Info.Schema + "\n"
	details += "ID: " + pc.Info.Id + "\n"
	details += "Variables:\n"
	for _, v := range pc.Variables {
		details += "- " + v.Key + ": omitted (" + v.Type + ")\n"
	}
	return details
}

type PostmanInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Schema      string `json:"schema"`
	Id          string `json:"_postman_id"`
}

type Variable struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

type PostmanItem struct {
	Name    string         `json:"name"`
	Request PostmanRequest `json:"request"`
}

type PostmanRequest struct {
	Method      string             `json:"method"`
	Url         string             `json:"url"`
	Header      []PostmanHeader    `json:"header"`
	Body        PostmanRequestBody `json:"body"`
	Description string             `json:"description"`
}

type PostmanHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type PostmanRequestBody struct {
	Mode string         `json:"mode"`
	Raw  string         `json:"raw"`
	Form map[string]any `json:"formdata"`
}

// PostmanProcessing defines the interface for processing Postman collections
type PostmanProcessing interface {
	// Generic method to post-process the Postman collection
	PostProcess(collection *PostmanCollection, postProcessor PostProcessor) error
}

// PostProcessor defines the interface for post-processing tasks
// such as variable interpolation and sending HTTP requests.
type PostProcessor interface {
	// Method to interpolate postman variables in strings
	Interpolate(input *string, variables map[string]string) error
	// Method to send HTTP requests based on PostmanRequest
	SendRequest(variableMaps map[string]string, request PostmanRequest, client HTTPClient) error
}

// PostmanParser defines the interface for parsing Postman collections
type PostmanParser interface {
	Parse(jsonCollectionPath string) (*PostmanCollection, error)
}
