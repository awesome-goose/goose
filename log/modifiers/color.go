package modifiers

import (
	"fmt"

	"github.com/awesome-goose/goose/types"
)

type ColorTagsModifier struct{}

func NewColorTagsModifier() *ColorTagsModifier {
	return &ColorTagsModifier{}
}

func (m *ColorTagsModifier) Modify(record types.Record) types.Record {
	colorStart := ""
	colorEnd := "\033[0m" // Reset code

	switch record.Level {
	case types.DebugLogLevel:
		colorStart = "\033[36m" // Cyan
	case types.InfoLogLevel, types.NoticeLogLevel:
		colorStart = "\033[32m" // Green
	case types.WarningLogLevel:
		colorStart = "\033[33m" // Yellow
	case types.ErrorLogLevel, types.CriticalLogLevel:
		colorStart = "\033[31m" // Red
	case types.AlertLogLevel, types.EmergencyLogLevel:
		colorStart = "\033[1;91m" // Bright red + bold
	default:
		colorStart = ""
		colorEnd = ""
	}

	record.Message = fmt.Sprintf("%s%s%s", colorStart, record.Message, colorEnd)
	return record
}
