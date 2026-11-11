package sql

import "sync"

// Ready is closed once every sql module created in this process, root or child, has run its
// migrations and seeders (its Boot), and is closed from the start when there is none. A child
// carries a library's own migrations (the queue tables come from queues' child module), so the
// root alone is not enough. Work that reads the schema from a goroutine started before Boot
// (queue workers, pollers) waits on it, so a first boot on an empty database does not query
// tables that do not exist yet (BUGS #26).
func Ready() <-chan struct{} {
	readyMu.Lock()
	defer readyMu.Unlock()
	if readyPending == 0 {
		return closedReady
	}
	return readyCh
}

var (
	readyMu      sync.Mutex
	readyPending int
	readyCh      = make(chan struct{})
	closedReady  = func() chan struct{} { c := make(chan struct{}); close(c); return c }()
)

// moduleCreated records a module that has not booted yet.
func moduleCreated() {
	readyMu.Lock()
	defer readyMu.Unlock()
	if readyPending == 0 {
		readyCh = make(chan struct{})
	}
	readyPending++
}

// moduleBooted records that a module's migrations and seeders have run.
func moduleBooted() {
	readyMu.Lock()
	defer readyMu.Unlock()
	if readyPending == 0 {
		return
	}
	readyPending--
	if readyPending == 0 {
		close(readyCh)
	}
}

// isClosed reports whether a ready channel is already closed.
func isClosed(c <-chan struct{}) bool {
	select {
	case <-c:
		return true
	default:
		return false
	}
}

// IsReady reports whether Ready is closed now.
func IsReady() bool { return isClosed(Ready()) }
