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
