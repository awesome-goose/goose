package log

import (
	"time"

	"github.com/awesome-goose/goose/types"
)

type AppLogChannel string

func (c AppLogChannel) String() string {
	return string(c)
}

type AppLoggers []*Logger

type Log struct {
	channels map[string][]*Logger
	channel  string
}

func NewLog(appChannel AppLogChannel, loggers ...*Logger) *Log {
	channel := appChannel.String()
	if channel == "" {
		channel = "noop"
		loggers = []*Logger{
			NewLogger(
				[]types.Modifier{
					NewNoopModifier(),
				},
				NewNoopFormatter(),
				NewNoopProcessor(),
			),
		}
	}

	l := &Log{}
	l.channels = make(map[string][]*Logger)
	l.channels[channel] = loggers

	return l.Use(channel)
}

func (l *Log) Use(channel string) *Log {
	log := *l
	log.channel = channel
	return &log
}

func (l *Log) Add(channel string, loggers ...*Logger) *Log {
	l.channels[channel] = append(l.channels[channel], loggers...)
	return l.Use(channel)
}

func (l *Log) Debug(message string, extra ...any) {
	l.log(types.Record{
		Datetime: time.Now(),
		Channel:  l.channel,
		Level:    types.DebugLogLevel,
		Message:  message,
		Extra:    extra,
	})
}

func (l *Log) Info(message string, extra ...any) {
	l.log(types.Record{
		Datetime: time.Now(),
		Channel:  l.channel,
		Level:    types.InfoLogLevel,
		Message:  message,
		Extra:    extra,
	})
}

func (l *Log) Notice(message string, extra ...any) {
	l.log(types.Record{
		Datetime: time.Now(),
		Channel:  l.channel,
		Level:    types.NoticeLogLevel,
		Message:  message,
		Extra:    extra,
	})
}

func (l *Log) Warning(message string, extra ...any) {
	l.log(types.Record{
		Datetime: time.Now(),
		Channel:  l.channel,
		Level:    types.WarningLogLevel,
		Message:  message,
		Extra:    extra,
	})
}

func (l *Log) Error(message string, extra ...any) {
	l.log(types.Record{
		Datetime: time.Now(),
		Channel:  l.channel,
		Level:    types.ErrorLogLevel,
		Message:  message,
		Extra:    extra,
	})
}

func (l *Log) Critical(message string, extra ...any) {
	l.log(types.Record{
		Datetime: time.Now(),
		Channel:  l.channel,
		Level:    types.CriticalLogLevel,
		Message:  message,
		Extra:    extra,
	})
}

func (l *Log) Alert(message string, extra ...any) {
	l.log(types.Record{
		Datetime: time.Now(),
		Channel:  l.channel,
		Level:    types.AlertLogLevel,
		Message:  message,
		Extra:    extra,
	})
}

func (l *Log) Emergency(message string, extra ...any) {
	l.log(types.Record{
		Datetime: time.Now(),
		Channel:  l.channel,
		Level:    types.EmergencyLogLevel,
		Message:  message,
		Extra:    extra,
	})
}

func (l *Log) log(records ...types.Record) {
	loggers, ok := l.channels[l.channel]
	if !ok {
		panic("No loggers found for channel: " + l.channel)
	}

	for _, logger := range loggers {
		logger.Write(records...)
	}
}
