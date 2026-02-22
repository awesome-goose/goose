package web

import "github.com/awesome-goose/goose/types"

type Platform struct {
	config *Config
}

func NewPlatform(options ...Option) *Platform {
	config := &Config{}

	for _, option := range options {
		option(config)
	}

	return &Platform{config}
}

func (p *Platform) Boot(container types.Container) (types.App, error) {
	app := NewApp(p.config)

	return app, nil
}
