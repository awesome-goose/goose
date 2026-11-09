package sql

import "sync"

// Change actions reported to a ChangeObserver.
const (
	ChangeCreate = "create"
	ChangeUpdate = "update"
	ChangeDelete = "delete"
)

// ChangeEvent describes one row an Entity wrote.
//
// Table is the entity's table name (Entity.Name()); Action is ChangeCreate,
// ChangeUpdate or ChangeDelete; Record is the row: the inserted *T (with its
// generated id) for Insert and Upsert, the changes *T passed to Update, and
// the *T that was deleted for Delete by id (for Delete by query, a zero *T
// whose fields are unknown). Query and Args are the WHERE clause of an Update
// or Delete.
type ChangeEvent struct {
	Table  string
	Action string
	Record any
	Query  string
	Args   []any
}

// ChangeObserver hears every ChangeEvent in the process. It runs
// synchronously on the writer's goroutine, after the write (and, inside
// Entity.Tx or Query.Tx, after the transaction commits), so it must be quick
// and must not panic; a handler that does real work starts its own goroutine.
type ChangeObserver func(ChangeEvent)

var (
	observersMu sync.RWMutex
	observers   = map[int]ChangeObserver{}
	observerSeq int
)

// ObserveChanges registers fn for every row written through an Entity
// (Insert, Upsert, Update, Delete) in this process and returns a function that
// removes it. Rows written with gorm directly, or through Query's table-level
// helpers, are not reported.
//
// This is how a host or a framework layer turns writes into domain events
// (scaffold core publishes them as apps.db.<table>.after-insert|update|delete,
// as its TS repository did).
func ObserveChanges(fn ChangeObserver) (remove func()) {
	if fn == nil {
		return func() {}
	}
	observersMu.Lock()
	observerSeq++
	id := observerSeq
	observers[id] = fn
	observersMu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			observersMu.Lock()
			delete(observers, id)
			observersMu.Unlock()
		})
	}
}

func dispatchChanges(events []ChangeEvent) {
	if len(events) == 0 {
		return
	}
	observersMu.RLock()
	if len(observers) == 0 {
		observersMu.RUnlock()
		return
	}
	fns := make([]ChangeObserver, 0, len(observers))
	for id := 1; id <= observerSeq; id++ {
		if fn, ok := observers[id]; ok {
			fns = append(fns, fn)
		}
	}
	observersMu.RUnlock()
	for _, ev := range events {
		for _, fn := range fns {
			fn(ev)
		}
	}
}

func hasObservers() bool {
	observersMu.RLock()
	defer observersMu.RUnlock()
	return len(observers) > 0
}

// changeBuffer holds the events of a transaction until it commits.
type changeBuffer struct {
	mu     sync.Mutex
	events []ChangeEvent
}

func (b *changeBuffer) add(ev ChangeEvent) {
	b.mu.Lock()
	b.events = append(b.events, ev)
	b.mu.Unlock()
}

func (b *changeBuffer) take() []ChangeEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := b.events
	b.events = nil
	return out
}

// emit reports ev now, or holds it until the enclosing Tx commits.
func (r *Query) emit(ev ChangeEvent) {
	if r != nil && r.pending != nil {
		r.pending.add(ev)
		return
	}
	dispatchChanges([]ChangeEvent{ev})
}

// settle hands a committed transaction's events on: to the enclosing
// transaction when this one was nested (a savepoint, so they still depend on
// the outer commit), else to the observers.
func (r *Query) settle(buf *changeBuffer) {
	events := buf.take()
	if r != nil && r.pending != nil {
		for _, ev := range events {
			r.pending.add(ev)
		}
		return
	}
	dispatchChanges(events)
}
