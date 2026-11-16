package sql

import (
	"strings"

	"github.com/awesome-goose/goose/errors"
)

// OrderBy returns a copy of the entity whose First, Last and Find order rows
// by sort instead of the sort given to Hydrate: "created_at ASC, id ASC".
// Each term is a column (checked like every other identifier) and an
// optional ASC or DESC, so a caller can pass a user's choice through without
// writing it into SQL unchecked. Keyset (cursor) pagination needs it: its
// order must match its WHERE clause exactly.
func (r *Entity[T]) OrderBy(sort string) (*Entity[T], error) {
	terms := strings.Split(sort, ",")
	out := make([]string, 0, len(terms))
	for _, term := range terms {
		fields := strings.Fields(term)
		if len(fields) == 0 || len(fields) > 2 || !validIdentifier.MatchString(fields[0]) {
			return nil, errors.ErrInvalidColumnName.WithMeta(strings.TrimSpace(term))
		}
		dir := "ASC"
		if len(fields) == 2 {
			switch strings.ToUpper(fields[1]) {
			case "ASC":
			case "DESC":
				dir = "DESC"
			default:
				return nil, errors.ErrInvalidColumnName.WithMeta(strings.TrimSpace(term))
			}
		}
		out = append(out, fields[0]+" "+dir)
	}
	clone := *r
	clone.sort = strings.Join(out, ", ")
	return &clone, nil
}
