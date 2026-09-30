package types

import (
	"context"
	"time"
)

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

// ContextStreamOutput is a StreamOutput whose callback also receives a
// context the kernel cancels when the client disconnects or the kernel begins
// shutting down. A long-lived stream that may sit idle (an event feed
// between events) needs it: write errors only surface on the NEXT write, so
// without a context an idle handler neither notices a dropped client nor can
// be told to stop for a graceful shutdown (which would otherwise wait out the
// platform's whole shutdown timeout for it).
type ContextStreamOutput interface {
	StreamOutput
	// StreamContextCallback is preferred over StreamCallback by the kernel.
	StreamContextCallback() func(ctx context.Context, writer func([]byte) error) error
}

// WriteTimeoutStreamOutput is a StreamOutput that bounds every individual
// write. A stream otherwise has NO write deadline (the kernel clears it so a
// stream can outlive the platform's fixed WriteTimeout), which means a client
// that stops reading blocks the handler's write — and everything the handler
// holds — indefinitely. With a timeout the stalled write fails and the
// handler can return.
type WriteTimeoutStreamOutput interface {
	StreamOutput
	// StreamWriteTimeout is the per-write deadline; <= 0 means none.
	StreamWriteTimeout() time.Duration
}

// RedirectOutput represents redirect responses.
type RedirectOutput interface {
	Output
	// URL returns the redirect destination
	URL() string
	// IsPermanent returns true for 301/308, false for 302/307
	IsPermanent() bool
}
