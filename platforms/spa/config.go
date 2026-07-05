package spa

type Config struct {
	Name        string
	Version     string
	Author      string
	Description string

	Host    string
	Port    int
	Timeout int

	// StaticDir is the directory served for non-API requests. Relative
	// paths resolve against the working directory of the running binary.
	StaticDir string
	// IndexFile is the SPA entry file within StaticDir served as a
	// fallback for client-side routes.
	IndexFile string
	// APIPrefix is the URL prefix routed to the goose kernel. Routes are
	// declared without the prefix.
	APIPrefix string
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

func WithStaticDir(dir string) Option {
	return func(Config *Config) {
		Config.StaticDir = dir
	}
}

func WithIndexFile(file string) Option {
	return func(Config *Config) {
		Config.IndexFile = file
	}
}

func WithAPIPrefix(prefix string) Option {
	return func(Config *Config) {
		Config.APIPrefix = prefix
	}
}
