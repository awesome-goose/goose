package queues

import "github.com/awesome-goose/goose/types"

// Root creates a root Queues module that initializes the queue tables and registers the Queue service.
// Use this in the main application module.
//
// Parameters:
//   - config: Module configuration
//   - handlers: Job handlers to register and start processing
//
// Example:
//
//	jobs.Root(&jobs.Config{
//	    DefaultRetryLimit: 3,
//	    DefaultRetryDelay: 5000,
//	}, []*jobs.JobHandler{
//	    jobs.NewHandler("emails", "send-welcome", handleSendWelcome),
//	    jobs.NewHandler("notifications", "push", handlePush).WithMaxWorkers(8),
//	})
func Root(config *Config, handlers ...*JobHandler) types.Module {
	return NewModule(config, handlers, true)
}

// Child creates a child Queues module that resolves the already-registered Queue service.
// Use this in sub-modules that need access to the Queue service.
//
// Parameters:
//   - handlers: Additional job handlers to register and start processing
//
// Example:
//
//	jobs.Child(
//	    jobs.NewHandler("reports", "generate", handleGenerateReport),
//	)
func Child(handlers ...*JobHandler) types.Module {
	return NewModule(nil, handlers, false)
}
