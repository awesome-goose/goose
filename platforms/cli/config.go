package cli

import "github.com/awesome-goose/goose/types"

type Config struct {
	Name        string
	Version     string
	Author      string
	Description string

	Routes  types.Routes
	Host    string
	Port    int
	Timeout int
}

type Option func(*Config)

// WithName sets the name of the .
func WithName(name string) Option {
	return func(Config *Config) {
		Config.Name = name
	}
}

// WithVersion sets the version of the .
func WithVersion(version string) Option {
	return func(Config *Config) {
		Config.Version = version
	}
}

// WithAuthor sets the author of the .
func WithAuthor(author string) Option {
	return func(Config *Config) {
		Config.Author = author
	}
}

// WithDescription sets the description of the .
func WithDescription(description string) Option {
	return func(Config *Config) {
		Config.Description = description
	}
}

func WithRoutes(routes ...types.Routes) Option {
	return func(Config *Config) {
		rs := make(types.Routes, 0)
		for _, r := range routes {
			rs = append(rs, r...)
		}

		Config.Routes = rs
	}
}

func WithHost(host string) Option {
	return func(Config *Config) {
		Config.Host = host
	}
}

func WithPort(port int) Option {
	return func(Config *Config) {
		Config.Port = port
	}
}

func WithTimeout(timeout int) Option {
	return func(Config *Config) {
		Config.Timeout = timeout
	}
}
