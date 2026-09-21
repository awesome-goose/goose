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

## 6. Module `Boot()` hooks run in non-deterministic order — OPEN

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

---

## 7. `sql.Config.Sync` / `WithSync` is dead code — OPEN

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

## Not a bug, but easy to trip over

- **Single-instance CLI apps take the raw argument list; multi-instance apps take `cli <command>`.**
  With one CLI instance, `myapp worker` runs `worker`; `myapp cli worker` reports `ROUTE_NOT_FOUND`.
  With `[SPA, CLI]`, `myapp cli worker` boots only the CLI instance. Both are intended.
- **The SPA platform listens on `localhost` by default.** Set `spa.WithHost("0.0.0.0")` in a container.
