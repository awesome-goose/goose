package sql

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// BUGS #26: work started before the root module's Boot waits for its migrations.
func TestReadyClosesWhenTheRootModuleHasMigrated(t *testing.T) {
	if !IsReady() {
		t.Skip("another root module is pending in this process")
	}
	m := NewModule(&Config{Dialect: "sqlite", Name: ":memory:"}, true)
	if IsReady() {
		t.Fatal("ready before the root module booted")
	}
	wait := Ready()
	m.booted = true // what Boot does after its migrations and seeders
	rootBooted()
	select {
	case <-wait:
	default:
		t.Fatal("Ready did not close after the root module booted")
	}
	if !IsReady() {
		t.Fatal("not ready after boot")
	}
}

type ctorRow struct {
	BaseEntity
	Name string `gorm:"column:name" json:"name"`
}

// Libraries build a Query and an Entity over a *gorm.DB for service-level tests, without DI.
func TestNewQueryAndNewEntity(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Table("ctor_rows").AutoMigrate(&ctorRow{}); err != nil {
		t.Fatal(err)
	}
	e := NewEntity[ctorRow](NewQuery(db))
	e.Hydrate("ctor_rows", nil, nil, nil, nil, nil, nil, "")
	if err := e.Insert(&ctorRow{Name: "a"}); err != nil {
		t.Fatal(err)
	}
	got, err := e.First("name = ?", "a")
	if err != nil || got == nil || got.Name != "a" {
		t.Fatalf("First = %+v %v", got, err)
	}
}
