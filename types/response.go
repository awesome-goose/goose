package types

type Response interface {
	// Write writes the response body with the given serial type and status code
	Write(serialType SerialType, outputData []byte, statusCode int) error
	// SetHeader sets a single response header
	SetHeader(key, value string)
	// SetHeaders sets multiple response headers
	SetHeaders(headers map[string]string)
	// Raw returns the underlying response writer (platform-specific)
	Raw() any
}
