package tests

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/awesome-goose/goose/core"
	"github.com/awesome-goose/goose/log"
	"github.com/awesome-goose/goose/modules/queues"
	"github.com/awesome-goose/goose/modules/sql"
	test "github.com/awesome-goose/goose/testing"
	"github.com/awesome-goose/goose/types"
)

type queuesTestModule struct {
	handlers []*queues.JobHandler
	dbPath   string
}

func (m *queuesTestModule) Imports() []types.Module {
	return []types.Module{
		// File-backed, not ":memory:" — see cron_module_test.go for why.
		sql.Root(&sql.Config{Dialect: "sqlite", Name: m.dbPath}),
		queues.Child(m.handlers...),
	}
}
func (m *queuesTestModule) Exports() []any      { return nil }
func (m *queuesTestModule) Declarations() []any { return nil }

func TestQueuesModuleBoot(t *testing.T) {
	test.NewSuiteRunner(t, &QueuesModuleBootSuite{}).Run()
}

type QueuesModuleBootSuite struct {
	test.Suite
}

// Regression test for the modules/queues instance of the same bug as #2 in
// BUGS.md: queuesModule.Boot looked up *Queue via container.Resolve, but
// *Queue is only made constructible through Declarations() (tracked by the
// registry's own declarationIndex), never through container.Register. That
// mismatch meant Boot always failed with NO_CONCRETE_FOUND for any app
// registering queue handlers.
func (s *QueuesModuleBootSuite) TestBootResolvesQueueServiceViaRegistry() {
	handlers := []*queues.JobHandler{
		queues.NewHandler("test", "job", func(ctx context.Context, job *queues.QueueJob) (any, error) {
			return nil, nil
		}),
	}

	dbPath := filepath.Join(s.T.T().TempDir(), "queues-test.db")
	root := &queuesTestModule{handlers: handlers, dbPath: dbPath}

	traverser := core.NewTraverser()
	container := traverser.Container()

	// Mirror core's default `services`: *Queue injects types.Log, so it
	// must already be bound before Traverse hydrates the module tree.
	err := container.Register(func() types.Log {
		return log.NewLog(log.AppLogChannel(""))
	}, "", true)
	s.T.Expect(err).ToBeNil()

	err = traverser.Traverse(root)
	s.T.Expect(err).ToBeNil()

	k := &fakeKernel{traverser: traverser}

	// As with the cron module's boot hook, worker goroutines started here
	// may race sql.Child's own Boot hook creating the queue tables and log
	// a harmless warning — unrelated to this assertion, which only cares
	// that Boot() itself returns synchronously without NO_CONCRETE_FOUND.
	err = traverser.OnBootHooks().ExecuteAll(func(fn func(types.Kernel) error) error {
		return fn(k)
	})
	s.T.Expect(err).ToBeNil()
}
