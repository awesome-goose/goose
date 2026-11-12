# Known bugs

Found while building a real application (sandbox/gox-apps) against this
framework on 2026-07-22. Originally left unfixed here by explicit choice at
the time — downstream consumers worked around them rather than having this
file patched — but all five have since been fixed directly in the framework
(2026-07-23). This file is kept as a record of what broke, why, and what
changed, so downstream consumers know they can drop their workarounds.

---

## 1. Default `types.Log` registration can never resolve — FIXED

**File:** `core/services.go`, lines 23 and 43.

The built-in `services` list registers the logger-slice producer under the
named type `log.AppLoggers`:

```go
func() (log.AppLoggers, error) {
    return []*log.Logger{...}, nil
},
```

but the `types.Log` constructor right after it asked for the *unnamed*
`[]*log.Logger`:

```go
func(channel log.AppLogChannel, loggers []*log.Logger) (types.Log, error) {
    return log.NewLog(channel, loggers...), nil
},
```

`log.AppLoggers` (`type AppLoggers []*Logger`, defined in `log/log.go:15`)
and `[]*log.Logger` are different `reflect.Type` values even though they
share an underlying representation, and `core/container.go`'s binding map
is keyed by exact `reflect.Type`. Since `Container.Register` with
`singleton: true` invokes the resolver **immediately** (not lazily), this
type mismatch means the `Log` constructor's own `container.arguments()`
call failed right there in the `services` registration loop
(`core/kernel.go`'s `runSingle`) — and that loop's `Register` return value
is discarded (`container.Register(fn, "", true)`, no error check), so the
failure was silent. `types.Log` simply never got a binding.

**Symptom:** any app that didn't override logging via a custom
`Initializer` failed to boot the moment anything needed `types.Log` (e.g.
`modules/sql.Configure`'s `container.Resolve(&log, "")`), with:

```
FAILED_TO_RESOLVE_LOG: Failed to resolve log (Could not resolve the logger
from container): NO_CONCRETE_FOUND: No concrete found for the given
abstraction (No implementation has been registered for this type)
```

**Fix applied:** changed the `Log` constructor's parameter type to match
what's actually registered:

```go
func(channel log.AppLogChannel, loggers log.AppLoggers) (types.Log, error) {
    return log.NewLog(channel, loggers...), nil
},
```

`NewLog`'s signature is `func(appChannel AppLogChannel, loggers
...*Logger) *Log` — spreading a `log.AppLoggers` value works identically to
spreading a raw `[]*Logger`, so this was a pure type-annotation fix with no
other code changes needed. Regression test:
`tests/default_services_test.go`.

Apps carrying the sandbox/gox-apps workaround (passing a custom `Log`
Initializer to every `goose.Start(...)` call) no longer need to — the
default `services` chain now resolves `types.Log` on its own — but the
workaround is still harmless to keep if you want a non-default channel or
formatter.

---

## 2. `modules/cron`'s own `Boot` hook can never resolve `*Cron` — FIXED

**File:** `modules/cron/module.go`, `cronModule.Boot`.

```go
func (m *cronModule) Declarations() []any {
    return []any{&Cron{}}
}

func (m *cronModule) Boot(k types.Kernel) error {
    ...
    var cronService *Cron
    err := container.Resolve(&cronService, "")
    if err != nil {
        return err
    }
    ...
}
```

`*Cron` is declared via `Declarations()`, which makes it constructible
*within the module tree* through the registry (`core/registry.go` calls
`r.container.Create(decl)` and tracks the result in its own
`declarationIndex` — this is a different bookkeeping structure from the raw
`Container`'s `bindings` map). It was never passed through
`container.Register(...)`, so `container.Resolve(&cronService, "")` — which
only ever looks at `Container.bindings` — couldn't find it, regardless of
whether `Boot` hooks run in any particular order relative to Traverse.

**Symptom:** any app that registered cron handlers via `cron.Root(...)` (or
`cron.Child(...)`) failed to boot with:

```
NO_CONCRETE_FOUND: No concrete found for the given abstraction (No
implementation has been registered for this type)
  meta="*cron.Cron"
```

**Fix applied:** `cronModule.Boot` now fetches its own declaration via the
registry (`k.Registry().Get(&Cron{})` + type assertion) instead of
`container.Resolve`, mirroring the pattern this bug forced consumers to use
for their own sibling-module lookups. This keeps `*Cron`'s `db`/`log`/
`config` fields correctly populated by the registry's own `Container.Create`
path. A new `errors.ErrDeclarationTypeMismatch` covers the (practically
unreachable, but worth guarding) case where the type assertion fails.
Regression test: `tests/cron_module_test.go`.

`modules/queues` was audited for the same declaration-vs-registration
mistake and turned out to have it too: `queuesModule.Boot` resolved its own
`*Queue` declaration via `container.Resolve(&queue, "")` for exactly the
same reason. Fixed the same way (`k.Registry().Get(&Queue{})` + type
assertion). Regression test: `tests/queues_module_test.go`.

Apps carrying the sandbox/gox-apps workaround (running periodic jobs on a
plain `time.Ticker` instead of `modules/cron`) can now switch to
`modules/cron` if they want its persistence/distributed-locking features.

---

## 3. Multi-instance mode's CLI dispatch argument leaked into the CLI platform's own routing — FIXED

**Files:** `core/kernel.go`'s `runCLI`, `platforms/cli/request.go`'s
`NewRequest`.

When an app combined a CLI instance with one or more server instances
(`goose.Start(goose.API(...), goose.CLI(...))`), the kernel decided which
to run by checking `os.Args[1] == "cli"` (`runMulti`):

```go
cliMode := len(os.Args) > 1 && os.Args[1] == "cli"
if cliMode && cliInstance != nil {
    return k.runCLI(cliInstance)
}
```

`runCLI` didn't strip `"cli"` from `os.Args` before booting the CLI
platform. `platforms/cli.NewRequest()` then built its own request straight
from the *same*, still-unmodified `os.Args`, so `Request.Paths()` — and
therefore route matching — saw `["cli", "install"]` for an invocation of
`myapp cli install`, not `["install"]`. This was silently inconsistent with
every single-instance CLI reference example (`awesome-goose/sandbox/cli`),
where routes are registered as plain `router.Cli("install", ...)` with no
`"cli/"` prefix, because a single CLI instance goes through `runSingle`
instead of `runMulti` and never hit this path at all.

**Symptom:** `ROUTE_NOT_FOUND` for every route in a multi-instance CLI
app's routes, with no indication that the "fix" was to prefix every one of
them with `cli/`.

**Fix applied:** `runCLI` now strips the leading `"cli"` selector from
`os.Args` before booting the CLI platform, so single- and multi-instance
CLI apps share the exact same route registration convention (no `cli/`
prefix needed either way). Regression test: `tests/cli_dispatch_test.go`.

Apps carrying the sandbox/agent workaround (registering every CLI route
under a `cli/` prefix, e.g. `router.Cli("cli/install", ...)`) should drop
the prefix — it will now 404 instead of matching.

---

## 4. CLI `Output.Code()` never reached the actual process exit code — FIXED

**File:** `platforms/cli/response.go` (`Response.Write`/`Response.Code`),
`platforms/cli/app.go` (`App.Run`).

```go
func (r *Response) Write(serialType types.SerialType, data []byte, statusCode int) error {
    var output []byte
    switch serialType { ... }
    if len(output) > 0 {
        _, err := r.raw.Write(output)
        ...
    }
    return nil
}
```

`statusCode` (the controller's returned `Output.Code()` — e.g. what
`output.ConsoleError(...)`/`.WithExitCode(1)` set) was accepted as a
parameter and then never used for anything. Nothing downstream of this —
not `runCLI`, not `main()`'s own return path — read it either.

**Symptom:** a goose CLI app had no way to signal command failure through
the normal `Output`/handler-return mechanism — any shell script, CI step,
or Makefile relying on `$?` to detect a failed command silently got `0`
regardless of what the handler actually returned.

**Fix applied:** `Response.Write` now stores `statusCode` and exposes it via
a new `Response.Code()` method. `App.Run` reads it after the handler
returns and calls `os.Exit(code)` whenever it's nonzero (a
non-2xx/nonzero code exits the process non-zero, mirroring how an HTTP
status code maps to success/failure). Code `0` — the default for
`Console`/`ConsoleSuccess`/`Line`/etc.; only `ConsoleError`/
`ConsoleWithCode`/`.WithExitCode(...)` set anything else, and that
convention was already consistent across `io/output/console.go` — falls
through normally so callers can still run their own deferred cleanup after
`goose.Start(...)` returns. Regression test:
`tests/cli_exit_code_test.go`.

Apps carrying the sandbox/agent workaround (a package-level exit-code
variable read by `main()` after `goose.Start(...)` returns) can drop it —
`os.Exit` now happens directly for nonzero codes, so `main()` won't even
reach code after `goose.Start(...)` in that case.

---

## 5. `sql.BaseEntity`'s embedded `TimeAware`/`SoftDeleteAware` hardcoded snake_case JSON tags — FIXED

**File:** `modules/sql/types.go`, `TimeAware`/`SoftDeleteAware`.

```go
type TimeAware struct {
    CreatedAt *time.Time `gorm:"index;column:created_at;not null" json:"created_at,omitempty" binding:"-"`
    UpdatedAt *time.Time `gorm:"index;column:updated_at;not null" json:"updated_at,omitempty" binding:"-"`
}

type SoftDeleteAware struct {
    DeletedAt gorm.DeletedAt `gorm:"index;column:deleted_at" json:"deleted_at,omitempty" binding:"-"`
}
```

Every entity built on `sql.BaseEntity` (which embeds both) therefore always
serialized these three timestamps as `created_at`/`updated_at`/`deleted_at`
over the wire, regardless of what JSON convention the entity's own fields
used — there was no per-entity way to override this, since Go's
promoted-field JSON marshaling is decided by these embedded structs' own
tags. This was inconsistent with `UUIDAware.Id`, right above it in the same
file, which already used `json:"id"`.

**Symptom:** any goose app whose JSON API consumer expected a single
consistent casing convention (e.g. an Angular frontend that sends/expects
camelCase) got `created_at`/`updated_at`/`deleted_at` back no matter how its
own entity fields were tagged — silently breaking any UI code that read
`payload.createdAt`.

**Fix applied:** `TimeAware`/`SoftDeleteAware`'s json tags are now camelCase
(`createdAt`/`updatedAt`/`deletedAt`), matching `UUIDAware.Id`'s existing
"no server-side translation layer downstream" convention. The `gorm:`
column tags are unchanged (`created_at`/`updated_at`/`deleted_at` remain the
actual DB columns — only the JSON wire format changed). Regression test:
`tests/sql_base_entity_json_test.go`.

Apps carrying the `ngx-apps/projects/spaces` workaround (reading
`payload?.createdAt ?? payload?.created_at` on the frontend) can drop the
`created_at` fallback — the API now always sends `createdAt`.

---

## 6. Module `Boot()` hooks run in non-deterministic order — FIXED

**File:** `core/traverser.go:53-56` (`collectHooks`), `core/registry.go:433-442`
(`ListModules`), `core/registry.go:445-459` (`ListDeclarations`).

Found while building `scrapper` (2026-07-29). `Hydrate` processes modules in
correct topological order — dependencies before dependents, via
`topologicalSort` and the `for _, mod := range sorted { processModule(mod) }`
loop — so `Configure()` hooks and declaration creation are ordered correctly.
But `Boot()` hooks are collected *separately*, afterward, via `collectHooks`:

```go
func (t *traverser) collectHooks() {
	seen := make(map[any]bool)

	for _, module := range t.registry.ListModules() {
		...
```

and `ListModules()` iterates a plain Go map:

```go
moduleRegistry map[types.Module]*resolvedModule
...
func (r *Registry) ListModules() []types.Module {
	...
	for _, rm := range r.moduleRegistry {
		modules = append(modules, rm.module)
	}
	return modules
}
```

Go map iteration order is randomized per the language spec, so the order
`Boot()` hooks actually fire in bears no relationship to the
dependency-respecting order `sorted` (and therefore `Configure()`) already
established. `ListDeclarations` has the identical issue one level down
(`availableDeclarations map[reflect.Type]*types.DeclarationInfo`).

**Symptom:** any module whose `Boot()` both (a) imports another module via
`sql.Child(&sql.Config{Migrations: ...})` and (b) starts using that
dependency's tables immediately in its own `Boot()` — as `modules/queues`
(`queue.Process(handler)`) and `modules/cron` (`go cronService.Start(...)`)
both do — can have its own `Boot()` fire *before* its `sql.Child`
dependency's `Boot()` has run the migration that creates those tables.
Reproduced consistently while running `scrapper run`/`serve` against a real
Postgres — non-deterministic per process invocation (roughly coin-flip odds,
consistent with 2-element map iteration randomization):

```
ERROR: relation "QueueQueues" does not exist (SQLSTATE 42P01)
  SELECT * FROM "QueueQueues" WHERE name = 'scrape' ORDER BY "QueueQueues"."id" LIMIT 1
Failed to register cron job scrapper/tick: ERROR: relation "CronJobs" does not exist (SQLSTATE 42P01)
```

Harmless in scrapper's specific case (the log noise doesn't affect the
actual scrape/persist path, since `run` never touches the queue), but it's a
real correctness gap for any app whose `Boot()`-time logic — not just
migrations — depends on import order.

**Suggested fix:** either (a) have `Hydrate` hand `collectHooks` the already
correctly-ordered `sorted` slice directly instead of routing through
`ListModules()`'s map, or (b) change `moduleRegistry`/`availableDeclarations`
from maps to an order-preserving structure (slice + index map) so iteration
order matches insertion/topological order. No regression test yet — would
need either a fixture with a real dependency chain (`sql.Root` → `sql.Child`
with a migration → a dependent module whose `Boot()` immediately queries the
migrated table) run enough times to catch the non-determinism, or a
deterministic repro that inspects `onBootStack`'s built order directly
instead of relying on the random map to misbehave.

**Fix applied (2026-10-09):** `Registry` now records modules in processing
(topological) order and gives each declaration a creation sequence;
`ListModules`/`ListDeclarations` return that order instead of map order.
`collectHooks` pushes hooks so that, under the stack's LIFO execution,
dependencies boot before the modules that import them and a module's own
declarations boot before the module (shutdown is the exact mirror). Each
instance's hooks are registered once, by the module that owns it.
Regression tests: `core/boot_order_test.go` (dependency order, shutdown
mirror, and 300 repeated traversals that must yield one identical order — it
fails on the old code).

---

## 7. `sql.Config.Sync` / `WithSync` is dead code — FIXED (deprecated)

**File:** `modules/sql/types.go:18` (field), `types.go:68` (`WithSync` option).

```go
type Config struct {
	...
	Sync     bool
	...
}
...
func WithSync(sync bool) Option {
	return func(c *Config) {
		c.Sync = sync
	}
}
```

Found while building `scrapper` (2026-07-29) — needed to know whether
`sql.Config{Sync: true}` would auto-migrate entity structs (the natural
reading of the name, and what the sandbox `api` app's `config.yaml` implies
by exposing it as `config.sql.sync`). Grepped every non-test `.go` file in
`modules/sql/`: `Config.Sync` is written once (`WithSync`) and never read
anywhere else — not in `module.go`'s `initialize()`/`Configure()`/`Boot()`,
not in `entity.go`, not in `runner.go`. There is no `AutoMigrate`-from-
struct-tags path in this module at all; schema changes are exclusively
hand-written `sql.Migration`s run via `Config.Migrations`.

**Symptom:** a consumer setting `Sync: true` (or `WithSync(true)`, or the
sandbox app's `config.sql.sync` key) reasonably expects some auto-migration
behavior and silently gets none — no error, no log, just a config field that
does nothing. `scrapper`'s own `docs/TRD.md` (§6.3) had to call this out
explicitly so nobody there relies on it.

**Suggested fix:** either wire it to something real (e.g. `db.AutoMigrate(...)`
over the module's own declared entities, if that's the intended behavior) or
remove the field/option entirely and let the compiler catch existing
callers — a silently-ignored bool is worse than a missing one.


**Fix applied (2026-10-09):** the field and option are kept for source
compatibility (other projects set them) but marked `Deprecated`, and setting
`Sync: true` now logs a warning at startup, so the no-op is never silent.
Remove the field in the next breaking release.
---

## 8. `modules/sql` built the Postgres DSN without quoting, so an empty password broke the connection — FIXED

**File:** `modules/sql/module.go` (now `modules/sql/dsn.go`).

The DSN was `fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s search_path=%s TimeZone=%s", ...)`
with every value unquoted. In libpq keyword/value syntax an empty value followed by a space
swallows the next token, so `password= dbname=app` makes the driver read `dbname=app` as the
password and connect with an empty database name. The same string also failed TLS
configuration on a blank `sslmode`, corrupted any value containing a space or quote, and gave an
empty `Schema` the table prefix `"."` (`tablePrefix = Schema + "."`), so tables were named
`.QueueJobs`.

**Symptom:** with `Pass: ""` (the zero value, and the default in most configs) the app failed to
boot with `database "<user>" does not exist`, naming the *user* where the database should be.
Found 2026-09-21 running a serve/worker spike against a local trust-auth Postgres.

**Fix applied:** `sql.PostgresDSN` single-quotes and escapes every value (`\` and `'`) and omits
unset fields so the driver applies its defaults; `sql.PostgresSchema` defaults an empty schema to
`public`. Tested by parsing the result with pgx's own `pgconn.ParseConfig` (`tests/sql_dsn_test.go`).
The MySQL DSN was not reviewed.

---

## 9. `SuiteRunner` bound assertions to the parent test, so failures were attributed to the wrong test — FIXED

**File:** `testing/suite.go`.

`SuiteRunner.Run` starts one subtest per `Test*` method but left the suite's `T` bound to the
parent `*testing.T`. A failed `Expect` failed the *parent* while the failing method reported `PASS`,
and a failed `Require` called `FailNow` on the parent from the subtest's goroutine, which Go
forbids (`subtest may have called FailNow on a parent test`) and which could abort or hang the
run.

**Symptom:** `go test -v` showed `--- PASS: TestX/TestFoo` under a `--- FAIL: TestX` whose log
lines named `TestFoo`'s assertion; running one method by name looked green. The overall exit code
was still correct, which is why it went unnoticed.

**Fix applied:** the runner rebinds the suite to each subtest's `T` for the duration of the
method and restores it afterwards. `tests/suite_runner_test.go` runs a deliberately failing
fixture suite in a child process and checks each failure lands on its own method.

---

## 10. `queues` and `cron` had no shutdown hook, so kernel shutdown left their workers running — FIXED

**Files:** `modules/queues/module.go`, `modules/cron/module.go`.

The kernel runs the `Shutdown` hook of every `types.Shutdownable` module and declaration. Neither
module implemented it, and `(*Queue).Shutdown()` has the wrong signature to be collected. Workers
(and the cron ticker) started in `Boot` therefore kept running after the kernel had shut down, and
a job in flight was abandoned. A worker process had to know to call `Queue.Shutdown` itself.

**Symptom:** after `stop()` returned, `Queue.TotalActiveWorkers()` was still above zero.

**Fix applied:** both modules implement `Shutdown(k)` and stop their service through the registry
(`Queue.Shutdown` waits for jobs in flight; `Cron.Stop` waits for running jobs). Added
`Cron.IsRunning()` so the runner's state can be observed. Tests: `TestKernelShutdownStopsTheWorkers`
and `TestKernelShutdownStopsTheRunner`, each confirmed to fail with the hook removed.

---

## 11. Quoting the Postgres DSN's `TimeZone` value broke `gorm.io/driver/postgres` — FIXED

**File:** `modules/sql/dsn.go` (`PostgresDSN`).

Bug 8's fix quoted every DSN value, including `TimeZone`, to survive libpq's keyword/value
parsing. But `gorm.io/driver/postgres.Initialize` re-parses the *raw* DSN a second time with its
own regexp (`timeZoneMatcher = regexp.MustCompile("(time_zone|TimeZone)=(.*?)($|&| )")`) to pull
`TimeZone` out as a startup runtime parameter — ahead of, and independently of, pgx's own (correct)
DSN parsing. That regexp knows nothing about libpq quoting: it captures verbatim up to the next
space/`&`, quote characters included, and hands Postgres literal `'UTC'` instead of `UTC`.

**Symptom:** any app using `gorm.Open(postgres.Open(sql.PostgresDSN(c)), ...)` with a non-empty
`TimeZone` failed to connect with `invalid value for parameter "TimeZone": "'UTC'"` (or
`"'America/New_York'"`, etc.) — a regression introduced by Bug 8's own fix. Empty `TimeZone` was
unaffected (omitted from the DSN entirely). Found 2026-09-22 writing an M0-23 verification test
against a real `gorm.Open` connection, not just pgx's `pgconn.ParseConfig` (which never exercises
gorm's second parser and so never caught this).

**Fix applied:** `TimeZone` is left unquoted in the DSN (with a defensive guard skipping any value
containing a quote, space, or `&`, which no valid IANA zone name does anyway). Every other value
stays quoted per Bug 8. Regression test: `tests/sql_dsn_gorm_test.go`, run against a real Postgres
via `gorm.Open` (not a mock) — confirmed it fails identically to the real-world symptom against the
unfixed code and passes after. `tests/sql_dsn_test.go` (pgx-level, from Bug 8) rerun unchanged, no
regression there.

---

## 12. `env`'s `.env` file source overrode real OS environment variables — FIXED

**File:** `env/sources/file.go` (`fileEnvSource.Load`).

`core/services.go` registers `env.NewEnv()` as a default service on every kernel boot, and
`env.NewEnv()` loads `NewOsEnvSource()` then `NewFileEnvSource()` in that order. `Load` called
`os.Setenv(key, value)` unconditionally for every key found in `.env`, with no check for whether
the real OS environment already defined it — so a `.env` file's value always won over a real
environment variable, backwards from standard dotenv convention (the file exists to *fill in* what
the environment doesn't already set, not to override it).

**Symptom:** any app relying on real environment variables (injected by Kubernetes, a CI secret, a
deployment platform, or a developer's shell) to override a checked-in or accidentally-deployed
`.env` file got silently overridden by the file instead — a genuine correctness and security
concern (a stale or example `.env` shipped alongside a binary can clobber a production secret pulled
from the real environment). Found 2026-09-22 writing an M0-23 verification test: `os.Setenv`
overrides set immediately before booting a goose kernel were clobbered by the real `.env`'s values
before `rootSQLConfig()` read them.

**Fix applied:** `Load` now checks `os.LookupEnv(key)` before setting anything, and skips the
file's value whenever the key is already present in the real environment — including when it's
explicitly set to `""`, since that's a deliberate choice (e.g. "disable this"), not an absence, and
must not be filled in either. Regression test: `tests/env_file_source_test.go`, covering all four
cases (real env wins, file fills in what's unset, explicit empty real value isn't overridden,
missing `.env` file is silently ignored as before).

---

## 13. `web`/`api`/`spa` platforms mapped every kernel error to HTTP 500, including "route not found" — FIXED

**Files:** `platforms/web/app.go`, `platforms/api/app.go`, `platforms/spa/app.go` (all `App.ServeHTTP`/
`serveAPI`).

Each platform's HTTP entry point ran the request through the kernel handler and, on any non-nil
error, wrote `http.StatusInternalServerError` unconditionally:

```go
err := a.fn(ctx)
if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
}
```

`core/router.go` returns `errors.ErrRouteNotFound` for any request path with no matching route —
the one kernel error a client can trigger simply by asking for something that was never mounted, or
was unmounted on purpose. That got the same 500 as an actual internal fault, indistinguishable to a
caller (or a test asserting "removed routes 404").

**Symptom:** `GET /api/apps/<name>` for an app deliberately unmounted from `router.Mount` returned
`500 Internal Server Error` with body `ROUTE_NOT_FOUND: Route not found (...)`, not `404 Not Found`.
Found 2026-09-22 verifying an Origine PLAN task ("unmount six apps; done: removed routes return
404") with a real end-to-end boot test — a stub/mock handler test wouldn't have caught this, since
the mismapping lives in the platform's own error-to-status translation, not routing itself.

**Fix applied:** each platform now maps `errors.ErrRouteNotFound` (checked via `stderrors.Is`) to
`http.StatusNotFound`; every other kernel error keeps the prior `http.StatusInternalServerError`
behavior unchanged — this is deliberately narrow, not a general error-code-to-HTTP-status mapping,
since most of the ~100 other built-in error codes represent boot-time misconfiguration never meant
to reach a live request. `web` and `api` gained an exported `SetHandler` (mirroring `spa`'s existing
one) so tests can drive `ServeHTTP` directly without a live listener. Regression tests:
`tests/web_platform_test.go`, `tests/api_platform_test.go`, and two new cases in
`tests/spa_platform_test.go` (`TestAPIRouteNotFoundIs404`, `TestAPIOtherErrorsStayInternalServerError`)
— each confirmed to fail (500 instead of 404) against the unfixed code.

---

## 14. `platforms/api` wrote a second response after a middleware had already answered — FIXED

**File:** `platforms/api/app.go` (`App.ServeHTTP`).

A middleware may answer a request itself and return an error only to stop the chain; scaffold's
`AuthMiddleware` does exactly this for a 401 (writes its JSON envelope, then returns a sentinel).
`ServeHTTP` treated every non-nil error as "nothing written yet" and called `http.Error`, so the
response got a second `WriteHeader` (`http: superfluous response.WriteHeader call` in the log) and
the error text appended to the JSON body (`{"success":false,...}ROUTE_NOT_FOUND: ...`).

**Fix applied:** the response writer is wrapped to record whether anything was written (keeping
`Flusher`, `Hijacker` and `Unwrap`, which SSE and WebSocket upgrades rely on); when the handler
already responded, the error is not written again. Regression test:
`tests/api_written_response_test.go` — fails against the unfixed code.

---

## 15. `utils/path.AppRoot` failed for a built binary, aborting boot — FIXED

**File:** `utils/path/base.go`.

`AppRoot` only found a directory by walking up from the executable and the working directory
looking for `go.mod`. A deployed binary has no `go.mod` above it, so `AppRoot` returned
`os.ErrNotExist`, the default config service failed to register, and the process exited with
`SERVICE_REGISTRATION_ERROR ... file does not exist`. The root is only used to find optional
resources (`config/`, `lang/`), so this should never be fatal.

**Fix applied:** resolution is now `APP_ROOT`, then the nearest `go.mod` (unchanged), then the
working directory. Regression tests: `tests/path_app_root_test.go`.

---

## 16. `platforms/spa` answered twice after a middleware responded, and bodyless statuses became 500s — FIXED

**Files:** `platforms/spa/app.go` (`serveAPI`), `platforms/api/response.go` (`Response.Write`).

Two related defects, both visible as `http: superfluous response.WriteHeader call` in the log:

1. `spa.serveAPI` repeated the #14 mistake: a middleware that wrote its own 401 and returned an error to
   stop the chain got `http.Error` appended to its body. The api platform was fixed; the SPA platform was not.
2. `Response.Write` wrote the JSON body for every status. For `304`, `204` and `1xx` net/http refuses
   (`ErrBodyNotAllowed`), the error propagated, and the platform turned it into a second 500 response. A
   conditional GET answering `304` therefore logged a fault on every poll.

**Fix applied:** `serveAPI` tracks whether the response started (reusing the package's `trackedWriter`)
and does not write again; `Response.Write` skips the body for statuses that cannot carry one.
Regression tests: `tests/spa_written_response_test.go`, `TestAPIBodylessStatusesDoNotWriteABody`.

---

## 17. A second cron module's handlers never ran — FIXED

**Files:** `modules/cron/cron.go` (`Cron.Start`).

`cron.Child` promises "register additional cron handlers", but every cron module's `Boot` calls `Start` on the
one `*Cron`, and `Start` returned at once when the runner was already running. Whichever cron module booted
second (a `Child` after the `Root`, or the `Root` after a `Child`, since boot-hook order across modules isn't
fixed) had its handlers silently dropped. Seen in Yan: a library's `cron.Child` tick next to the scaffold cron
app's `Root` never fired, and the log still said "Cron runner started with 6 handlers".

**Fix applied:** `Start` on a running runner registers the extra handlers on it (`addHandlers`, logged as "Cron
runner added N handlers"). Additive: `Cron.HasHandler(group, name)`. Regression test:
`TestCronModulesShareOneRunner` (fails without the fix).

Seen while testing it, not fixed here: on SQLite the cron tables' `TIMESTAMP(6)` columns come back as strings
(the driver only parses a declared type of exactly `timestamp`), so `Register` fails to read the job back and
every handler is skipped. Postgres is unaffected. Changing the DDL would alter MySQL's precision and semantics,
so it needs its own decision.

---

## 18. Error sentinels were changed per request (a data race) — FIXED

**Files:** `errors/index.go` (`WithError`, `WithMeta`, `WithDetail`, `WithSuggestedAction`).

The package-level sentinels (`ErrFailedToHydrateInput`, `ErrRecordNotFound`, ...) are shared by every
request, and the `With*` methods set a field on the receiver and returned it. `kernel.go` returns
`errors.ErrFailedToHydrateInput.WithError(err)` per request, so concurrent requests overwrote each other's
wrapped error: a data race, and one request's detail could show up in another's response or log. Found by Yan
(PLAN U16).

**Fix applied:** the `With*` methods return a copy and leave the receiver alone. `errors.Is` still matches
(it compares `Code`), and no caller in Goose, scaffold or Yan used them for their side effect. The old tests
that asserted "returns the same pointer" now assert the copy. Regression test:
`TestSentinelWithIsSafeConcurrently` (reports a DATA RACE under `-race` without the fix).

---

## 19. Nothing reported what an Entity wrote, so no `apps.db.*` event ever fired — FIXED (additive)

**Files:** `modules/sql/changes.go` (new), `modules/sql/entity.go`, `modules/sql/query.go`.

The scaffold libraries were ported from TS, where the repository published `apps.db.<table>.after-insert`,
`after-update` and `after-delete` on every write, and their `Subscriptions` still listen for them (identity
seeds a workspace's roles and alerts on a new device, capital opens a wallet, audit logs). In Go nothing
published them: `sql.Entity` had per-entity hooks only, so there was no process-wide way to see a write. Found
by Yan (PLAN U12): no `identity.user.created`.

**Added:** `sql.ObserveChanges(fn) (remove func())`. Every `Entity.Insert`, `Upsert`, `Update` and `Delete`
reports a `ChangeEvent{Table, Action, Record, Query, Args}` once it succeeds: the inserted row with its id; for
an update, each touched row as it is afterwards (ids noted before the update, rows read after, at most 1000);
for a delete, the deleted row(s). Inside `Entity.Tx` and `Query.Tx` (and an `Entity.With` that Tx's `Query`)
events wait for the commit and are dropped on rollback; a nested Tx hands them to the outer one. With no
observer registered, writes do no extra work. Rows written with gorm directly are not reported. scaffold
`core/events/dbevents` turns these into bus events. Tests: `tests/sql_changes_test.go`.

---

## 20. A wrong app root silently loaded no `.env`, so the app ran on defaults — FIXED (additive)

**Files:** `utils/path/base.go` (`ResolveAppRoot`), `env/sources/file.go`.

`AppRoot` takes the first `go.mod` above the executable. A binary built into another module's directory (a
`bin/` inside some other checkout) therefore resolved to that module, found no `.env` there, and the file
source returned without a word: the app started on defaults, `DB_NAME=yan` included, and ran migrations on a
shared database. Found by Yan (lane 3, 2026-11-09).

**Fix applied (non-breaking):** resolution order is unchanged. `path.ResolveAppRoot()` also says which rule
picked the root. The `.env` loader now (1) warns on stderr when the file is missing, naming the path and the
rule, with the fix (`APP_ROOT`); (2) reads `GOOSE_ENV_FILE` instead of `<root>/.env` when set, and stops the
process (exit 78, EX_CONFIG) if that file is missing; (3) stops the process on a missing file when
`GOOSE_ENV_FILE_REQUIRED=true`; (4) stays quiet with `GOOSE_ENV_FILE=-` (configured by the real environment on
purpose). Products that keep a `.env` at the root or set `APP_ROOT` see no change. Tests:
`env/sources/file_internal_test.go`, `TestResolveAppRootSaysHow`.

---

## 21. JSON numbers above 2^53 were rounded before they reached a DTO — FIXED

**Files:** `io/input/index.go`.

The binder decoded a JSON body into `map[string]any`, so every number went through float64: an `int64`,
`uint64`, `string` or struct field (or a `json.RawMessage` re-marshalled from the map) got
9007199254740993 as 9007199254740992. Money in minor units and chain amounts (wei) were silently changed. Found
by Yan (lane 2).

**Fix applied (non-breaking):** the body is decoded with `UseNumber` (still nothing allowed after the value, as
`json.Unmarshal` required); integer, unsigned, string, `json.Number`, slice, map and struct fields get the exact
value. Fields typed `any`, `map[string]any`, `[]any` or `props.Props` still receive float64s, as before, so
handlers that type-assert `float64` see no change. **Also added:** `headers:"all"` hands a DTO every request
header (`map[string]string` or `map[string][]string`, a copy), for webhook receivers that verify a signature
header. Tests: `tests/input_numbers_headers_test.go`.

---

## 22. Migrations and seeders are recorded by `%T`, once per database — PARTLY ADDRESSED (additive)

**Files:** `modules/sql/runner.go`.

A migration or seeder is recorded as `fmt.Sprintf("%T", m)`, e.g. `*migrations.CreateLogs`: the package name
without its path. Two libraries that each have a `migrations` package with the same type name collide, and the
second one never runs. Seeders run once per database by the same record, so reference data added later (a new
currency, a new topic) never reaches a database that was seeded before. Found by Yan (lane 2).

**Added, not changed:** renaming recorded entries would re-run every existing migration, so `%T` stays the
default. A migration or seeder may implement `Named` (`MigrationName() string`) to choose a qualified name
(`"wallet/seed-currencies@2"`: a new version of reference data under a new name runs once more), or
`Repeatable` (`Repeat() bool`) to run on every boot (idempotent reference data). `sql.RecordName(m)` reports the
name. Test: `TestSQLMigrateNamedAndRepeatable`. **Still open:** existing same-named pairs across libraries are
not detected; a boot-time check that warns on duplicate record names would catch them.

---

## 23. Postgres `timestamptz` values came back in the machine's time zone — FIXED

**Files:** `modules/sql/timezone.go`, `modules/sql/module.go`; `go.mod` (pgx v5.7.1).

`Config.TimeZone` set the session time zone and `NowFunc`, but pgx scans a `timestamptz` into `time.Local`
whatever the session says. On a machine at +01:00 a row's `createdAt` was `2026-11-10T19:28:12+01:00` for
columns created as `timestamptz` and `...18:28:12Z` for plain `timestamp` ones, so one API mixed offsets and
broke a contract that asks for UTC (Yan TRD §7.6). Found by Yan (U25).

**Fix applied (non-breaking):** with a `TimeZone` set, Postgres connections register pgx's `timestamptz`
codec with `ScanLocation` in that zone (the zone `NowFunc` already writes in). The instant never changes,
only the offset it is printed with. No `TimeZone` keeps the old behaviour. Needs pgx v5.7 or later (the
`ScanLocation` field), so Goose now requires v5.7.1. Test: `TestTimestamptzScansInTheConfiguredZone`
(real Postgres, `GOOSE_TEST_DB_HOST`).

---

## 24. The served OpenAPI document named operations by registration order — FIXED (additive)

**Files:** `openapi/build.go`, `openapi/ids.go`.

`operationId` is the handler's method name, de-duplicated by the tag, then by method and path, in the order
routes registered. Two handlers named `List` got different ids from one boot to the next, so a host had to
rewrite every id in a script to commit a stable document, and the document it served still had the old ids
(found by Yan, U25).

**Added:** `Options.OperationID func(method, path, handler string) string` names every operation (a clash is
an error), and `PathOperationID(skip...)` derives the id from the method and path only, title-casing each
word as Python's `str.title()` does: `GET /api/apps/social/posts/{id}` is `getSocialPostsById`. Also added for
typed responses: `Operation.Request` (the JSON body of a handler that reads the body itself),
`ControllerDocumented` (a controller documents its own handler methods, for generic resources whose shared
Dtos cannot name the entity) and `Page[T]` (the shared `{items, next}` page). Without the new options the
document is unchanged. Tests: `openapi/operation_ids_test.go`.

---

## 25. A handler could not learn the client's address — FIXED (additive)

**Files:** `types/clientip.go`, `platforms/api/request.go`, `platforms/web/request.go`.

`types.Request` has headers but no peer address, so nothing behind a Goose server knew who was calling: scaffold
core read the address from `X-Forwarded-For` only (spoofable, and with a lookup that never matched canonical
keys), every identity login-attempt row had an empty address, and each per-address limit was one global bucket
(10 sign-ups an hour for the whole of Yan; one script could block them all). Found by Yan (lane 2).

**Added:** the api, web and spa requests implement `types.RemoteAddresser` (`RemoteAddr()`, net/http's peer);
`types.RemoteAddr(req)` is "" for a platform without one, so the `Request` interface is unchanged.
`types.ClientIP(req, trusted)` is the peer address unless the peer is a trusted proxy; then `X-Forwarded-For`
is read from the right, skipping trusted hops, with `X-Real-Ip` as the fallback. `types.ParsePrefixes` reads
a CIDR list. Headers from an untrusted peer are never believed. Test: `tests/client_ip_test.go`.

---

## 26. Queue workers polled `QueueQueues` before the migrations had made it — FIXED

**Files:** `modules/sql/ready.go`, `modules/sql/module.go`, `modules/queues/queue.go`.

A library that wires its queue handlers before Boot (scaffold notification starts its workers when the pusher is
wired) started polling at once. On a fresh database the root sql module had not migrated yet, so the first boot
logged an ERROR per poll for a missing `"QueueQueues"` table. Found by Yan (lane 3).

**Fix applied:** `sql.Ready()` is closed once every sql module (root and child) has run its migrations and seeders (closed
from the start when there is none); queue workers wait on it before their first poll, and start at once, as
before, when it is already closed. A module that never boots holds them back for at most a minute. Tests:
`TestReadyClosesWhenEverySQLModuleHasMigrated`, the queue suite.

**Also added (lane 3):** `sql.NewQuery(*gorm.DB)` and `sql.NewEntity[T](*Query)`, the supported way to build a
Query or an Entity outside the container, for libraries' service-level tests (`(&sql.Query{}).With(&sql.Db{DB:
db})` worked but was undocumented). Test: `TestNewQueryAndNewEntity`.

---

## 27. `kv` increments were lost under concurrency, and a key could get two rows — FIXED

**Files:** `modules/kv/kv.go`, `modules/kv/entry.go`, `modules/kv/migrations/unique-group-key.go`.

`IncrBy` read the row, added in Go and wrote the sum back inside a transaction with no row lock, and
`("group", "key")` had a plain index, not a unique one. Two increments at once both read `n` and both wrote `n+1`;
two first increments of a missing key each created a row, and later reads picked either. `SetNX` locked the row but
could not lock one that did not exist yet, so two callers on a missing key could both create it. Found by Origine
(U-G11): 40 goroutines × 25 increments ended at a value returned twice, and 4 processes ended at 74 of 1,600.

**Fix applied:** a migration (`kv/unique-group-key`) removes duplicate rows (keeping the active row updated last) and
makes `("group", "key")` unique. `Set`, `SetNX`, `GetSet` and `IncrBy` now take the key's one row with
`INSERT … ON CONFLICT DO NOTHING` (a deleted placeholder when the key is new) and then `SELECT … FOR UPDATE`, so a
second caller waits for the first. A deleted or expired key counts from 0 with no expiry; a live key keeps its expiry.
SQLite skips `FOR UPDATE` (it serializes writers). Tests: `modules/kv/kv_incr_test.go` (goroutines and processes
against Postgres with `GOOSE_TEST_DB_HOST`; unique index; the migration on duplicates; SQLite).

---

## Not a bug, but easy to trip over

- **Single-instance CLI apps take the raw argument list; multi-instance apps take `cli <command>`.**
  With one CLI instance, `myapp worker` runs `worker`; `myapp cli worker` reports `ROUTE_NOT_FOUND`.
  With `[SPA, CLI]`, `myapp cli worker` boots only the CLI instance. Both are intended.
- **The SPA platform listens on `localhost` by default.** Set `spa.WithHost("0.0.0.0")` in a container.

---

# Legger spikes (from `legger/index/TRD.md` §4.3, 2026-10-06)

Each is a spike first; fix in the framework, non-breaking, if the gap is real.

- **G1 `openapi`:** confirm the builder covers controller-method routes (`[]any{Controller{}, "Method"}`), `router.Mount` prefixes, error envelopes and header declarations (`Idempotency-Key`).
- **G2 `sql`:** a request-scoped transaction a handler, an outbox write and a ledger call can share.
- **G3 `router`:** a documented middleware-order hook for a group of mounted routes (auth → tenant → rate limit → idempotency → handler).

## Results (2026-10-06)

- **G1 `openapi` — works, one fix made.** Booted `legger/cloud` and built a document: 134 paths, 228 operations; request bodies, headers and bearer auth come from the Dto tags. Fixed: the same handler registered on the same method and path through two parent modules (by design per `ForRoutes`) was reported as "registered twice"; it is now one operation, while genuinely different handlers still error (tests added). Remaining limits, not bugs: responses are untyped (`{}`) unless a Dto implements `OpenAPI()`; generic `crud.Resource` routes carry no body schema (103 of 228 operations); the error envelope documented is Goose's, so `/v1` documents its own.
- **G2 `sql` — no change needed.** `Query.Tx` plus `Entity.With(query)` already let several entities share one transaction. The work is in Legger's ledger (`Layer1` must accept an outer `*gorm.DB` tx) and outbox.
- **G3 `router` — corrected: a change was needed (made).** Mount's ordered chain is fine, but a middleware could only continue or fail and could not see the output, so idempotent replay and 429s were impossible. Added `types.Respond` (a middleware answers the request) and `types.AfterMiddleware` (observe/replace the output); tests in `tests/middleware_respond_test.go`. Original finding: no change needed. `router.Mount(prefix, module, middlewares...)` attaches an ordered chain to the whole subtree (mount-level first, route-level after). Caveat: middlewares are attached at the prefix's innermost segment, so give `/v1` one `Mount` call.
- **Fixed earlier (found by Legger e2e):** the default config service parsed every YAML under the app root (now `<root>/config`; `NewConfig` tolerates a missing directory).
