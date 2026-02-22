package core

import (
	"github.com/awesome-goose/goose/config"
	"github.com/awesome-goose/goose/env"
	"github.com/awesome-goose/goose/log"
	"github.com/awesome-goose/goose/log/formatters"
	"github.com/awesome-goose/goose/log/modifiers"
	"github.com/awesome-goose/goose/log/processors"
	"github.com/awesome-goose/goose/types"
	"github.com/awesome-goose/goose/utils/path"
)

var (
	services = []any{
		func() (config.AppConfigPath, error) {
			path, err := path.AppRoot()
			return config.AppConfigPath(path), err
		},
		func() (log.AppLogChannel, error) {
			return log.AppLogChannel("std"), nil
		},
		func() (log.AppLoggers, error) {
			return []*log.Logger{
				log.NewLogger(
					[]types.Modifier{
						modifiers.NewUUID(),
						modifiers.NewColorTagsModifier(),
						modifiers.NewSystemInfo(),
						modifiers.NewStackTrace(),
					},
					formatters.NewJSON(),
					processors.NewConsole(),
				),
			}, nil
		},
		func(path config.AppConfigPath) (types.Config, error) {
			return config.NewConfig(path)
		},
		func(path config.AppConfigPath) (types.Env, error) {
			return env.NewEnv(), nil
		},
		func(channel log.AppLogChannel, loggers []*log.Logger) (types.Log, error) {
			return log.NewLog(channel, loggers...), nil
		},
	}
)
