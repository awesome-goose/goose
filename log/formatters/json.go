package formatters

import (
	"encoding/json"
	"time"

	"github.com/awesome-goose/goose/types"
	"github.com/awesome-goose/goose/utils/props"
)

type JSON struct{}

func NewJSON() *JSON {
	return &JSON{}
}

func (j *JSON) Format(record types.Record) []byte {
	b, err := json.Marshal(record)
	if err != nil {
		fallback := props.Props{
			"time":    time.Now(),
			"channel": "system",
			"level":   types.ErrorLogLevel,
			"message": "log/formatter/json: failed to marshal log record",
		}

		b, _ = json.Marshal(fallback)
	}

	return b
}
