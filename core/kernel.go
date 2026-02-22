package core

import (
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"syscall"

	"github.com/awesome-goose/goose/input"
	"github.com/awesome-goose/goose/types"
)

// ErrDuplicateRoute is returned when attempting to add a route that already exists.
var ErrDuplicateRoute = fmt.Errorf("kernel: duplicate route detected")
var ErrInvalidRoute = fmt.Errorf("kernel: invalid route detected")

type kernel struct {
	// Core components
	router     types.RouterFinder
	routes     types.Routes
	serializer types.Serializer
	traverser  types.Traverser

	// Multi-platform support
	runningApps  []types.App
	childKernels []*kernel
	mu           sync.Mutex
	shutdownOnce sync.Once
}

func NewKernel() *kernel {
	return &kernel{
		router:       NewRouter(),
		routes:       []types.Route{},
		serializer:   NewSerializer(),
		traverser:    NewTraverser(),
		runningApps:  make([]types.App, 0),
		childKernels: make([]*kernel, 0),
	}
}

// Start starts one or more platform instances
// - Single instance: runs directly
// - Multiple instances: API/Web run concurrently, CLI runs when `cli` arg is passed
func (k *kernel) Start(instances ...*types.Instance) (func() error, error) {
	if len(instances) == 0 {
		return func() error { return nil }, fmt.Errorf("no instances provided")
	}

	// Single instance mode
	if len(instances) == 1 {
		return k.runSingle(instances[0])
	}

	// Multi-instance mode
	return k.runMulti(instances)
}

// runSingle runs a single platform instance
func (k *kernel) runSingle(inst *types.Instance) (func() error, error) {
	stop := func() error {
		return k.shutdown()
	}

	container := k.traverser.Container()
	for _, fn := range services {
		container.Register(fn, "", true)
	}

	for _, initFn := range inst.Initializers {
		if err := initFn(container); err != nil {
			return stop, err
		}
	}

	if err := k.traverser.Traverse(inst.Module); err != nil {
		return stop, err
	}

	if err := k.traverser.OnBootHooks().ExecuteAll(func(fn func(types.Kernel) error) error {
		return fn(k)
	}); err != nil {
		return stop, err
	}

	app, err := inst.Platform.Boot(container)
	if err != nil {
		return stop, err
	}

	k.mu.Lock()
	k.runningApps = append(k.runningApps, app)
	k.mu.Unlock()

	err = app.Run(k.createHandler())
	return stop, err
}

// runMulti runs multiple platform instances
func (k *kernel) runMulti(instances []*types.Instance) (func() error, error) {
	// Validate: only one CLI instance allowed
	cliCount := 0
	var cliInstance *types.Instance
	var serverInstances []*types.Instance

	for _, inst := range instances {
		if inst.Type == types.PlatformTypeCLI {
			cliCount++
			cliInstance = inst
		} else {
			serverInstances = append(serverInstances, inst)
		}
	}

	if cliCount > 1 {
		return func() error { return nil }, fmt.Errorf("only one CLI instance is allowed, found %d", cliCount)
	}

	// Check if CLI mode is requested via command line args
	cliMode := len(os.Args) > 1 && os.Args[1] == "cli"

	// If CLI mode requested but no CLI instance defined
	if cliMode && cliInstance == nil {
		return func() error { return nil }, fmt.Errorf("CLI mode requested but no CLI instance defined")
	}

	// If CLI mode, only run the CLI instance
	if cliMode && cliInstance != nil {
		return k.runCLI(cliInstance)
	}

	// Otherwise, run server instances concurrently
	if len(serverInstances) == 0 {
		if cliInstance != nil {
			return k.runCLI(cliInstance)
		}
		return func() error { return nil }, fmt.Errorf("no runnable instances available")
	}

	return k.runServers(serverInstances)
}

// runCLI runs a CLI instance in the main goroutine (blocking)
func (k *kernel) runCLI(inst *types.Instance) (func() error, error) {
	childKernel := NewKernel()
	k.mu.Lock()
	k.childKernels = append(k.childKernels, childKernel)
	k.mu.Unlock()

	stop := func() error {
		return k.shutdown()
	}

	container := childKernel.traverser.Container()
	for _, fn := range services {
		container.Register(fn, "", true)
	}

	for _, initFn := range inst.Initializers {
		if err := initFn(container); err != nil {
			return stop, err
		}
	}

	if err := childKernel.traverser.Traverse(inst.Module); err != nil {
		return stop, err
	}

	if err := childKernel.traverser.OnBootHooks().ExecuteAll(func(fn func(types.Kernel) error) error {
		return fn(childKernel)
	}); err != nil {
		return stop, err
	}

	app, err := inst.Platform.Boot(container)
	if err != nil {
		return stop, err
	}

	k.mu.Lock()
	k.runningApps = append(k.runningApps, app)
	k.mu.Unlock()

	err = app.Run(childKernel.createHandler())
	return stop, err
}

// runServers runs multiple server instances concurrently
func (k *kernel) runServers(instances []*types.Instance) (func() error, error) {
	stop := func() error {
		return k.shutdown()
	}

	errChan := make(chan error, len(instances))
	var wg sync.WaitGroup

	for _, inst := range instances {
		wg.Add(1)
		go func(inst *types.Instance) {
			defer wg.Done()

			childKernel := NewKernel()
			k.mu.Lock()
			k.childKernels = append(k.childKernels, childKernel)
			k.mu.Unlock()

			container := childKernel.traverser.Container()
			for _, fn := range services {
				container.Register(fn, "", true)
			}

			for _, initFn := range inst.Initializers {
				if err := initFn(container); err != nil {
					errChan <- fmt.Errorf("[%s] initialization error: %w", inst.Name, err)
					return
				}
			}

			if err := childKernel.traverser.Traverse(inst.Module); err != nil {
				errChan <- fmt.Errorf("[%s] module traversal error: %w", inst.Name, err)
				return
			}

			if err := childKernel.traverser.OnBootHooks().ExecuteAll(func(fn func(types.Kernel) error) error {
				return fn(childKernel)
			}); err != nil {
				errChan <- fmt.Errorf("[%s] boot hook error: %w", inst.Name, err)
				return
			}

			app, err := inst.Platform.Boot(container)
			if err != nil {
				errChan <- fmt.Errorf("[%s] platform boot error: %w", inst.Name, err)
				return
			}

			k.mu.Lock()
			k.runningApps = append(k.runningApps, app)
			k.mu.Unlock()

			if err := app.Run(childKernel.createHandler()); err != nil {
				errChan <- fmt.Errorf("[%s] runtime error: %w", inst.Name, err)
			}
		}(inst)
	}

	// Wait for shutdown signal or error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		return stop, nil
	case err := <-errChan:
		k.shutdown()
		return stop, err
	}
}

// createHandler creates the request handler function
func (k *kernel) createHandler() func(context types.Context) error {
	return func(context types.Context) error {
		routes := k.Routes()
		route, params, err := k.router.Find(routes, context.Request().Method().String(), context.Request().Paths())
		if err != nil {
			return err
		}

		context.Request().PopulateParams(params)

		for _, middleware := range route.Middlewares {
			if err := middleware.Handle(context); err != nil {
				return err
			}
		}

		output, err := k.processHandler(route, context)
		if err != nil {
			return err
		}

		if headers := output.Headers(); headers != nil {
			context.Response().SetHeaders(headers)
		}

		if contentType := output.ContentType(); contentType != "" {
			context.Response().SetHeader("Content-Type", contentType)
		}

		serialType, buf, err := k.serializer.Serialize(output.Data())
		if err != nil {
			return err
		}

		return context.Response().Write(serialType, buf, output.Code())
	}
}

// shutdown gracefully shuts down all running apps
func (k *kernel) shutdown() error {
	var shutdownErr error
	k.shutdownOnce.Do(func() {
		k.mu.Lock()
		apps := k.runningApps
		childKernels := k.childKernels
		k.mu.Unlock()

		// Shutdown all apps
		for _, app := range apps {
			if err := app.Shutdown(); err != nil && shutdownErr == nil {
				shutdownErr = err
			}
		}

		// Execute shutdown hooks for main kernel
		k.traverser.OnShutdownHooks().ExecuteAll(func(fn func(types.Kernel) error) error {
			return fn(k)
		})

		// Execute shutdown hooks for child kernels
		for _, child := range childKernels {
			child.traverser.OnShutdownHooks().ExecuteAll(func(fn func(types.Kernel) error) error {
				return fn(child)
			})
		}
	})
	return shutdownErr
}

func (k *kernel) Router() types.RouterFinder {
	return k.router
}

func (k *kernel) Routes() []types.Route {
	return k.routes
}

func (k *kernel) AppendRoutes(routes ...types.Route) ([]types.Route, error) {
	for _, newRoute := range routes {
		// Only validate handler if the route has one (group routes might not have handlers)
		if newRoute.Handler != nil && !k.isValidHandler(&newRoute) {
			return k.routes, fmt.Errorf("%w: %s %s", ErrInvalidRoute, newRoute.Method, newRoute.Path)
		}

		for _, existingRoute := range k.routes {
			if existingRoute.Equals(&newRoute) {
				return k.routes, fmt.Errorf("%w: %s %s", ErrDuplicateRoute, newRoute.Method, newRoute.Path)
			}
		}
		k.routes = append(k.routes, newRoute)
	}
	return k.routes, nil
}

func (k *kernel) Container() types.Container {
	return k.traverser.Container()
}

func (k *kernel) Registry() types.Registry {
	return k.traverser.Registry()
}

func (k *kernel) processHandler(route *types.Route, context types.Context) (types.Output, error) {
	var output types.Output
	switch handler := route.Handler.(type) {
	case []any:
		if len(handler) != 2 {
			return nil, fmt.Errorf("invalid handler format: expected [controller, string], got %v", handler)
		}

		controller := handler[0]
		methodName, ok := handler[1].(string)
		if !ok {
			return nil, fmt.Errorf("invalid handler format: method name must be a string, got %T", handler[1])
		}

		resolvedController, err := k.Container().Create(controller)
		if err != nil {
			return nil, fmt.Errorf("failed to make controller: %w", err)
		}

		controllerValue := reflect.ValueOf(resolvedController)
		method := controllerValue.MethodByName(methodName)

		if !method.IsValid() {
			return nil, fmt.Errorf("method %s not found on controller %T", methodName, resolvedController)
		}

		methodType := method.Type()
		if methodType.NumIn() != 1 {
			return nil, fmt.Errorf("method %s must have exactly 1 argument, got %d", methodName, methodType.NumIn())
		}

		argType := methodType.In(0)
		if argType.Kind() != reflect.Ptr || argType.Elem().Kind() != reflect.Struct {
			return nil, fmt.Errorf("method %s argument must be a pointer to a struct, got %v", methodName, argType)
		}

		argPtr := reflect.New(argType.Elem())

		inp := input.NewInput(context)
		err = inp.Populate(argPtr.Interface())
		if err != nil {
			return nil, fmt.Errorf("failed to hydrate input struct: %w", err)
		}

		args := []reflect.Value{argPtr}
		results := method.Call(args)

		if len(results) > 0 {
			output = (results[0].Interface()).(types.Output)
		}
	default:
		handlerValue := reflect.ValueOf(route.Handler)
		handlerType := handlerValue.Type()

		if handlerType.Kind() == reflect.Func &&
			handlerType.NumIn() == 1 &&
			handlerType.In(0).Kind() == reflect.Ptr &&
			handlerType.In(0).Elem().Kind() == reflect.Struct &&
			handlerType.NumOut() == 1 {

			paramType := handlerType.In(0).Elem()
			paramPtr := reflect.New(paramType)

			input := input.NewInput(context)
			err := input.Populate(paramPtr.Interface())
			if err != nil {
				return nil, fmt.Errorf("failed to hydrate input struct: %w", err)
			}

			results := handlerValue.Call([]reflect.Value{paramPtr})
			if len(results) > 0 {
				output = (results[0].Interface()).(types.Output)
			}
		} else {
			return nil, fmt.Errorf("unsupported handler type: %T", route.Handler)
		}
	}

	return output, nil
}

func (k *kernel) isValidHandler(route *types.Route) bool {
	switch handler := route.Handler.(type) {
	case []any:
		if len(handler) != 2 {
			return false
		}

		controller := handler[0]
		methodName, ok := handler[1].(string)
		if !ok {
			return false
		}

		resolvedController, err := k.Container().Create(controller)
		if err != nil {
			return false
		}

		controllerValue := reflect.ValueOf(resolvedController)
		method := controllerValue.MethodByName(methodName)

		if !method.IsValid() {
			return false
		}

		methodType := method.Type()
		if methodType.NumIn() != 1 {
			return false
		}

		argType := methodType.In(0)
		if argType.Kind() != reflect.Ptr || argType.Elem().Kind() != reflect.Struct {
			return false
		}
	default:
		handlerValue := reflect.ValueOf(route.Handler)
		handlerType := handlerValue.Type()

		if !(handlerType.Kind() == reflect.Func &&
			handlerType.NumIn() == 1 &&
			handlerType.In(0).Kind() == reflect.Ptr &&
			handlerType.In(0).Elem().Kind() == reflect.Struct &&
			handlerType.NumOut() == 1) {
			return false
		}
	}

	return true
}
