package log

import "github.com/awesome-goose/goose/types"

type Logger struct {
	modifiers []types.Modifier
	formatter types.Formatter
	processor types.Processor
}

func NewLogger(modifiers []types.Modifier, formatter types.Formatter, processor types.Processor) *Logger {
	return &Logger{modifiers, formatter, processor}
}

func (c *Logger) Write(records ...types.Record) {
	for _, record := range records {
		// Apply each modifier and use the returned modified record
		for _, modifier := range c.modifiers {
			record = modifier.Modify(record)
		}

		c.processor.Process(c.formatter.Format(record))
	}
}
