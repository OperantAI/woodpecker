package postman

// HumanStatusCode converts an HTTP status code to a human-readable string.
func HumanStatusCode(statusCodeStr int) string {
	switch {
	case statusCodeStr >= 200 && statusCodeStr < 300:
		return "SUCCESS"
	case statusCodeStr >= 400 && statusCodeStr < 500:
		return "CLIENT_ERROR"
	case statusCodeStr >= 500:
		return "SERVER_ERROR"
	default:
		return "OTHER"
	}
}
