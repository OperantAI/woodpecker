package postman

import "github.com/operantai/woodpecker/internal/output"

// checkStatusCode provides human-readable output based on HTTP status codes.
func checkStatusCode(statusCodeStr int) {
	switch {
	case statusCodeStr >= 400 && statusCodeStr < 500:
		output.WriteError("client error occurred: %d", statusCodeStr)
	case statusCodeStr >= 500:
		output.WriteError("server error occurred: %d", statusCodeStr)
	default:
	}
}
