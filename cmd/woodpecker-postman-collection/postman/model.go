package postman

import (
	"encoding/json"
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
	Request PostmanRequest `json:"request,omitempty"`
	Items   []PostmanItem  `json:"item,omitempty"` // allows recursive n items = multiple nested dirs
}

type PostmanRequest struct {
	Method      string             `json:"method"`
	URL         URL                `json:"url"`
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

type URLObject struct {
	Raw      string          `json:"raw,omitempty"`
	Protocol string          `json:"protocol,omitempty"`
	Host     URLHost         `json:"host,omitempty"`
	Path     URLPath         `json:"path,omitempty"`
	Port     string          `json:"port,omitempty"`
	Query    []URLQueryParam `json:"query,omitempty"`
	Hash     string          `json:"hash,omitempty"`
	Variable []URLVariable   `json:"variable,omitempty"`
}

type URLHost []string

type URLPath []any // strings or objects with {type, value}

type URLQueryParam struct {
	Key         *string `json:"key,omitempty"`
	Value       *string `json:"value,omitempty"`
	Disabled    bool    `json:"disabled,omitempty"`
	Description string  `json:"description,omitempty"`
}

type URLVariable struct {
	ID    string `json:"id,omitempty"`
	Value string `json:"value,omitempty"`
}

type URL struct {
	IsString bool       // if the field was a string
	String   string     // the raw string form
	Object   *URLObject // parsed object form
}

func (u URL) GetRaw() *string {
	if u.IsString {
		return &u.String
	}
	if u.Object != nil {
		return &u.Object.Raw
	}
	return nil
}

// Custom polimorphic parsing of URL
func (u *URL) UnmarshalJSON(data []byte) error {
	// First: check if it's a JSON string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		u.IsString = true
		u.String = s
		u.Object = nil
		return nil
	}

	// Otherwise: it's an object
	var obj URLObject
	if err := json.Unmarshal(data, &obj); err == nil {
		u.IsString = false
		u.Object = &obj
		return nil
	}

	return fmt.Errorf("url must be string or object, got: %s", string(data))
}

// PostmanProcessing defines the interface for processing Postman collections
type PostmanProcessing interface {
	// Generic method to post-process the Postman collection
	PostProcess(client HTTPClient, collection *PostmanCollection, postProcessor PostProcessor) error
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
