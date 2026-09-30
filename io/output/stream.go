package output

import (
	"context"
	"net/http"
	"time"
)

// StreamingOutput is a generic types.StreamOutput implementation — a
// handler returns one of these to have the kernel write the response body
// incrementally (flushing after every write, with the write deadline
// cleared for the duration) instead of buffering it and writing once.
// Mirrors io/output/file.go's StreamDownloadOutput, minus the
// file-download-specific FileOutput methods (Content-Disposition, etc).
type StreamingOutput struct {
	callback     func(write func([]byte) error) error
	ctxCallback  func(ctx context.Context, write func([]byte) error) error
	code         int
	headers      map[string]string
	contentType  string
	writeTimeout time.Duration
}

// StreamOption configures a StreamingOutput.
type StreamOption func(*StreamingOutput)

// WithStreamCode overrides the default 200 status code.
func WithStreamCode(code int) StreamOption {
	return func(s *StreamingOutput) { s.code = code }
}

// WithStreamHeaders adds custom headers.
func WithStreamHeaders(headers map[string]string) StreamOption {
	return func(s *StreamingOutput) {
		for k, v := range headers {
			s.headers[k] = v
		}
	}
}

// WithStreamWriteTimeout bounds each write to the client: a write that cannot
// complete within d (a peer that stopped reading) fails, ending the stream.
// Unset means no bound.
func WithStreamWriteTimeout(d time.Duration) StreamOption {
	return func(s *StreamingOutput) { s.writeTimeout = d }
}

// WithStreamContentType overrides the content type (default: none — the
// platform's usual detection applies unless a specific type is set here).
func WithStreamContentType(contentType string) StreamOption {
	return func(s *StreamingOutput) { s.contentType = contentType }
}

// Stream creates a types.StreamOutput from a callback: the kernel invokes
// callback once, handing it a write function that writes and flushes a
// chunk to the client immediately. callback controls the pacing — write as
// many chunks as needed, whenever the handler has data ready; returning
// ends the response.
func Stream(callback func(write func([]byte) error) error, opts ...StreamOption) *StreamingOutput {
	s := &StreamingOutput{
		callback: callback,
		code:     http.StatusOK,
		headers:  make(map[string]string),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// StreamContext is Stream for a callback that needs to know when to stop: ctx
// is cancelled when the client disconnects or the kernel starts shutting
// down, even while the callback is blocked waiting and writing nothing.
func StreamContext(callback func(ctx context.Context, write func([]byte) error) error, opts ...StreamOption) *StreamingOutput {
	s := Stream(nil, opts...)
	s.ctxCallback = callback
	return s
}

// Data returns nil — streaming bypasses the normal serialize-then-write path.
func (s *StreamingOutput) Data() any { return nil }

// Code returns the HTTP status code.
func (s *StreamingOutput) Code() int { return s.code }

// Headers returns custom headers.
func (s *StreamingOutput) Headers() map[string]string { return s.headers }

// ContentType returns the content type.
func (s *StreamingOutput) ContentType() string { return s.contentType }

// StreamCallback returns the streaming callback (types.StreamOutput).
func (s *StreamingOutput) StreamCallback() func(write func([]byte) error) error {
	if s.ctxCallback != nil {
		// Platforms/kernels that predate ContextStreamOutput: run without
		// cancellation rather than not at all.
		return func(write func([]byte) error) error {
			return s.ctxCallback(context.Background(), write)
		}
	}
	return s.callback
}

// StreamWriteTimeout returns the per-write deadline (types.WriteTimeoutStreamOutput).
func (s *StreamingOutput) StreamWriteTimeout() time.Duration { return s.writeTimeout }

// StreamContextCallback returns the context-aware callback (types.ContextStreamOutput).
// For a plain Stream it adapts the callback, ignoring ctx.
func (s *StreamingOutput) StreamContextCallback() func(ctx context.Context, write func([]byte) error) error {
	if s.ctxCallback != nil {
		return s.ctxCallback
	}
	return func(_ context.Context, write func([]byte) error) error { return s.callback(write) }
}
