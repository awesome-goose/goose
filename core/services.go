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
			// YAML config lives in <app root>/config (see docs: Configuration).
			// Pointing at the app root would parse every YAML file under it
			// (assets, media, node_modules) and fail on the first non-config one.
			dir, err := path.Config()
			return config.AppConfigPath(dir), err
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
		func(path config.AppConfigPath) (types.Env, error) {
			return env.NewEnv(), nil
		},
		func(path config.AppConfigPath, _ types.Env) (types.Config, error) {
			return config.NewConfig(path)
		},
		func(channel log.AppLogChannel, loggers log.AppLoggers) (types.Log, error) {
			return log.NewLog(channel, loggers...), nil
		},
	}
)
