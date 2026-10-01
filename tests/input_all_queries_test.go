package tests

import (
	"testing"

	"github.com/awesome-goose/goose/io/input"
	test "github.com/awesome-goose/goose/testing"
)

type allQueriesDto struct {
	ID      string            `param:"id"`
	Queries map[string]string `queries:"all"`
}

// A list endpoint filters on whatever the caller put in the query string
// (`?project_id=…&perPage=5&query=…`); the fields it can name are not known to
// the DTO, so the binder must be able to hand the whole map over. Without it the
// scaffold's CRUD lists ignored every filter, search and page size.
func TestInput_AllQueriesTag_DeliversEveryQueryParameter(t *testing.T) {
	ctx := test.NewMockContext()
	ctx.MockRequest().WithParams(map[string]string{"id": "x"}).WithQueries(map[string]string{"project_id": "p1", "perPage": "5", "query": "bread"})
	var dto allQueriesDto
	if err := input.NewInput(ctx).Populate(&dto); err != nil {
		t.Fatal(err)
	}
	if dto.ID != "x" || len(dto.Queries) != 3 || dto.Queries["project_id"] != "p1" || dto.Queries["perPage"] != "5" || dto.Queries["query"] != "bread" {
		t.Fatalf("got %+v", dto)
	}
}

func TestInput_AllQueriesTag_IsAnEmptyMapNotNilWithNoQuery(t *testing.T) {
	ctx := test.NewMockContext()
	var dto allQueriesDto
	if err := input.NewInput(ctx).Populate(&dto); err != nil {
		t.Fatal(err)
	}
	if dto.Queries == nil {
		t.Fatal("a handler ranging over or reading the map must not meet nil")
	}
}

// The binder gives the handler its own copy: changing it must not change what
// other binders (or a later middleware) see.
func TestInput_AllQueriesTag_IsACopy(t *testing.T) {
	ctx := test.NewMockContext()
	ctx.MockRequest().WithQueries(map[string]string{"a": "1"})
	var dto allQueriesDto
	if err := input.NewInput(ctx).Populate(&dto); err != nil {
		t.Fatal(err)
	}
	dto.Queries["a"] = "changed"
	if got := ctx.Request().Queries()["a"]; got != "1" {
		t.Fatalf("the request's own query map was changed: %q", got)
	}
}
