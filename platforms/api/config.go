package api

type Config struct {
	Name        string
	Version     string
	Author      string
	Description string

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
