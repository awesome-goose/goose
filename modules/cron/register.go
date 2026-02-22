package cron

import "github.com/awesome-goose/goose/types"

// Root creates a root Cron module that initializes the cron tables and registers the Cron service.
// Use this in the main application module.
//
// Parameters:
//   - config: Module configuration (pass nil for defaults)
//   - handlers: Cron job handlers to register and start processing
//
// Example:
//
//	cron.Root(&cron.Config{
//	    TickInterval: 60 * time.Second,
//	    Timezone: "America/New_York",
//	},
//	    cron.NewHandler("reports", "daily-summary", "0 9 * * *", handleDailySummary),
//	    cron.NewHandler("cleanup", "temp-files", "0 0 * * *", handleCleanup).WithRetryLimit(5),
//	)
func Root(config *Config, handlers ...*CronHandler) types.Module {
	return NewModule(config, handlers, true)
}

// Child creates a child Cron module that resolves the already-registered Cron service.
// Use this in sub-modules that need to register additional cron handlers.
//
// Parameters:
//   - handlers: Additional cron job handlers to register
//
// Example:
//
//	cron.Child(
//	    cron.NewHandler("billing", "invoice-reminder", "0 8 * * 1", handleInvoiceReminder),
//	)
func Child(handlers ...*CronHandler) types.Module {
	return NewModule(nil, handlers, false)
}
