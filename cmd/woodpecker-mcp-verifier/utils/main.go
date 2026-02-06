package utils

import "errors"
type MCMCPprotocol string

const (
	STDIO MCMCPprotocol = "stdio"
	SSE MCMCPprotocol = "sse"
	STREAMABLE_HTTP MCMCPprotocol = "streamable-http"
)

func (m *MCMCPprotocol) String() string {
	return string(*m)
}

func (m *MCMCPprotocol) Set(value string) error{
	switch value {
	case "stdio", "sse", "streamable-http":
		*m = MCMCPprotocol(value)
		return nil
	default:
		return errors.New(`must be one of "stdio", "sse", "streamable-http"`)
	}
}

func (m *MCMCPprotocol) Type() string {
	return `MCPProtocol, one of "stdio", "sse", "streamable-http"`
}
