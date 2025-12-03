package tests

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/operantai/woodpecker/cmd/woodpecker-postman-collection/postman"
	"github.com/operantai/woodpecker/internal/output"
)

// Mock functions
type httpClientMock struct{}

func (h *httpClientMock) Do(req *http.Request) (*http.Response, error) {

	resp := http.Response{
		Status:     "200 Ok",
		StatusCode: 200,
		Request:    req,
		Header:     req.Header,
		Body:       req.Body,
	}
	return &resp, nil
}

func NewMockHttpClient() postman.HTTPClient {
	return &httpClientMock{}
}

var _ = Describe("Postman Collection Tests", func() {
	Context("A well formatted collection schema", func() {
		It("should run with no errors, client response mocked", func() {
			parser := postman.NewParser()
			collection, err := parser.Parse("./data/example.postman_collection.json")
			if err != nil {
				output.WriteError("error parsing collection: %v", err)
			}
			Expect(err).NotTo(HaveOccurred())
			mockHttpClient := NewMockHttpClient()

			postmanProcessing := postman.NewPostmanProcessing()
			postProcessing := postman.NewPostProcessing()
			if err = postmanProcessing.PostProcess(mockHttpClient, collection, postProcessing); err != nil {
				output.WriteError("error during post-processing: %v", err)
			}
			Expect(err).NotTo(HaveOccurred())
		})
	})

})
