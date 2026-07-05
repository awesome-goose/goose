<p align="center">
  <strong>🪿 Goose Framework</strong><br>
  <sub>Modular • Scalable • Multi-Platform</sub>
</p>

---

# Goose Framework

A modular Go framework for building API, Web, SPA, and CLI applications from
a single codebase, with DI, declarative routing, and SQL/cache/cron/queue
modules that all share one database connection.

---

## Features

- **Multi-Platform** – Build API, Web, SPA, and CLI apps from one module tree.
- **Modular Architecture** – Compose code with `Imports / Exports / Declarations`.
- **Dependency Injection** – `inject:""` tag, zero-value-resolves-to-instance.
- **Database Support** – GORM-based SQL module for Postgres, MySQL, and SQLite.
- **Built-in Modules** – `cache`, `cron`, `queues`, `kv`, `router`.
- **Layered Logging** – formatters + modifiers + processors composed at boot.
- **Env / Config** – `env.NewEnv()` plus a YAML-tree `config` package.

---

## Installation

```bash
go get github.com/awesome-goose/goose
```

---

## Quick Start

### API application

```go
package main

import (
    "os"

    "myapp/app"

    "github.com/awesome-goose/goose"
    "github.com/awesome-goose/goose/log"
    "github.com/awesome-goose/goose/log/formatters"
    "github.com/awesome-goose/goose/log/modifiers"
    "github.com/awesome-goose/goose/log/processors"
    "github.com/awesome-goose/goose/platforms/api"
    "github.com/awesome-goose/goose/types"
)

func main() {
    platform := api.NewPlatform(
        api.WithName("my-api"),
        api.WithHost("localhost"),
        api.WithPort(8080),
    )

    initializers := []func(types.Container) error{
        func(c types.Container) error {
            return c.Register(func() types.Log {
                return log.NewLog(
                    log.AppLogChannel("std"),
                    log.NewLogger(
                        []types.Modifier{
                            modifiers.NewUUID(),
                            modifiers.NewColorTagsModifier(),
                            modifiers.NewSystemInfo(),
                            modifiers.NewStackTrace(),
                        },
                        formatters.NewSyslog("my-api", os.Getpid()),
                        processors.NewConsole(),
                    ),
                )
            }, "", true)
        },
    }

    stop, err := goose.Start(
        goose.API(platform, &app.AppModule{}, initializers),
    )
    if err != nil {
        panic(err)
    }
    defer stop()
}
```

### Multi-platform application

```go
package main

import (
    "github.com/awesome-goose/goose"
    "github.com/awesome-goose/goose/platforms/api"
    "github.com/awesome-goose/goose/platforms/cli"
    "github.com/awesome-goose/goose/platforms/web"
)

func main() {
    stop, err := goose.Start(
        goose.API(apiPlatform, apiModule, apiInitializers),
        goose.Web(webPlatform, webModule, webInitializers),
        goose.CLI(cliPlatform, cliModule, cliInitializers),
    )
    if err != nil {
        panic(err)
    }
    defer stop()
}
```

When more than one instance is registered, API/Web run concurrently in the
background and CLI runs only when the binary is invoked with the `cli`
sub-argument.

---

## Project Structure

```
goose/
├── index.go              # goose.Start / goose.API / goose.Web / goose.SPA / goose.CLI
├── config/               # YAML-tree config (config.NewConfig)
├── core/                 # kernel, DI container, router, registry
├── env/                  # env.NewEnv + sources (OS, .env file)
├── errors/               # Typed error catalogue
├── io/                   # Input parsing and output (JSON/HTML/Console)
├── log/                  # log.NewLog / log.NewLogger + formatters/modifiers/processors
├── modules/              # Built-in modules
│   ├── cache/            # SQL-backed cache (Get/Set/Remember[T])
│   ├── cron/             # Scheduled jobs
│   ├── kv/               # SQL-backed key-value store
│   ├── queues/           # SQL-backed job queue
│   ├── router/           # Routing utilities
│   └── sql/              # GORM database (Postgres/MySQL/SQLite)
├── platforms/            # api / web / spa / cli
├── testing/              # Suite runner, assertions, mocks
├── types/                # Public interfaces (Module, Route, Context, ...)
└── utils/                # Small helpers (path, rand, string, slice)
```

---

## Core Concepts

### Modules

A module declares what it brings in, what it makes available to the rest of
the app, and which services it owns:

```go
type AppModule struct{}

func (m *AppModule) Imports() []types.Module {
    return []types.Module{
        sql.Root(&sql.Config{Dialect: "sqlite", Name: "app.db", Sync: true}),
        cache.NewModule(&cache.Config{DefaultTTL: time.Hour}, false),
    }
}

func (m *AppModule) Exports() []any {
    return []any{&AppService{}}
}

func (m *AppModule) Declarations() []any {
    return []any{
        &AppController{},
        &AppService{},
    }
}
```

### Controllers

Controllers define routes and inject the services they need:

```go
type AppController struct {
    Service *AppService `inject:""`
}

func (c *AppController) Routes() types.Routes {
    return types.Routes{
        {Method: "GET", Path: "/",         Handler: c.Index},
        {Method: "GET", Path: "/users/:id", Handler: c.GetUser},
    }
}

func (c *AppController) Index(ctx types.Context) any {
    return map[string]string{"message": "Hello, Goose!"}
}
```

### Services

```go
type AppService struct {
    DB    *sql.Db      `inject:""`
    Cache *cache.Cache `inject:""`
}

func (s *AppService) GetUser(id string) (*User, error) {
    var user User
    return &user, s.DB.First(&user, "id = ?", id).Error
}
```

### Dependency Injection

Use the `inject:""` tag for automatic injection of pointer-to-struct or
interface fields. Tag values control naming:

- `inject:""`  – type-only, single registration
- `inject:"name"` – named registration
- `inject:"type"` / `inject:"name"` – explicit name strategy

```go
type MyController struct {
    UserService *UserService `inject:""`
    Logger      types.Log    `inject:""`
}
```

---

## Built-in Modules

### SQL Module

```go
sql.Root(&sql.Config{
    Dialect:  "postgres", // "postgres" | "mysql" | "sqlite"
    Host:     "localhost",
    Port:     5432,
    Name:     "myapp",
    User:     "postgres",
    Pass:     "secret",
    SSLMode:  "disable",
    Schema:   "public",
    TimeZone: "UTC",
    Sync:     true, // auto-migrate registered entities
})
```

`sql.Root(cfg)` is shorthand for `sql.NewModule(cfg, true)`; use
`sql.Child(cfg)` (or `sql.NewModule(cfg, false)`) to reuse the root
connection inside a feature module.

### Cache Module

```go
cache.NewModule(&cache.Config{
    Group:           "app",
    DefaultTTL:      time.Hour,
    CleanupInterval: 10 * time.Minute,
}, false)
```

The cache is SQL-backed and shares the connection registered by the SQL
module. Use `cache.Remember[T]` / `cache.GetAs[T]` for typed access.

### Cron Module

```go
cron.NewModule(&cron.Config{
    TickInterval: time.Minute,
    Timezone:     "Etc/UTC",
}, handlers, false)
```

Where `handlers` is a `[]*cron.CronHandler` built via `cron.NewHandler` /
`cron.NewSimpleHandler` / `cron.NewTypedHandler[T]`.

### Queues Module

```go
queues.Root(queues.DefaultConfig(), jobHandlers...)
```

`jobHandlers` are `*queues.JobHandler` values. Worker counts are
per-handler via `.WithMinWorkers(n).WithMaxWorkers(n)` — there is no
`Workers` field on the module config.

### KV Module

```go
kv.NewModule(&kv.Config{
    Group:           "app",
    DefaultTTL:      0,           // 0 = no expiration
    CleanupInterval: time.Hour,
}, false)
```

`*kv.KV` exposes `Get/Set/SetNX/GetSet/Del/TTL/Expire/Persist/Exists/Keys/Incr/IncrBy`.
It is not a Redis client — it persists through the SQL module's connection.

---

## Platforms

### API Platform

```go
api.NewPlatform(
    api.WithName("my-api"),
    api.WithHost("0.0.0.0"),
    api.WithPort(8080),
)
```

### Web Platform

```go
web.NewPlatform(
    web.WithName("my-web"),
    web.WithPort(3000),
)
```

### SPA Platform

One HTTP service that serves JSON API routes under an API prefix (default
`/api`) and a single-page app's static assets for every other path, with an
`index.html` fallback for client-side routing.

```go
spa.NewPlatform(
    spa.WithName("my-spa"),
    spa.WithPort(8080),
    spa.WithStaticDir("public"),   // built frontend assets
    spa.WithIndexFile("index.html"),
    spa.WithAPIPrefix("/api"),
)
```

Routes are declared exactly like an API app — without the prefix. A route
registered as `GET /users` is served at `GET /api/users`.

### CLI Platform

```go
cli.NewPlatform(
    cli.WithName("my-cli"),
)
```

---

## Configuration

### Environment variables

`env.NewEnv()` auto-loads from the OS environment and a `.env` file in the
working directory:

```go
import "github.com/awesome-goose/goose/env"

e := env.NewEnv()

host := e.GetWithDefault("HOST", "localhost")
port := e.GetInt("PORT")
debug := e.GetBool("DEBUG")
```

Methods on `*env.Env`: `Get(key) string`, `Set(key, value)`,
`GetWithDefault(key, default) string`, `GetInt(key) int`,
`GetBool(key) bool`, `GetFloat(key) float64`, plus
`FromSources(...types.EnvSource)`.

### YAML config

```go
import "github.com/awesome-goose/goose/config"

cfg, err := config.NewConfig("./config") // reads every *.yaml/*.yml in dir
if err != nil {
    panic(err)
}

cfg.Tree() // map keyed by file basename
cfg.Dir()  // absolute path to the loaded directory
```

The kernel injects an `*config.Config` if you register it from an
initializer.

---

## Logging

Goose logs flow through three composable pieces:

1. **Modifiers** enrich the record (UUID, colors, stack traces, system info)
2. **Formatter** turns the record into bytes (Line, JSON, Syslog)
3. **Processor** writes the bytes (Console, File, Syslog)

```go
logger := log.NewLog(
    log.AppLogChannel("std"),
    log.NewLogger(
        []types.Modifier{
            modifiers.NewUUID(),
            modifiers.NewColorTagsModifier(),
            modifiers.NewSystemInfo(),
            modifiers.NewStackTrace(),
        },
        formatters.NewSyslog("my-api", os.Getpid()),
        processors.NewConsole(),
    ),
)

logger.Info("Server started", "port", 8080)
logger.Error("Failed to connect", "error", err)
```

Add additional channels with `logger.Add("file", anotherLogger)` and switch
between them via `logger.Use("file")`.

---

## Utilities

The `utils` package is a small collection of sub-packages, not a single
namespaced object:

- `utils/path` – `UserHome`, `AppRoot`, `CurrentDir`, plus typed
  app-relative helpers (`Config`, `App`, `Database`, `Lang`, `Public`,
  `Assets`, `Storage`).
- `utils/rand` – `UUID()`.
- `utils/string` – `IsValidHTTPMethod(s)`, `SplitPath(p)`, `Split(s, sep)`.
- `utils/slice` – currently empty (reserved for future helpers).

For string-case conversions or generic slice operations, use the standard
library or `golang.org/x/text/cases`.

---

## Testing

Goose ships a small testing package with fluent assertions, mocks, and a
suite runner.

### Running tests

```bash
go test ./tests/...
go test ./tests/... -v
go test ./tests/... -run TestRouter
go test ./tests/... -run TestRouter/TestFind_SimpleRouteMatch
```

### Writing tests

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
    // before each test
}

func (s *MySuite) TeardownTest() {
    // after each test
}

func (s *MySuite) TestSomething() {
    s.T.Expect("hello").ToEqual("hello")
    s.T.Expect(42).Not().ToEqual(0)
    s.T.Expect([]int{1, 2, 3}).ToHaveLength(3)
    s.T.Expect(err).ToBeNil()
}
```

### Available assertions

```go
// Equality
s.T.Expect(actual).ToEqual(expected)
s.T.Expect(actual).Not().ToEqual(unexpected)
s.T.Expect(actual).ToDeepEqual(expected)
s.T.Expect(actual).ToBe(expected) // identity for pointers

// Nil
s.T.Expect(value).ToBeNil()
s.T.Expect(value).Not().ToBeNil()

// Boolean
s.T.Expect(condition).ToBeTrue()
s.T.Expect(condition).ToBeFalse()

// Length / emptiness
s.T.Expect(slice).ToHaveLength(3)
s.T.Expect(slice).ToBeEmpty()

// Contains
s.T.Expect(slice).ToContain(element)
s.T.Expect("hello world").ToContainString("world")

// Numeric
s.T.Expect(n).ToBeGreaterThan(0)
s.T.Expect(n).ToBeLessOrEqual(100)
s.T.Expect(n).ToBeBetween(0, 10)

// Strings
s.T.Expect("foo-bar").ToHavePrefix("foo")
s.T.Expect("foo-bar").ToHaveSuffix("bar")
s.T.Expect("abc123").ToMatchRegex(`^\w+$`)

// Types
s.T.Expect(value).ToBeType("string")
s.T.Expect(thing).ToImplement((*io.Reader)(nil))

// Panics
s.T.Expect(func() { panic("x") }).ToPanic()
s.T.Expect(fn).ToPanicWith("expected")

// Maps
s.T.Expect(m).ToHaveKey("id")
s.T.Expect(m).ToHaveKeyValue("status", "ok")
```

### Mocking

```go
type MockUserService struct {
    mock *test.Mock
}

func (m *MockUserService) GetUser(id int) string {
    args := m.mock.Called("GetUser", id)
    if len(args) > 0 {
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
