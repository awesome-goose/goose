package tests

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/awesome-goose/goose/modules/sql"
	test "github.com/awesome-goose/goose/testing"
)

func TestSQLBaseEntityJSONTags(t *testing.T) {
	test.NewSuiteRunner(t, &SQLBaseEntityJSONTagsSuite{}).Run()
}

type SQLBaseEntityJSONTagsSuite struct {
	test.Suite
}

// Regression test for bug #5 in BUGS.md: TimeAware hardcoded snake_case json
// tags (created_at/updated_at), inconsistent with UUIDAware.Id's own
// "no server-side translation" convention. Every entity built on
// sql.BaseEntity serialized these fields in snake_case no matter how the
// entity's own fields were tagged.
func (s *SQLBaseEntityJSONTagsSuite) TestTimestampsSerializeAsCamelCase() {
	now := time.Now().UTC()
	entity := sql.BaseEntity{}
	entity.CreatedAt = &now
	entity.UpdatedAt = &now

	data, err := json.Marshal(entity)
	s.T.Expect(err).ToBeNil()

	var decoded map[string]any
	err = json.Unmarshal(data, &decoded)
	s.T.Expect(err).ToBeNil()

	s.T.Expect(decoded).ToHaveKey("createdAt")
	s.T.Expect(decoded).ToHaveKey("updatedAt")
	s.T.Expect(decoded).Not().ToHaveKey("created_at")
	s.T.Expect(decoded).Not().ToHaveKey("updated_at")
}

// gorm.DeletedAt has its own MarshalJSON, so this checks the struct tag
// directly rather than relying on marshal output shape.
func (s *SQLBaseEntityJSONTagsSuite) TestSoftDeleteTagIsCamelCase() {
	field, ok := reflect.TypeOf(sql.SoftDeleteAware{}).FieldByName("DeletedAt")
	s.T.Expect(ok).ToBeTrue()

	tag, hasTag := field.Tag.Lookup("json")
	s.T.Expect(hasTag).ToBeTrue()
	s.T.Expect(tag).ToHavePrefix("deletedAt")
}
