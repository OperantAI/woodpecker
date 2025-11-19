package postman

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"

	"github.com/operantai/woodpecker/internal/output"
)

// Interface for Postman parser
type PostmanParser interface {
	Parse(jsonCollectionPath string) (*PostmanCollection, error)
}

type jsonPostmanParser struct{}

// Implementation of Parse method that reads and unmarshals a Postman collection JSON file
func (p *jsonPostmanParser) Parse(jsonCollectionPath string) (*PostmanCollection, error) {
	// Read the JSON file
	jsonData, err := os.ReadFile(jsonCollectionPath)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return nil, err
	}
	var collection PostmanCollection
	err = json.Unmarshal(jsonData, &collection)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return nil, err
	}
	return &collection, nil
}

// Factory method to create a new Postman parser
func NewParser() PostmanParser {
	return &jsonPostmanParser{}
}

type jsonPostmanPostProcessing struct{}

// Implementation of PostProcess method that processes the Postman collection
func (p *jsonPostmanPostProcessing) PostProcess(collection *PostmanCollection) error {
	// Processing each item in the collection
	// First we interpolate the URL and body payloads with the necessary variables
	// Then we send the HTTP request using the BaseHTTPClient singleton
	client := GetHTTPClient()
	for _, item := range collection.Items {
		output.WriteInfo("Processing item: %s", item.Name)
		if err := p.interpolate(&item.Request.Url, collection.GetVariableMap()); err != nil {
			return fmt.Errorf("error interpolating URL %s: %v", item.Request.Url, err)
		}
		if err := p.interpolate(&item.Request.Body.Raw, collection.GetVariableMap()); err != nil {
			return fmt.Errorf("error interpolating Body payload: %v", err)
		}
		if err := p.sendRequest(collection.GetVariableMap(), item.Request, client); err != nil {
			return fmt.Errorf("error sending request for item %s: %v", item.Name, err)
		}
	}
	return nil
}

// Implementation of variable interpolation method based on the variables of the Postman collection
func (p *jsonPostmanPostProcessing) interpolate(input *string, variables map[string]string) error {
	// Simple variable substitution using regex
	re := regexp.MustCompile(`\{\{(\w+)\}\}`)
	matches := re.FindAllStringSubmatch(*input, -1)
	for _, match := range matches {
		if val, ok := variables[match[1]]; ok {
			*input = string(bytes.ReplaceAll([]byte(*input), []byte(match[0]), []byte(val)))
		} else {
			return fmt.Errorf("no value found for variable: %s", match[1])
		}
	}
	return nil
}

// Implementation of sending HTTP request based on PostmanRequest
func (p *jsonPostmanPostProcessing) sendRequest(variableMaps map[string]string, request PostmanRequest, client HTTPClient) error {
	req, err := http.NewRequest(request.Method, request.Url, bytes.NewBuffer([]byte(request.Body.Raw)))
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %v", err)
	}
	// Set headers and interpolate their values with the necessary variables
	for _, header := range request.Header {
		if err := p.interpolate(&header.Value, variableMaps); err != nil {
			return fmt.Errorf("error interpolating header value for key %s: %v", header.Key, err)
		}
		req.Header.Set(header.Key, header.Value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending HTTP request: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}
	output.WriteInfo("%s Method: %s, URL: %s\n, Response: %s\n, Status_Code: %d", HumanStatusCode(resp.StatusCode), request.Method, request.Url, string(body), resp.StatusCode)

	return nil
}

// Factory method to create a new Postman post-processing instance
func NewPostProcessing() PostProcessing {
	return &jsonPostmanPostProcessing{}
}

func RunCollection(collectionPath string) {
	output.WriteInfo("Running Postman collection from path: %s", collectionPath)
	parser := NewParser()
	collection, err := parser.Parse(collectionPath)

	if err != nil {
		output.WriteError("Error parsing collection: %v", err)
		return
	}

	postProcessing := NewPostProcessing()
	if err := postProcessing.PostProcess(collection); err != nil {
		output.WriteError("Error during post-processing: %v", err)
		return
	}
}
