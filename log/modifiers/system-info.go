package modifiers

import (
	"os"
	"runtime"

	"github.com/awesome-goose/goose/types"
	"github.com/awesome-goose/goose/utils/props"
)

type SystemInfo struct{}

func NewSystemInfo() *SystemInfo {
	return &SystemInfo{}
}

func (m *SystemInfo) Modify(record types.Record) types.Record {
	hostname, _ := os.Hostname()

	info := props.Props{
		"go_version": runtime.Version(),
		"go_os":      runtime.GOOS,
		"go_arch":    runtime.GOARCH,
		"num_cpu":    runtime.NumCPU(),
		"hostname":   hostname,
		"pid":        os.Getpid(),
	}

	record.Extra = append(record.Extra, props.Props{
		"system-info": info,
	})

	return record
}
