package errors

// ============================================================================
// CONFIG ERRORS
// ============================================================================

var (
	ErrConfigFileNotFound = New(
		"CONFIG_FILE_NOT_FOUND",
		"Config file for the specified namespace not found",
		"The configuration file was not found in the expected location",
		"Ensure the config file exists at the specified path or create it with default values",
	)
	ErrFailedToReadConfigFile = New(
		"FAILED_TO_READ_CONFIG_FILE",
		"Failed to read config file",
		"Unable to read the contents of the configuration file",
		"Check file permissions and ensure the file is not corrupted",
	)
	ErrFailedToUnmarshalConfigToStruct = New(
		"FAILED_TO_UNMARSHAL_CONFIG_TO_STRUCT",
		"Failed to unmarshal config to struct",
		"The configuration data could not be parsed into the expected structure",
		"Verify the config file format matches the expected schema (YAML/JSON)",
	)
	ErrNamespaceRequired = New(
		"NAMESPACE_REQUIRED",
		"Namespace is required",
		"A configuration namespace must be provided",
		"Specify a namespace when accessing configuration values",
	)
	ErrFailedToMarshalStruct = New(
		"FAILED_TO_MARSHAL_STRUCT",
		"Failed to marshal struct",
		"Unable to convert the struct to a serialized format",
		"Ensure all struct fields are serializable",
	)
	ErrFailedToConvertStructToMap = New(
		"FAILED_TO_CONVERT_STRUCT_TO_MAP",
		"Failed to convert struct to map",
		"The struct could not be converted to a map representation",
		"Check that the struct fields have proper tags for conversion",
	)
	ErrPathNotFound = New(
		"PATH_NOT_FOUND",
		"The specified path was not found in the configuration",
		"The dotted path does not exist in the configuration tree",
		"Verify the path exists in your configuration or use a default value",
	)
	ErrKeyNotFound = New(
		"KEY_NOT_FOUND",
		"The specified key was not found in the configuration at the given path",
		"The key does not exist at the specified configuration path",
		"Check the key name or provide a default value",
	)
	ErrInvalidSet = New(
		"INVALID_SET",
		"Cannot set key on non-map value at the specified path",
		"Attempted to set a key on a value that is not a map",
		"Ensure the parent path is a map before setting nested values",
	)
	ErrFailedToResolveConfigPath = New(
		"FAILED_TO_RESOLVE_CONFIG_PATH",
		"Failed to resolve config path",
		"The configuration file path could not be resolved",
		"Check that the config file exists and the path is accessible",
	)
)

// ============================================================================
// CONTAINER ERRORS
// ============================================================================

var (
	ErrInvalidResolver = New(
		"INVALID_RESOLVER",
		"The resolver must be a function",
		"Container bindings require a function that returns the concrete implementation",
		"Provide a function that returns the desired type when binding to the container",
	)
	ErrNoConcreteFound = New(
		"NO_CONCRETE_FOUND",
		"No concrete found for the given abstraction",
		"No implementation has been registered for this type",
		"Register a concrete implementation for this type before resolving",
	)
	ErrInvalidAbstraction = New(
		"INVALID_ABSTRACTION",
		"The abstraction must be a pointer to an interface or struct",
		"Container can only resolve pointers to interfaces or structs",
		"Pass a pointer to an interface or struct when resolving dependencies",
	)
	ErrInvalidConstructor = New(
		"INVALID_CONSTRUCTOR",
		"No valid constructor found for the given type",
		"The type does not have a valid constructor or resolver",
		"Register a resolver function or ensure the type can be constructed",
	)
	ErrInvalidResolverSignature = New(
		"INVALID_RESOLVER_SIGNATURE",
		"The resolver function signature is invalid",
		"Resolver must return 1 value or 2 values (value, error)",
		"Ensure your resolver function has a valid return signature",
	)
	ErrCannotResolve = New(
		"CANNOT_RESOLVE",
		"Cannot resolve the given type from the container",
		"The requested type could not be resolved from the container",
		"Check if the type or its dependencies are properly registered",
	)
	ErrConstructorDidNotReturnAnything = New(
		"CONSTRUCTOR_DID_NOT_RETURN_ANYTHING",
		"The constructor did not return anything",
		"The constructor function returned no value",
		"Ensure your constructor returns the created instance",
	)
	ErrInvalidMethod = New(
		"INVALID_METHOD",
		"The specified method does not exist on the given type",
		"The method name provided does not exist on the target type",
		"Verify the method name and ensure it exists on the type",
	)
	ErrBindingNoResolverOrInstance = New(
		"BINDING_NO_RESOLVER_OR_INSTANCE",
		"Binding has no resolver and no instance",
		"A container binding must have either a resolver function or a cached instance",
		"Register the type with a resolver function before resolving",
	)
	ErrCannotInvokeNilFunction = New(
		"CANNOT_INVOKE_NIL_FUNCTION",
		"Cannot invoke nil function",
		"A nil function was passed to the container",
		"Ensure the function is not nil before invoking",
	)
	ErrResolverMustBeFunction = New(
		"RESOLVER_MUST_BE_FUNCTION",
		"Resolver must be a function",
		"The provided resolver is not a function type",
		"Provide a function that returns the concrete type",
	)
	ErrCannotGetArgumentsForNil = New(
		"CANNOT_GET_ARGUMENTS_FOR_NIL",
		"Cannot get arguments for nil function",
		"Attempted to resolve arguments for a nil function",
		"Provide a valid function reference",
	)
	ErrCircularDependency = New(
		"CIRCULAR_DEPENDENCY",
		"Circular dependency detected",
		"A circular dependency was found in the dependency graph",
		"Refactor your dependencies to remove circular references",
	)
	ErrFailedToResolveField = New(
		"FAILED_TO_RESOLVE_FIELD",
		"Failed to resolve field",
		"Could not resolve a struct field dependency",
		"Ensure all field dependencies are properly registered",
	)
	ErrCannotCreateInterfaceField = New(
		"CANNOT_CREATE_INTERFACE_FIELD",
		"Cannot create interface field with no binding",
		"An interface field cannot be created without a registered binding",
		"Register an implementation for the interface before creating",
	)
	ErrCannotResolveField = New(
		"CANNOT_RESOLVE_FIELD",
		"Cannot resolve field",
		"The container could not resolve a struct field",
		"Register the field type with the container",
	)
	ErrResolverCannotBeNil = New(
		"RESOLVER_CANNOT_BE_NIL",
		"Resolver cannot be nil",
		"A nil resolver was provided to the container",
		"Provide a valid non-nil resolver function",
	)
	ErrInvalidFunction = New(
		"INVALID_FUNCTION",
		"Invalid function",
		"The provided value is not a valid function",
		"Ensure you are passing a function to Call()",
	)
	ErrInvalidStructure = New(
		"INVALID_STRUCTURE",
		"Invalid structure",
		"The provided value is not a valid struct pointer",
		"Pass a pointer to a struct when calling Fill()",
	)
	ErrCreateValueCannotBeNil = New(
		"CREATE_VALUE_CANNOT_BE_NIL",
		"Create's value cannot be nil",
		"Attempted to create a nil value",
		"Provide a non-nil value to Create()",
	)
	ErrSecondReturnNotError = New(
		"SECOND_RETURN_NOT_ERROR",
		"Second return value is not an error",
		"Resolver's second return value must be an error type",
		"Ensure your resolver returns (value, error) or just (value)",
	)
	ErrFailedToResolveArgument = New(
		"FAILED_TO_RESOLVE_ARGUMENT",
		"Failed to resolve function argument",
		"Could not resolve a function argument dependency",
		"Ensure all argument dependencies are registered",
	)
)

// ============================================================================
// REGISTRY ERRORS
// ============================================================================

var (
	ErrCircularImport = New(
		"CIRCULAR_IMPORT",
		"Circular module import detected",
		"Module A imports B which imports A (directly or indirectly)",
		"Refactor your modules to remove circular import dependencies",
	)
	ErrInvalidExport = New(
		"INVALID_EXPORT",
		"Cannot export declaration not owned by module",
		"A module tried to export a declaration it doesn't own",
		"Only export declarations that are defined in the module's Declarations()",
	)
	ErrDuplicateDeclaration = New(
		"DUPLICATE_DECLARATION",
		"Declaration type already registered",
		"The same type was declared in multiple modules",
		"Remove duplicate declarations or use different types",
	)
	ErrModuleNotFound = New(
		"MODULE_NOT_FOUND",
		"Module not found in registry",
		"The requested module is not registered",
		"Ensure the module is imported before accessing it",
	)
	ErrDeclarationNotInScope = New(
		"DECLARATION_NOT_IN_SCOPE",
		"Declaration not available in module scope",
		"The field type is not available in the module's scope",
		"Import or export the required declaration between modules",
	)
	ErrDeclarationNotFound = New(
		"DECLARATION_NOT_FOUND",
		"Declaration not found in any module",
		"The requested declaration type was not found",
		"Register the declaration in a module before accessing",
	)
	ErrInstanceNotModule = New(
		"INSTANCE_NOT_MODULE",
		"Instance created from module is not a module",
		"The resolved instance does not implement the Module interface",
		"Ensure your module type implements types.Module",
	)
	ErrModuleConfigureFailed = New(
		"MODULE_CONFIGURE_FAILED",
		"Module Configure failed",
		"The module's Configure hook returned an error",
		"Check the Configure method implementation for errors",
	)
)

// ============================================================================
// KERNEL ERRORS
// ============================================================================

var (
	ErrDuplicateRoute = New(
		"DUPLICATE_ROUTE",
		"Duplicate route detected",
		"A route with the same method and path already exists",
		"Use unique method+path combinations for each route",
	)
	ErrInvalidRoute = New(
		"INVALID_ROUTE",
		"Invalid route detected",
		"The route configuration is invalid",
		"Verify the route has a valid method, path, and handler",
	)
	ErrNoInstancesProvided = New(
		"NO_INSTANCES_PROVIDED",
		"No instances provided",
		"Start() was called with no platform instances",
		"Provide at least one platform instance to Start()",
	)
	ErrMultipleCLIInstances = New(
		"MULTIPLE_CLI_INSTANCES",
		"Only one CLI instance is allowed",
		"More than one CLI platform instance was provided",
		"Use only one CLI instance when running in multi-platform mode",
	)
	ErrCLIModeNoCLIInstance = New(
		"CLI_MODE_NO_CLI_INSTANCE",
		"CLI mode requested but no CLI instance defined",
		"The 'cli' argument was passed but no CLI instance is configured",
		"Add a CLI platform instance to your configuration",
	)
	ErrNoRunnableInstances = New(
		"NO_RUNNABLE_INSTANCES",
		"No runnable instances available",
		"No platform instances are available to run",
		"Provide at least one API, Web, SPA, or CLI instance",
	)
	ErrInitializationError = New(
		"INITIALIZATION_ERROR",
		"Initialization error",
		"An error occurred during platform initialization",
		"Check your initializer functions for errors",
	)
	ErrModuleTraversalError = New(
		"MODULE_TRAVERSAL_ERROR",
		"Module traversal error",
		"An error occurred while traversing modules",
		"Check module imports and declarations for errors",
	)
	ErrBootHookError = New(
		"BOOT_HOOK_ERROR",
		"Boot hook error",
		"An error occurred in a boot hook",
		"Review your boot hooks for errors",
	)
	ErrPlatformBootError = New(
		"PLATFORM_BOOT_ERROR",
		"Platform boot error",
		"The platform failed to boot",
		"Check platform configuration and dependencies",
	)
	ErrRuntimeError = New(
		"RUNTIME_ERROR",
		"Runtime error",
		"An error occurred during runtime",
		"Check application logs for details",
	)
	ErrInvalidHandlerFormat = New(
		"INVALID_HANDLER_FORMAT",
		"Invalid handler format",
		"The route handler format is incorrect",
		"Use [controller, methodName] format for controller handlers",
	)
	ErrMethodNameNotString = New(
		"METHOD_NAME_NOT_STRING",
		"Method name must be a string",
		"The handler method name is not a string",
		"Provide the method name as a string in [controller, 'MethodName']",
	)
	ErrFailedToMakeController = New(
		"FAILED_TO_MAKE_CONTROLLER",
		"Failed to make controller",
		"Could not create the controller instance",
		"Ensure the controller type is properly defined",
	)
	ErrMethodNotFound = New(
		"METHOD_NOT_FOUND",
		"Method not found on controller",
		"The specified method does not exist on the controller",
		"Verify the method name matches a method on the controller",
	)
	ErrMethodWrongArgCount = New(
		"METHOD_WRONG_ARG_COUNT",
		"Method must have exactly 1 argument",
		"Controller methods must accept exactly one argument",
		"Define the method with a single struct pointer argument",
	)
	ErrMethodArgNotPtrStruct = New(
		"METHOD_ARG_NOT_PTR_STRUCT",
		"Method argument must be a pointer to a struct",
		"Controller method argument must be *SomeStruct",
		"Change the method signature to accept a pointer to a struct",
	)
	ErrFailedToHydrateInput = New(
		"FAILED_TO_HYDRATE_INPUT",
		"Failed to hydrate input struct",
		"Could not populate the input struct from the request",
		"Check that input tags match the request data format",
	)
	ErrUnsupportedHandlerType = New(
		"UNSUPPORTED_HANDLER_TYPE",
		"Unsupported handler type",
		"The handler type is not supported",
		"Use a function or [controller, method] tuple as handler",
	)
)

// ============================================================================
// ROUTER ERRORS
// ============================================================================

var (
	ErrNoRouteMatch = New(
		"NO_ROUTE_MATCH",
		"No route match found for the given paths",
		"No registered route matches the request method and path",
		"Verify the route is registered and the URL is correct",
	)
	ErrRouteNotFound = New(
		"ROUTE_NOT_FOUND",
		"Route not found",
		"The requested route was not found",
		"Check the route registration and request URL",
	)
	ErrRouteDepthExceeded = New(
		"ROUTE_DEPTH_EXCEEDED",
		"Route nesting depth exceeded",
		"The route tree was traversed beyond the maximum supported depth",
		"Flatten deeply nested route trees or raise maxRouteDepth if intentional",
	)
	ErrInvalidRouterInstance = New(
		"INVALID_ROUTER_INSTANCE",
		"Invalid router instance",
		"The resolved router is not a valid Router type",
		"Ensure your router implements types.Router",
	)
	ErrHandlerMustBeTuple = New(
		"HANDLER_MUST_BE_TUPLE",
		"Handler must be a tuple of [handler, methodName]",
		"Resource handlers must use the [controller, method] format",
		"Use Route().Handler([]any{&Controller{}, 'Method'})",
	)
	ErrHandlerMethodNotString = New(
		"HANDLER_METHOD_NOT_STRING",
		"Handler method must be a string",
		"The method name in the handler tuple is not a string",
		"Provide method name as string: [controller, 'MethodName']",
	)
)

// ============================================================================
// SERIALIZER ERRORS
// ============================================================================

var (
	ErrJSONMarshalFailed = New(
		"JSON_MARSHAL_FAILED",
		"JSON marshal failed",
		"Failed to serialize data to JSON format",
		"Ensure all data types are JSON-serializable",
	)
	ErrUnsupportedType = New(
		"UNSUPPORTED_TYPE",
		"Unsupported type",
		"The serializer does not support this type",
		"Use a supported type or implement custom serialization",
	)
)

// ============================================================================
// TRAVERSER ERRORS
// ============================================================================

var (
	ErrFailedToMakeDeclaration = New(
		"FAILED_TO_MAKE_DECLARATION",
		"Failed to make declaration",
		"Could not create the declaration instance",
		"Check the declaration type and its dependencies",
	)
	ErrFailedToGetRoutesFromRouter = New(
		"FAILED_TO_GET_ROUTES_FROM_ROUTER",
		"Failed to get routes from router",
		"The router's Routes() method returned an error",
		"Check your router implementation for errors",
	)
	ErrInvalidModuleInstance = New(
		"INVALID_MODULE_INSTANCE",
		"Invalid module instance",
		"The module instance is not valid",
		"Ensure the module implements types.Module correctly",
	)
)

// ============================================================================
// INPUT ERRORS
// ============================================================================

var (
	ErrPayloadMustBeNonNilPointer = New(
		"PAYLOAD_MUST_BE_NON_NIL_POINTER",
		"Payload must be a non-nil pointer to a struct",
		"The input payload must be a pointer to a struct, not nil",
		"Pass a valid pointer to a struct: &MyStruct{}",
	)
	ErrPayloadMustBePointerToStruct = New(
		"PAYLOAD_MUST_BE_POINTER_TO_STRUCT",
		"Payload must be a pointer to a struct",
		"The payload kind is not a struct",
		"Pass a pointer to a struct type",
	)
	ErrFailedToSetHeaderField = New(
		"FAILED_TO_SET_HEADER_FIELD",
		"Failed to set header field",
		"Could not set the struct field from header value",
		"Check field type compatibility with header value",
	)
	ErrFailedToSetContextField = New(
		"FAILED_TO_SET_CONTEXT_FIELD",
		"Failed to set context field",
		"Could not set the struct field from context value",
		"Check field type compatibility with context value",
	)
	ErrFailedToSetParamField = New(
		"FAILED_TO_SET_PARAM_FIELD",
		"Failed to set param field",
		"Could not set the struct field from URL parameter",
		"Check field type compatibility with param value",
	)
	ErrFailedToSetQueryField = New(
		"FAILED_TO_SET_QUERY_FIELD",
		"Failed to set query field",
		"Could not set the struct field from query parameter",
		"Check field type compatibility with query value",
	)
	ErrFailedToSetFlagField = New(
		"FAILED_TO_SET_FLAG_FIELD",
		"Failed to set flag field",
		"Could not set the struct field from flag value",
		"Check field type compatibility with flag value",
	)
	ErrFailedToSetFormField = New(
		"FAILED_TO_SET_FORM_FIELD",
		"Failed to set form field",
		"Could not set the struct field from form data",
		"Check field type compatibility with form value",
	)
	ErrFailedToSetJSONField = New(
		"FAILED_TO_SET_JSON_FIELD",
		"Failed to set JSON field",
		"Could not set the struct field from JSON data",
		"Check field type compatibility with JSON value",
	)
	ErrUnsupportedSliceType = New(
		"UNSUPPORTED_SLICE_TYPE",
		"Unsupported slice element type",
		"The slice element type is not supported for conversion",
		"Use string slices or implement custom conversion",
	)
	ErrUnsupportedFieldType = New(
		"UNSUPPORTED_FIELD_TYPE",
		"Unsupported field type",
		"The struct field type is not supported",
		"Use basic types (string, int, bool, float) or implement custom handling",
	)
	ErrCannotConvertToInt = New(
		"CANNOT_CONVERT_TO_INT",
		"Cannot convert value to int",
		"The value cannot be converted to an integer type",
		"Ensure the value is a valid integer or numeric string",
	)
	ErrCannotConvertToUint = New(
		"CANNOT_CONVERT_TO_UINT",
		"Cannot convert value to uint",
		"The value cannot be converted to an unsigned integer",
		"Ensure the value is a valid positive integer",
	)
	ErrCannotConvertToFloat = New(
		"CANNOT_CONVERT_TO_FLOAT",
		"Cannot convert value to float",
		"The value cannot be converted to a float type",
		"Ensure the value is a valid decimal number",
	)
	ErrCannotConvertToBool = New(
		"CANNOT_CONVERT_TO_BOOL",
		"Cannot convert value to bool",
		"The value cannot be converted to a boolean",
		"Use true/false, 1/0, or yes/no values",
	)
	ErrUnsupportedJSONFieldType = New(
		"UNSUPPORTED_JSON_FIELD_TYPE",
		"Unsupported field type for JSON",
		"The field type cannot be populated from JSON",
		"Use basic types, slices, maps, or structs",
	)
)

// ============================================================================
// OUTPUT/TEMPLATE ERRORS
// ============================================================================

var (
	ErrTemplateNotFound = New(
		"TEMPLATE_NOT_FOUND",
		"Template not found",
		"The template file could not be found in any configured directory",
		"Verify the template path and ensure files exist in views directory",
	)
	ErrReadingPartial = New(
		"ERROR_READING_PARTIAL",
		"Error reading partial",
		"Failed to read a partial template file",
		"Check file permissions and path for the partial",
	)
	ErrParsingPartial = New(
		"ERROR_PARSING_PARTIAL",
		"Error parsing partial",
		"Failed to parse a partial template file",
		"Check the partial template syntax for errors",
	)
)

// ============================================================================
// SQL/DATABASE ERRORS
// ============================================================================

var (
	ErrRecordNotFound = New(
		"RECORD_NOT_FOUND",
		"Record not found",
		"The requested database record does not exist",
		"Verify the query conditions or handle not-found case",
	)
	ErrInvalidColumnName = New(
		"INVALID_COLUMN_NAME",
		"Invalid column name",
		"The column name contains invalid characters",
		"Use alphanumeric characters and underscores only",
	)
	ErrFailedToCheckMigrationStatus = New(
		"FAILED_TO_CHECK_MIGRATION_STATUS",
		"Failed to check migration status",
		"Could not verify if migration has been executed",
		"Check database connection and migration table",
	)
	ErrUnknownMigrationType = New(
		"UNKNOWN_MIGRATION_TYPE",
		"Unknown migration type",
		"The runnable is neither a Migration nor a Seeder",
		"Implement either Migration or Seeder interface",
	)
	ErrFailedToResolveLog = New(
		"FAILED_TO_RESOLVE_LOG",
		"Failed to resolve log",
		"Could not resolve the logger from container",
		"Ensure a logger is registered in the container",
	)
	ErrFailedToResolveDatabase = New(
		"FAILED_TO_RESOLVE_DATABASE",
		"Failed to resolve database",
		"Could not resolve the database from container",
		"Ensure the SQL module is configured as root in app module",
	)
	ErrDeleteRequiresWhereClause = New(
		"DELETE_REQUIRES_WHERE_CLAUSE",
		"Delete requires a where clause",
		"Cannot delete without a where condition to prevent accidental data loss",
		"Use DeleteAll for removing all records or provide a where clause",
	)
	ErrNoDatabaseConfigured = New(
		"NO_DATABASE_CONFIGURED",
		"No database configured",
		"The database connection is nil",
		"Configure the SQL module with valid database settings",
	)
	ErrFailedToRunMigration = New(
		"FAILED_TO_RUN_MIGRATION",
		"Failed to run migration",
		"A database migration failed to execute",
		"Check migration code and database schema",
	)
	ErrFailedToRunSeeder = New(
		"FAILED_TO_RUN_SEEDER",
		"Failed to run seeder",
		"A database seeder failed to execute",
		"Check seeder code and data constraints",
	)
)

// ============================================================================
// CRON ERRORS
// ============================================================================

var (
	ErrCronJobNotFound = New(
		"CRON_JOB_NOT_FOUND",
		"Cron job not found",
		"The requested cron job does not exist",
		"Verify the job ID and ensure it was registered",
	)
	ErrCronJobExpired = New(
		"CRON_JOB_EXPIRED",
		"Cron job expired",
		"The cron job has passed its expiration time",
		"Reschedule the job or remove expiration constraint",
	)
	ErrCronJobLocked = New(
		"CRON_JOB_LOCKED",
		"Cron job locked by another worker",
		"Another worker is currently executing this job",
		"Wait for the lock to be released or increase lock timeout",
	)
	ErrCronShuttingDown = New(
		"CRON_SHUTTING_DOWN",
		"Cron service is shutting down",
		"The cron service is in shutdown mode",
		"Wait for service restart or allow graceful shutdown",
	)
	ErrCronJobTimeout = New(
		"CRON_JOB_TIMEOUT",
		"Cron job execution timed out",
		"The job took longer than the configured timeout",
		"Optimize the job or increase the timeout setting",
	)
	ErrCronRetryExhausted = New(
		"CRON_RETRY_EXHAUSTED",
		"Retry limit exceeded",
		"The job has exhausted all retry attempts",
		"Fix the underlying issue or increase retry limit",
	)
	ErrCronInvalidHandler = New(
		"CRON_INVALID_HANDLER",
		"Invalid cron handler",
		"The job handler is not a valid function",
		"Provide a function with signature func(context.Context) error",
	)
	ErrCronInvalidPattern = New(
		"CRON_INVALID_PATTERN",
		"Invalid cron pattern",
		"The cron schedule pattern is invalid",
		"Use a valid cron expression (e.g., '*/5 * * * *')",
	)
	ErrInvalidLogStatus = New(
		"INVALID_LOG_STATUS",
		"Invalid log status",
		"The log status must be 'success' or 'failed'",
		"Use LogStatusSuccess or LogStatusFailed constants",
	)
)

// ============================================================================
// QUEUE ERRORS
// ============================================================================

var (
	ErrQueueNotFound = New(
		"QUEUE_NOT_FOUND",
		"Queue not found",
		"The requested queue does not exist",
		"Create the queue before pushing jobs",
	)
	ErrQueueJobNotFound = New(
		"QUEUE_JOB_NOT_FOUND",
		"Job not found",
		"The requested job does not exist in the queue",
		"Verify the job ID and queue name",
	)
	ErrQueueJobExpired = New(
		"QUEUE_JOB_EXPIRED",
		"Job expired",
		"The job has passed its expiration time",
		"Requeue the job or remove expiration constraint",
	)
	ErrQueueJobLocked = New(
		"QUEUE_JOB_LOCKED",
		"Job locked by another worker",
		"Another worker is processing this job",
		"Wait for the worker to complete or investigate stale locks",
	)
	ErrQueueShuttingDown = New(
		"QUEUE_SHUTTING_DOWN",
		"Queue service is shutting down",
		"The queue service is in shutdown mode",
		"Wait for service restart or allow graceful shutdown",
	)
	ErrQueueJobTimeout = New(
		"QUEUE_JOB_TIMEOUT",
		"Job execution timed out",
		"The job took longer than the configured timeout",
		"Optimize the job or increase the timeout setting",
	)
	ErrQueueRetryExhausted = New(
		"QUEUE_RETRY_EXHAUSTED",
		"Retry limit exceeded",
		"The job has exhausted all retry attempts",
		"Fix the underlying issue or increase retry limit",
	)
	ErrQueueInvalidHandler = New(
		"QUEUE_INVALID_HANDLER",
		"Invalid job handler",
		"The job handler is not valid",
		"Provide a valid handler function",
	)
	ErrQueueInvalidStatus = New(
		"QUEUE_INVALID_STATUS",
		"Invalid job status",
		"The provided job status is not valid",
		"Use 'success' or 'failed' as the status value",
	)
	ErrQueueJobCancelNotAllowed = New(
		"QUEUE_JOB_CANCEL_NOT_ALLOWED",
		"Job cannot be cancelled",
		"Only jobs with 'new' status can be cancelled",
		"Wait for the job to complete or check its current status",
	)
	ErrJobHandlerPanic = New(
		"JOB_HANDLER_PANIC",
		"Job handler panicked",
		"The job handler caused a panic during execution",
		"Add proper error handling in your job handler",
	)
)

// ============================================================================
// LOG ERRORS
// ============================================================================

var (
	ErrSyslogNotSupported = New(
		"SYSLOG_NOT_SUPPORTED",
		"Syslog is not supported on Windows",
		"The syslog processor is not available on Windows",
		"Use file or console processor instead on Windows",
	)
	ErrSyslogPermissionDenied = New(
		"SYSLOG_PERMISSION_DENIED",
		"Permission denied: cannot write to syslog",
		"The application lacks permission to write to syslog",
		"Run with appropriate permissions or use alternative log processor",
	)
	ErrSyslogWriteFailed = New(
		"SYSLOG_WRITE_FAILED",
		"Unable to write to syslog",
		"Failed to write log entry to syslog",
		"Check syslog service status and configuration",
	)
	ErrNoLoggersFound = New(
		"NO_LOGGERS_FOUND",
		"No loggers found for channel",
		"No log processors are configured for this channel",
		"Configure at least one processor for the log channel",
	)
)
