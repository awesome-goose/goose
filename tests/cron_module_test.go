package tests

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/awesome-goose/goose/core"
	"github.com/awesome-goose/goose/log"
	"github.com/awesome-goose/goose/modules/cron"
	"github.com/awesome-goose/goose/modules/sql"
	test "github.com/awesome-goose/goose/testing"
	"github.com/awesome-goose/goose/types"
)

// fakeKernel exposes only the Container/Registry a module's Boot hook needs,
// letting tests drive Traverse + boot hooks without a running platform/App.
type fakeKernel struct {
	traverser types.Traverser
}

func (k *fakeKernel) Start(instances ...*types.Instance) (func() error, error) {
	return func() error { return nil }, nil
}
func (k *fakeKernel) Routes() []types.Route { return nil }
func (k *fakeKernel) AppendRoutes(routes ...types.Route) ([]types.Route, error) {
	return nil, nil
}
func (k *fakeKernel) Container() types.Container { return k.traverser.Container() }
func (k *fakeKernel) Registry() types.Registry   { return k.traverser.Registry() }

type cronTestModule struct {
	handlers []*cron.CronHandler
	dbPath   string
}

func (m *cronTestModule) Imports() []types.Module {
	return []types.Module{
		// A file-backed DB, not ":memory:": GORM/database-sql pools multiple
		// connections, and a bare ":memory:" DSN gives each connection its
		// own isolated database, which flakes AutoMigrate/table lookups
		// independently of the fix under test here.
		sql.Root(&sql.Config{Dialect: "sqlite", Name: m.dbPath}),
		cron.Child(m.handlers...),
	}
}
func (m *cronTestModule) Exports() []any      { return nil }
func (m *cronTestModule) Declarations() []any { return nil }

func TestCronModuleBoot(t *testing.T) {
	test.NewSuiteRunner(t, &CronModuleBootSuite{}).Run()
}

type CronModuleBootSuite struct {
	test.Suite
}

// Regression test for bug #2 in BUGS.md: cronModule.Boot used to look up
// *Cron via container.Resolve, but *Cron is only made constructible through
// Declarations() (tracked by the registry's own declarationIndex), never
// through container.Register. That mismatch meant Boot always failed with
// NO_CONCRETE_FOUND for any app registering cron handlers.
func (s *CronModuleBootSuite) TestBootResolvesCronServiceViaRegistry() {
	handlers := []*cron.CronHandler{
		cron.NewSimpleHandler("test", "job", "* * * * *", func(job *cron.CronJob) (any, error) {
			return nil, nil
		}),
	}

	dbPath := filepath.Join(s.T.T().TempDir(), "cron-test.db")
	root := &cronTestModule{handlers: handlers, dbPath: dbPath}

	traverser := core.NewTraverser()
	container := traverser.Container()

	// Mirror core's default `services`: *Cron injects types.Log, so it must
	// already be bound before Traverse hydrates the module tree.
	err := container.Register(func() types.Log {
		return log.NewLog(log.AppLogChannel(""))
	}, "", true)
	s.T.Expect(err).ToBeNil()

	err = traverser.Traverse(root)
	s.T.Expect(err).ToBeNil()

	k := &fakeKernel{traverser: traverser}

	// Boot hook order across modules is unordered (registry.ListModules
	// ranges a map), so the cron runner's background goroutine can start
	// inserting rows before sql.Child's own Boot hook has finished creating
	// the CronJobs table on this run, logging a harmless "no such table"
	// warning. That's a pre-existing, separate boot-ordering quirk (not
	// bug #2) and doesn't affect this assertion: it's Boot() itself
	// returning synchronously, before that goroutine runs, that proves
	// *Cron resolved via the registry instead of failing with
	// NO_CONCRETE_FOUND.
	err = traverser.OnBootHooks().ExecuteAll(func(fn func(types.Kernel) error) error {
		return fn(k)
	})
	s.T.Expect(err).ToBeNil()
}

func TestCronModuleShutdown(t *testing.T) {
	test.NewSuiteRunner(t, &CronModuleShutdownSuite{}).Run()
}

type CronModuleShutdownSuite struct {
	test.Suite
}

// The kernel runs every Shutdownable module's hook on shutdown, but the cron
// module implemented none, so its runner kept ticking after the kernel had shut
// down.
func (s *CronModuleShutdownSuite) TestKernelShutdownStopsTheRunner() {
	handlers := []*cron.CronHandler{
		cron.NewSimpleHandler("test", "job", "* * * * *", func(job *cron.CronJob) (any, error) {
			return nil, nil
		}),
	}
	root := &cronTestModule{handlers: handlers, dbPath: filepath.Join(s.T.T().TempDir(), "cron-shutdown.db")}

	traverser := core.NewTraverser()
	s.T.Require(traverser.Container().Register(func() types.Log {
		return log.NewLog(log.AppLogChannel(""))
	}, "", true)).ToBeNil()
	s.T.Require(traverser.Traverse(root)).ToBeNil()

	k := &fakeKernel{traverser: traverser}
	s.T.Require(traverser.OnBootHooks().ExecuteAll(func(fn func(types.Kernel) error) error {
		return fn(k)
	})).ToBeNil()

	info, err := k.Registry().Get(&cron.Cron{})
	s.T.Require(err).ToBeNil()
	runner := info.Instance.(*cron.Cron)

	// The runner starts on its own goroutine; wait until it is up.
	deadline := time.Now().Add(5 * time.Second)
	for !runner.IsRunning() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	s.T.Require(runner.IsRunning()).ToEqual(true)

	s.T.Require(traverser.OnShutdownHooks().ExecuteAll(func(fn func(types.Kernel) error) error {
		return fn(k)
	})).ToBeNil()

	s.T.Expect(runner.IsRunning()).ToEqual(false)
}
