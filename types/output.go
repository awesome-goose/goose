package types

// Output is the base interface for all response types.
// All handler methods should return an implementation of this interface.
type Output interface {
	// Data returns the response body data
	Data() any
	// Code returns the HTTP status code
	Code() int
	// Headers returns additional HTTP headers to be set on the response
	Headers() map[string]string
	// ContentType returns the MIME type for the response (overrides auto-detection)
	ContentType() string
}

// FileOutput represents responses that serve file content.
// Implementations can provide file paths or raw content for streaming.
type FileOutput interface {
	Output
	// FilePath returns the path to the file to serve (empty if using raw content)
	FilePath() string
	// FileName returns the display name for the file
	FileName() string
	// IsInline returns true for inline display (file), false for download attachment
	IsInline() bool
}

// StreamOutput represents responses with streaming content.
type StreamOutput interface {
	Output
	// StreamCallback provides the callback function for streaming content
	StreamCallback() func(writer func([]byte) error) error
}

// RedirectOutput represents redirect responses.
type RedirectOutput interface {
	Output
	// URL returns the redirect destination
	URL() string
	// IsPermanent returns true for 301/308, false for 302/307
	IsPermanent() bool
}
