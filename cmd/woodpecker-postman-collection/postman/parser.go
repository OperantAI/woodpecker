package postman

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/operantai/woodpecker/internal/output"
)

type jsonPostmanParser struct{}

// Factory method to create a new Postman parser
func NewParser() PostmanParser {
	return &jsonPostmanParser{}
}

// Implementation of Parse method that reads and unmarshals a Postman collection JSON file
func (p *jsonPostmanParser) Parse(jsonCollectionPath string) (*PostmanCollection, error) {
	// Read the JSON file
	jsonData, err := os.ReadFile(jsonCollectionPath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}
	var collection PostmanCollection
	err = json.Unmarshal(jsonData, &collection)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %v", err)
	}
	return &collection, nil
}

type postmanProcessing struct{}

// Factory method to create a new Postman processing instance
func NewPostmanProcessing() PostmanProcessing {
	return &postmanProcessing{}
}

// Implementation of PostProcess method that post-processes the Postman collection
func (p *postmanProcessing) PostProcess(client HTTPClient, collection *PostmanCollection, postProcessor PostProcessor) error {
	// Processing each item in the collection
	// First we interpolate the URL and body payloads with the necessary variables
	// Then we send the HTTP request using the BaseHTTPClient singleton
	err := processItems(client, collection.Items, collection, postProcessor)

	return err
}

// processItems will process recursively all nested items of a postman collection and send their request
func processItems(client HTTPClient, postmanItems []PostmanItem, collection *PostmanCollection, postProcessor PostProcessor) error {
	for _, item := range postmanItems {
		output.WriteInfo("processing item: %s", item.Name)

		if len(item.Items) > 0 {
			// Recursive call to nested items for processing
			if err := processItems(client, item.Items, collection, postProcessor); err != nil {
				return err
			}
		}
		// Check if current item has request objects to be run
		if !reflect.DeepEqual(item.Request, PostmanRequest{}) {
			if err := postProcessor.Interpolate(item.Request.URL.GetRaw(), collection.GetVariableMap()); err != nil {
				return fmt.Errorf("error interpolating URL %s: %v", *item.Request.URL.GetRaw(), err)
			}
			if err := postProcessor.Interpolate(&item.Request.Body.Raw, collection.GetVariableMap()); err != nil {
				return fmt.Errorf("error interpolating Body payload: %v", err)
			}
			if err := postProcessor.SendRequest(collection.GetVariableMap(), item.Request, client); err != nil {
				return fmt.Errorf("error sending request for item %s: %v", item.Name, err)
			}
		}
	}
	return nil
}

type postProcessing struct{}

// Factory method to create a new Postman post-processing instance
func NewPostProcessing() PostProcessor {
	return &postProcessing{}
}

// Implementation of variable interpolation method based on the variables of the Postman collection
func (p *postProcessing) Interpolate(input *string, variables map[string]string) error {
	// Simple variable substitution using regex
	re := regexp.MustCompile(`\{\{(\w+)\}\}`)
	matches := re.FindAllStringSubmatch(*input, -1)
	for _, match := range matches {
		if val, ok := variables[match[1]]; ok {
			*input = strings.ReplaceAll(*input, match[0], val)
		} else {
			return fmt.Errorf("no value found for variable: %s", match[1])
		}
	}
	return nil
}

// Implementation of sending HTTP request based on PostmanRequest
func (p *postProcessing) SendRequest(variableMaps map[string]string, request PostmanRequest, client HTTPClient) error {
	req, err := http.NewRequest(request.Method, *request.URL.GetRaw(), bytes.NewBuffer([]byte(request.Body.Raw)))
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %v", err)
	}
	// Set headers and interpolate their values with the necessary variables
	for _, header := range request.Header {
		if err := p.Interpolate(&header.Value, variableMaps); err != nil {
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
	checkStatusCode(resp.StatusCode)
	output.WriteInfo("Method: %s, URL: %s,\n Response: %s\n", request.Method, *request.URL.GetRaw(), string(body))

	return nil
}

func RunCollection(collectionPath string) {
	output.WriteInfo("running Postman collection from path: %s", collectionPath)
	parser := NewParser()
	collection, err := parser.Parse(collectionPath)

	if err != nil {
		output.WriteError("error parsing collection: %v", err)
		return
	}
	client := NewBaseHTTPClient()

	postmanProcessing := NewPostmanProcessing()
	postProcessing := NewPostProcessing()
	if err := postmanProcessing.PostProcess(client, collection, postProcessing); err != nil {
		output.WriteError("error during post-processing: %v", err)
		return
	}
}
