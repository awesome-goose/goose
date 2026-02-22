package formatters

import (
	"fmt"
	"os"
	"strings"

	"github.com/awesome-goose/goose/types"
)

type Syslog struct {
	AppName string
	PID     int
}

func NewSyslog(appName string, pid int) *Syslog {
	return &Syslog{
		AppName: appName,
		PID:     pid,
	}
}

func (f *Syslog) Format(record types.Record) []byte {
	if f.PID == 0 {
		f.PID = os.Getpid()
	}

	if f.AppName == "" {
		f.AppName = os.Getenv("APP_NAME")
	}

	timestamp := record.Datetime.Format("Jan 2 15:04:05") // RFC3164 style
	hostname, _ := os.Hostname()
	msg := fmt.Sprintf("%s %s %s[%d]: %s",
		timestamp,
		hostname,
		f.AppName,
		f.PID,
		strings.TrimSpace(record.Message),
	)

	return []byte(msg)
}
