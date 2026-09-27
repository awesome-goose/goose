package output

import (
	"fmt"
	"strings"
)

// SSEEvent is one Server-Sent Event. ID and Name are optional — a client
// only needs Data to receive a working stream; ID lets it resume with
// Last-Event-ID, and Name lets it dispatch on `addEventListener(Name, ...)`
// instead of the default `message` event.
type SSEEvent struct {
	ID   string
	Name string
	Data []byte
}

// SSE creates a types.StreamOutput that reads events off ch and writes each
// one to the client in the standard text/event-stream wire format, in the
// order received, until ch is closed. Sets Content-Type: text/event-stream
// and Cache-Control: no-cache — every other detail (heartbeats, Last-Event-ID
// replay, reconnect semantics) is the application's job, layered on top of
// this primitive.
func SSE(ch <-chan SSEEvent, opts ...StreamOption) *StreamingOutput {
	callback := func(write func([]byte) error) error {
		for ev := range ch {
			if err := write([]byte(formatSSE(ev))); err != nil {
				return err
			}
		}
		return nil
	}

	base := []StreamOption{
		WithStreamContentType("text/event-stream; charset=utf-8"),
		WithStreamHeaders(map[string]string{
			"Cache-Control": "no-cache",
			// Common reverse-proxy convention (nginx) to disable response
			// buffering for a stream; harmless where it isn't recognized.
			"X-Accel-Buffering": "no",
		}),
	}
	return Stream(callback, append(base, opts...)...)
}

// formatSSE renders one event per the text/event-stream wire format: an
// optional `id:` line, an optional `event:` line, one `data:` line per line
// of ev.Data (multi-line data is valid SSE — each line gets its own `data:`
// prefix), and the blank line that terminates the event.
func formatSSE(ev SSEEvent) string {
	var b strings.Builder
	if ev.ID != "" {
		fmt.Fprintf(&b, "id: %s\n", ev.ID)
	}
	if ev.Name != "" {
		fmt.Fprintf(&b, "event: %s\n", ev.Name)
	}
	lines := strings.Split(string(ev.Data), "\n")
	for _, line := range lines {
		fmt.Fprintf(&b, "data: %s\n", line)
	}
	b.WriteString("\n")
	return b.String()
}
