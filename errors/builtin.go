package errors

var (
	ErrConfigFileNotFound = New(
		"CONFIG_FILE_NOT_FOUND",
		"Config file for the specified namespace not found",
	)
	ErrFailedToReadConfigFile = New(
		"FAILED_TO_READ_CONFIG_FILE",
		"Failed to read config file",
	)
	ErrFailedToUnmarshalConfigToStruct = New(
		"FAILED_TO_UNMARSHAL_CONFIG_TO_STRUCT",
		"Failed to unmarshal config to struct",
	)
	ErrNamespaceRequired = New(
		"NAMESPACE_REQUIRED",
		"Namespace is required",
	)
	ErrFailedToMarshalStruct = New(
		"FAILED_TO_MARSHAL_STRUCT",
		"Failed to marshal struct",
	)
	ErrFailedToConvertStructToMap = New(
		"FAILED_TO_CONVERT_STRUCT_TO_MAP",
		"Failed to convert struct to map",
	)
)

var (
	ErrPathNotFound = New("PATH_NOT_FOUND", "The specified path was not found in the configuration")
	ErrKeyNotFound  = New("KEY_NOT_FOUND", "The specified key was not found in the configuration at the given path")
	ErrInvalidSet   = New("INVALID_SET", "Cannot set key on non-map value at the specified path")
)

var (
	ErrInvalidResolver = New(
		"INVALID_RESOLVER",
		"the resolver must be a function",
	)
	ErrNoConcreteFound = New(
		"NO_CONCRETE_FOUND",
		"no concrete found for the given abstraction",
	)
	ErrInvalidAbstraction = New(
		"INVALID_ABSTRACTION",
		"the abstraction must be a pointer to an interface or struct",
	)
	ErrInvalidConstructor = New(
		"INVALID_CONSTRUCTOR",
		"no valid constructor found for the given type",
	)
	ErrInvalidResolverSignature = New(
		"INVALID_RESOLVER_SIGNATURE",
		"the resolver function signature is invalid",
	)
	ErrCannotResolve = New(
		"CANNOT_RESOLVE",
		"cannot resolve the given type from the container",
	)
	ErrConstructorDidNotReturnAnything = New(
		"CONSTRUCTOR_DID_NOT_RETURN_ANYTHING",
		"the constructor did not return anything",
	)
	ErrInvalidMethod = New(
		"INVALID_METHOD",
		"the specified method does not exist on the given type",
	)
)

var (
	ErrNoRouteMatch = New("ERR_NO_ROUTE_MATCH", "No route match found for the given paths")
)

var (
	JSONMARSHALFAILED = New(
		"ERR_SERIALIZER_JSON_MARSHAL_FAILED",
		"JSON marshal failed",
	)
	UNSUPPORTEDTYPE = New(
		"ERR_SERIALIZER_UNSUPPORTED_TYPE",
		"Unsupported type",
	)
)

var (
	ErrFailedToMakeDeclaration = New(
		"ERR_FAILED_TO_MAKE_DECLARATION",
		"Failed to make declaration",
	)
	ErrFailedToGetRoutesFromRouter = New(
		"ERR_FAILED_TO_GET_ROUTES_FROM_ROUTER",
		"Failed to get routes from router",
	)
	ErrInvalidModuleInstance = New(
		"ERR_INVALID_MODULE_INSTANCE",
		"Invalid module instance",
	)
)

var (
	ErrSyslogNotSupported = New(
		"SYSLOG_NOT_SUPPORTED",
		"syslog is not supported on Windows",
	)
	ErrSyslogPermissionDenied = New(
		"SYSLOG_PERMISSION_DENIED",
		"permission denied: cannot write to syslog",
	)
	ErrSyslogWriteFailed = New(
		"SYSLOG_WRITE_FAILED",
		"unable to write to syslog",
	)
)
