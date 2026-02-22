package modifiers

import (
	"runtime/debug"

	"github.com/awesome-goose/goose/types"
	"github.com/awesome-goose/goose/utils/props"
)

type StackTrace struct{}

func NewStackTrace() *StackTrace {
	return &StackTrace{}
}

func (m *StackTrace) Modify(record types.Record) types.Record {
	trace := debug.Stack()
	record.Extra = append(record.Extra, props.Props{
		"stack": string(trace),
	})
	return record
}
