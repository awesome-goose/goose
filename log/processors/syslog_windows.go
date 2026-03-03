//go:build windows

package processors

import "github.com/awesome-goose/goose/errors"

type Syslog struct{}

func NewSyslog(tag string) (*Syslog, error) {
	return nil, errors.ErrSyslogNotSupported
}

func (p *Syslog) Process(record []byte) {}
