package tests

import (
	"errors"
	"sync"
	"testing"

	"github.com/awesome-goose/goose/modules/sql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type changeWidget struct {
	sql.BaseEntity
	Name   string `gorm:"column:name"`
	Status string `gorm:"column:status"`
}

func (changeWidget) TableName() string { return "ChangeWidgets" }

type changeLog struct {
	mu     sync.Mutex
	events []sql.ChangeEvent
}

func (l *changeLog) add(ev sql.ChangeEvent) {
	l.mu.Lock()
	l.events = append(l.events, ev)
	l.mu.Unlock()
}

func (l *changeLog) take() []sql.ChangeEvent {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := l.events
	l.events = nil
	return out
}

func changeFixture(t *testing.T) (*sql.Entity[changeWidget], *sql.Query, *changeLog) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if raw, err := db.DB(); err == nil {
		raw.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(&changeWidget{}); err != nil {
		t.Fatal(err)
	}
	q := (&sql.Query{}).With(&sql.Db{DB: db})
	e := (&sql.Entity[changeWidget]{}).With(q)
	e.Hydrate("ChangeWidgets", nil, nil, nil, nil, nil, nil, "")
	log := &changeLog{}
	remove := sql.ObserveChanges(log.add)
	t.Cleanup(remove)
	return e, q, log
}

func TestEntityWritesAreReportedToChangeObservers(t *testing.T) {
	e, _, log := changeFixture(t)

	a, b := &changeWidget{Name: "a", Status: "new"}, &changeWidget{Name: "b", Status: "new"}
	if err := e.Insert(a, b); err != nil {
		t.Fatal(err)
	}
	got := log.take()
	if len(got) != 2 || got[0].Action != sql.ChangeCreate || got[0].Table != "ChangeWidgets" {
		t.Fatalf("insert reported %+v", got)
	}
	if w, ok := got[0].Record.(*changeWidget); !ok || w.Id == "" || w.Name != "a" {
		t.Fatalf("insert record = %#v, want the saved row with its id", got[0].Record)
	}

	// An update reports each row it touched, as it is afterwards.
	if n, err := e.Update(&changeWidget{Status: "done"}, "status = ?", "new"); err != nil || n != 2 {
		t.Fatalf("update: %d %v", n, err)
	}
	got = log.take()
	if len(got) != 2 {
		t.Fatalf("update reported %d events, want 2", len(got))
	}
	for _, ev := range got {
		w := ev.Record.(*changeWidget)
		if ev.Action != sql.ChangeUpdate || w.Id == "" || w.Status != "done" || w.Name == "" {
			t.Fatalf("update event %+v / %+v", ev, w)
		}
	}

	// An update that matches nothing reports nothing.
	if _, err := e.Update(&changeWidget{Status: "x"}, "status = ?", "missing"); err != nil {
		t.Fatal(err)
	}
	if got = log.take(); len(got) != 0 {
		t.Fatalf("an empty update reported %+v", got)
	}

	// Delete by id reports the row; delete by query reports each row.
	if _, err := e.Delete(a.Id); err != nil {
		t.Fatal(err)
	}
	got = log.take()
	if len(got) != 1 || got[0].Action != sql.ChangeDelete || got[0].Record.(*changeWidget).Id != a.Id {
		t.Fatalf("delete by id reported %+v", got)
	}
	if _, err := e.Delete("name = ?", "b"); err != nil {
		t.Fatal(err)
	}
	got = log.take()
	if len(got) != 1 || got[0].Record.(*changeWidget).Id != b.Id {
		t.Fatalf("delete by query reported %+v", got)
	}
}

func TestChangesInATransactionAreReportedOnCommitOnly(t *testing.T) {
	e, q, log := changeFixture(t)

	boom := errors.New("boom")
	err := e.Tx(func(tx *sql.Entity[changeWidget]) error {
		if err := tx.Insert(&changeWidget{Name: "rolled back"}); err != nil {
			return err
		}
		if n := len(log.take()); n != 0 {
			t.Fatalf("%d events reported before the commit", n)
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatal(err)
	}
	if got := log.take(); len(got) != 0 {
		t.Fatalf("a rolled-back insert was reported: %+v", got)
	}

	if err := e.Tx(func(tx *sql.Entity[changeWidget]) error {
		return tx.Insert(&changeWidget{Name: "kept"})
	}); err != nil {
		t.Fatal(err)
	}
	if got := log.take(); len(got) != 1 || got[0].Record.(*changeWidget).Name != "kept" {
		t.Fatalf("a committed insert reported %+v", got)
	}

	// Query.Tx: an Entity used on the transaction's Query defers too.
	if err := q.Tx(func(tq *sql.Query) error {
		if err := e.With(tq).Insert(&changeWidget{Name: "via query"}); err != nil {
			return err
		}
		if n := len(log.take()); n != 0 {
			t.Fatalf("%d events reported inside Query.Tx", n)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if got := log.take(); len(got) != 1 {
		t.Fatalf("Query.Tx commit reported %d events, want 1", len(got))
	}
}

func TestRemovedObserverHearsNothing(t *testing.T) {
	e, _, log := changeFixture(t)
	other := &changeLog{}
	remove := sql.ObserveChanges(other.add)
	remove()
	remove() // idempotent
	if err := e.Insert(&changeWidget{Name: "x"}); err != nil {
		t.Fatal(err)
	}
	if len(other.take()) != 0 || len(log.take()) != 1 {
		t.Fatal("a removed observer was called, or the live one was not")
	}
}
