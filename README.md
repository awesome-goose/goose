<p align="center">
  <strong>🪿 Goose Framework</strong><br>
  <sub>Modular • Scalable • Multi-Platform</sub>
</p>

---

# Goose Framework

A modular, scalable Go framework for building modern applications across multiple platforms — API, Web, and CLI.

---

## Features

- **Multi-Platform** - Build API, Web, and CLI applications from a single codebase
- **Modular Architecture** - Organize code into reusable, composable modules
- **Dependency Injection** - Built-in IoC container for clean, testable code
- **Database Support** - GORM-based SQL module with migrations (PostgreSQL, MySQL, SQLite)
- **Built-in Modules** - Cache, Cron, Queues, Key-Value store out of the box
- **Flexible Routing** - Route-based request handling with middleware support
- **Environment Config** - Load configuration from files, environment variables, or structs

---

## Installation

```bash
go get github.com/awesome-goose/goose
```

---

## Quick Start

### API Application

```go
package main

import (
    "github.com/awesome-goose/goose"
    "github.com/awesome-goose/goose/platforms/api"
    "myapp/app"
)

func main() {
    platform := api.NewPlatform(
        api.WithHost("localhost"),
        api.WithPort(8080),
    )

    stop, err := goose.Start(
        goose.API(platform, app.NewModule(), nil),
    )
    if err != nil {
        panic(err)
    }
    defer stop()
}
```

### Multi-Platform Application

```go
package main

import (
    "github.com/awesome-goose/goose"
    "github.com/awesome-goose/goose/platforms/api"
    "github.com/awesome-goose/goose/platforms/web"
    "github.com/awesome-goose/goose/platforms/cli"
)

func main() {
    stop, err := goose.Start(
        goose.API(apiPlatform, apiModule, nil),
        goose.Web(webPlatform, webModule, nil),
        goose.CLI(cliPlatform, cliModule, nil),
    )
    if err != nil {
        panic(err)
    }
    defer stop()
}
```

---

## Project Structure

```
goose/
├── index.go              # Main entry point (Start, API, Web, CLI)
├── config/               # Configuration management
│   ├── config.go         # Config loading and parsing
│   ├── dotted.go         # Dotted key access
│   └── struct.go         # Struct-based config
├── core/                 # Core framework components
│   ├── kernel.go         # Application kernel
│   ├── container.go      # Dependency injection container
│   ├── router.go         # Request router
│   ├── registry.go       # Service registry
│   ├── serializer.go     # Data serialization
│   └── traverser.go      # Module traversal
├── env/                  # Environment variable handling
│   └── sources/          # File and OS env sources
├── errors/               # Error types and handling
├── io/                   # Input/Output handling
│   ├── input/            # Request input parsing
│   ├── output/           # Response outputs (JSON, HTML, Console)
│   └── resources/        # Static resources
├── log/                  # Logging system
│   ├── formatters/       # JSON, Line, Syslog formats
│   ├── modifiers/        # Color, Stack trace, UUID
│   └── processors/       # Console, File, Syslog output
├── modules/              # Built-in modules
│   ├── cache/            # Caching module
│   ├── cron/             # Scheduled tasks
│   ├── kv/               # Key-value store
│   ├── queues/           # Job queues
│   ├── router/           # Routing utilities
│   └── sql/              # Database (GORM)
├── platforms/            # Platform implementations
│   ├── api/              # REST API platform
│   ├── cli/              # CLI platform
│   └── web/              # Web platform
├── types/                # Core type definitions
└── utils/                # Utility functions
```

---

## Core Concepts

### Modules

Modules are the building blocks of a Goose application. Each module defines its imports, exports, and declarations:

```go
type appModule struct{}

func (m *appModule) Imports() []types.Module {
    return []types.Module{
        sql.NewModule(&sql.Config{Driver: "sqlite"}, true),
        cache.NewModule(nil, true),
    }
}

func (m *appModule) Exports() []any {
    return []any{&AppService{}}
}

func (m *appModule) Declarations() []any {
    return []any{
        &AppController{},
        &AppService{},
    }
}
```

### Controllers

Controllers handle requests and define routes:

```go
type AppController struct {
    Service *AppService `inject:""`
}

func (c *AppController) Routes() []types.Route {
    return []types.Route{
        {Method: "GET", Path: "/", Handler: c.Index},
        {Method: "GET", Path: "/users/:id", Handler: c.GetUser},
    }
}

func (c *AppController) Index(ctx *types.Context) {
    ctx.JSON(200, map[string]string{"message": "Hello, Goose!"})
}
```

### Services

Services contain business logic and are injected into controllers:

```go
type AppService struct {
    DB    *sql.Db    `inject:""`
    Cache *cache.Cache `inject:""`
}

func (s *AppService) GetUser(id int) (*User, error) {
    var user User
    err := s.DB.First(&user, id).Error
    return &user, err
}
```

### Dependency Injection

Use the `inject:""` tag for automatic dependency injection:

```go
type MyController struct {
    UserService  *UserService  `inject:""`
    Logger       types.Log     `inject:""`
}
```

---

## Built-in Modules

### SQL Module

Database access with GORM:

```go
sql.NewModule(&sql.Config{
    Driver:   "postgres", // postgres, mysql, sqlite
    Host:     "localhost",
    Port:     5432,
    Database: "mydb",
    Username: "user",
    Password: "pass",
    Migrations: []types.Migration{...},
}, true)
```

### Cache Module

Key-value caching with TTL:

```go
cache.NewModule(&cache.Config{
    DefaultTTL: 3600, // seconds
}, true)
```

### Cron Module

Scheduled task execution:

```go
cron.NewModule(&cron.Config{}, true)
```

### Queues Module

Background job processing:

```go
queues.NewModule(&queues.Config{
    Workers: 5,
}, true)
```

### KV Module

Persistent key-value storage:

```go
kv.NewModule(&kv.Config{}, true)
```

---

## Platforms

### API Platform

REST API with JSON responses:

```go
api.NewPlatform(
    api.WithName("my-api"),
    api.WithHost("0.0.0.0"),
    api.WithPort(8080),
)
```

### Web Platform

Web application with HTML templates:

```go
web.NewPlatform(
    web.WithName("my-web"),
    web.WithPort(3000),
)
```

### CLI Platform

Command-line interface:

```go
cli.NewPlatform(
    cli.WithName("my-cli"),
)
```

---

## Configuration

### Environment Variables

```go
import "github.com/awesome-goose/goose/env"

env.Load(
    env.FromFile(".env"),
    env.FromOS(),
)

port := env.Get("PORT", "8080")
```

### Struct-based Config

```go
import "github.com/awesome-goose/goose/config"

type AppConfig struct {
    Port     int    `env:"PORT" default:"8080"`
    Database string `env:"DATABASE_URL"`
}

var cfg AppConfig
config.Load(&cfg)
```

---

## Logging

```go
import "github.com/awesome-goose/goose/log"

logger := log.NewLogger(
    log.WithFormatter(formatters.JSON()),
    log.WithProcessor(processors.Console()),
    log.WithModifier(modifiers.Color()),
)

logger.Info("Server started", map[string]any{"port": 8080})
logger.Error("Failed to connect", map[string]any{"error": err})
```

---

## Utilities

```go
import "github.com/awesome-goose/goose/utils"

// String utilities
utils.String.Slug("Hello World")     // "hello-world"
utils.String.Capitalize("hello")     // "Hello"

// Slice utilities
utils.Slice.Contains([]int{1, 2, 3}, 2)  // true

// Random utilities
utils.Rand.String(16)                // Random 16-char string

// Path utilities
utils.Path.Join("a", "b", "c")       // "a/b/c"
```

---

## Testing

Goose includes a comprehensive testing package with fluent assertions, mocks, and suite runners.

### Running Tests

```bash
# Run all tests
go test ./tests/...

# Run with verbose output
go test ./tests/... -v

# Run a specific test file
go test ./tests/... -run TestRouter

# Run a specific test case
go test ./tests/... -run TestRouter/TestFind_SimpleRouteMatch
```

### Code Coverage

```bash
# Coverage for all goose packages
go test ./tests/... -coverprofile=coverage.out -coverpkg=./...

# Coverage for specific packages
go test ./tests/... -coverprofile=coverage.out -coverpkg=./config,./core,./env,./errors,./utils/...
```

### Writing Tests

Use the built-in testing utilities:

```go
package tests

import (
    "testing"
    test "github.com/awesome-goose/goose/testing"
)

func TestMyFeature(t *testing.T) {
    test.NewSuiteRunner(t, &MySuite{}).Run()
}

type MySuite struct {
    test.Suite
}

func (s *MySuite) SetupTest() {
    // Runs before each test
}

func (s *MySuite) TeardownTest() {
    // Runs after each test
}

func (s *MySuite) TestSomething() {
    s.T.Expect("hello").ToEqual("hello")
    s.T.Expect(42).Not().ToEqual(0)
    s.T.Expect([]int{1, 2, 3}).ToHaveLength(3)
    s.T.Expect(err).ToBeNil()
}
```

### Available Assertions

```go
// Equality
s.T.Expect(actual).ToEqual(expected)
s.T.Expect(actual).Not().ToEqual(unexpected)

// Nil checks
s.T.Expect(value).ToBeNil()
s.T.Expect(value).Not().ToBeNil()

// Boolean
s.T.Expect(condition).ToBeTrue()
s.T.Expect(condition).ToBeFalse()

// Length
s.T.Expect(slice).ToHaveLength(3)
s.T.Expect(slice).ToBeEmpty()

// Contains
s.T.Expect("hello world").ToContain("world")
s.T.Expect(slice).ToContain(element)
```

### Mocking

```go
type MockUserService struct {
    mock *test.Mock
}

func (m *MockUserService) GetUser(id int) string {
    args := m.mock.Called("GetUser", id)
    if args != nil && len(args) > 0 {
        return args[0].(string)
    }
    return ""
}

func (s *MySuite) TestWithMock() {
    mock := &MockUserService{mock: test.NewMock(s.T.T())}
    mock.mock.On("GetUser", "John")

    result := mock.GetUser(1)

    s.T.Expect(result).ToEqual("John")
    s.T.Expect(mock.mock.WasCalled("GetUser")).ToBeTrue()
}
```

---

## License

MIT License - see LICENSE file for details.
