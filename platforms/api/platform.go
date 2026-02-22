package api

import "github.com/awesome-goose/goose/types"

type Platform struct {
	config *Config
}

func NewPlatform(options ...Option) *Platform {
	config := &Config{
		Name: "api",
		Host: "localhost",
		Port: 8080,
	}

	for _, option := range options {
		option(config)
	}

	return &Platform{config}
}

func (p *Platform) Type() types.PlatformType {
	return types.PlatformTypeAPI
}

func (p *Platform) Name() string {
	return p.config.Name
}

func (p *Platform) Boot(container types.Container) (types.App, error) {
	app := NewApp(p.config)

	return app, nil
}
