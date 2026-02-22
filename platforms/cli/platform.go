package cli

import "github.com/awesome-goose/goose/types"

type Platform struct {
	config *Config
}

func NewPlatform(options ...Option) *Platform {
	config := &Config{
		Name: "cli",
	}

	for _, option := range options {
		option(config)
	}

	return &Platform{config}
}

func (p *Platform) Type() types.PlatformType {
	return types.PlatformTypeCLI
}

func (p *Platform) Name() string {
	return p.config.Name
}

func (p *Platform) Boot(container types.Container) (types.App, error) {
	app := NewApp(p.config)

	return app, nil
}
