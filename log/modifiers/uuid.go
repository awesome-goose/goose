package modifiers

import (
	"github.com/awesome-goose/goose/types"
	"github.com/awesome-goose/goose/utils/props"
	"github.com/awesome-goose/goose/utils/rand"
)

type UUID struct{}

func NewUUID() *UUID {
	return &UUID{}
}

func (m *UUID) Modify(record types.Record) types.Record {
	record.Extra = append(record.Extra, props.Props{
		"id": rand.UUID(),
	})

	return record
}
