package core

import (
	"bytes"
	gocontext "context"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"syscall"
	"time"

	"github.com/awesome-goose/goose/errors"
	"github.com/awesome-goose/goose/io/input"
	"github.com/awesome-goose/goose/types"
)

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
	// shutdownOnce is a pointer so shutdown() can swap in a fresh one once a
	// cycle finishes (see the reset at the end of shutdown()) — a plain
	// sync.Once value would only ever run its Do body once for this
	// kernel's whole lifetime, which is fine for the single Start() per
	// process every caller used to do, but not for a kernel Start()ed more
	// than once (e.g. a test binary with more than one test that each boot
	// their own server on the same *kernel).
	shutdownOnce *sync.Once

	// streamsCtx is cancelled at the very start of shutdown() so open
	// streaming responses end BEFORE the HTTP servers are asked to drain —
	// otherwise Server.Shutdown waits (up to the platform's shutdown
	// timeout) for connections that have no reason to ever finish.
	streamsCtx    gocontext.Context
	cancelStreams gocontext.CancelFunc
}

func NewKernel() *kernel {
	k := &kernel{
		router:       NewRouter(),
		routes:       []types.Route{},
		serializer:   NewSerializer(),
		traverser:    NewTraverser(),
		runningApps:  make([]types.App, 0),
		childKernels: make([]*kernel, 0),
		shutdownOnce: &sync.Once{},
	}
	k.streamsCtx, k.cancelStreams = gocontext.WithCancel(gocontext.Background())
	return k
}

// Start starts one or more platform instances
// - Single instance: runs directly
// - Multiple instances: API/Web run concurrently, CLI runs when `cli` arg is passed
func (k *kernel) Start(instances ...*types.Instance) (func() error, error) {
	if len(instances) == 0 {
		return func() error { return nil }, errors.ErrNoInstancesProvided
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
		if err := container.Register(fn, "", true); err != nil {
			return stop, errors.ErrServiceRegistrationError.WithError(err)
		}
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
		return func() error { return nil }, errors.ErrMultipleCLIInstances.WithMeta(cliCount)
	}

	// Check if CLI mode is requested via command line args
	cliMode := len(os.Args) > 1 && os.Args[1] == "cli"

	// If CLI mode requested but no CLI instance defined
	if cliMode && cliInstance == nil {
		return func() error { return nil }, errors.ErrCLIModeNoCLIInstance
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
		return func() error { return nil }, errors.ErrNoRunnableInstances
	}

	return k.runServers(serverInstances)
}

// runCLI runs a CLI instance in the main goroutine (blocking)
func (k *kernel) runCLI(inst *types.Instance) (func() error, error) {
	// Strip the "cli" mode selector (used by runMulti to route between
	// platforms) before the CLI platform parses os.Args, so a multi-instance
	// app's Request sees the same argument shape as a single-instance CLI
	// app: os.Args[1:] holds only the real command args, with no "cli/"
	// prefix required on registered routes.
	if len(os.Args) > 1 && os.Args[1] == "cli" {
		os.Args = append(os.Args[:1], os.Args[2:]...)
	}

	childKernel := NewKernel()
	k.mu.Lock()
	k.childKernels = append(k.childKernels, childKernel)
	k.mu.Unlock()

	stop := func() error {
		return k.shutdown()
	}

	container := childKernel.traverser.Container()
	for _, fn := range services {
		if err := container.Register(fn, "", true); err != nil {
			return stop, errors.ErrServiceRegistrationError.WithError(err)
		}
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
				if err := container.Register(fn, "", true); err != nil {
					errChan <- errors.ErrServiceRegistrationError.WithError(err).WithMeta(inst.Name)
					return
				}
			}

			for _, initFn := range inst.Initializers {
				if err := initFn(container); err != nil {
					errChan <- errors.ErrInitializationError.WithError(err).WithMeta(inst.Name)
					return
				}
			}

			if err := childKernel.traverser.Traverse(inst.Module); err != nil {
				errChan <- errors.ErrModuleTraversalError.WithError(err).WithMeta(inst.Name)
				return
			}

			if err := childKernel.traverser.OnBootHooks().ExecuteAll(func(fn func(types.Kernel) error) error {
				return fn(childKernel)
			}); err != nil {
				errChan <- errors.ErrBootHookError.WithError(err).WithMeta(inst.Name)
				return
			}

			app, err := inst.Platform.Boot(container)
			if err != nil {
				errChan <- errors.ErrPlatformBootError.WithError(err).WithMeta(inst.Name)
				return
			}

			k.mu.Lock()
			k.runningApps = append(k.runningApps, app)
			k.mu.Unlock()

			if err := app.Run(childKernel.createHandler()); err != nil {
				errChan <- errors.ErrRuntimeError.WithError(err).WithMeta(inst.Name)
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

		if streamOutput, ok := output.(types.StreamOutput); ok {
			return k.writeStream(context, streamOutput)
		}

		serialType, buf, err := k.serializer.Serialize(output.Data())
		if err != nil {
			return err
		}

		return context.Response().Write(serialType, buf, output.Code())
	}
}

// writeStream serves a types.StreamOutput (U-G2): on an HTTP platform (any
// Response whose Raw() is an http.ResponseWriter — spa, api, web), it clears
// the write deadline the same way the F2 fix already does for a single
// response (a stream must be able to outlive a fixed WriteTimeout) and
// flushes after every chunk so the client sees it immediately rather than
// buffered until the connection closes. Headers/Content-Type are already set
// on context.Response() by createHandler before this runs.
//
// On a non-HTTP platform (Raw() isn't an http.ResponseWriter — e.g. CLI),
// there is no live connection to flush incrementally against, so the
// callback's output is buffered and written once via the normal Response
// .Write path — the handler still works, just without the incremental
// delivery that only makes sense over a real connection.
func (k *kernel) writeStream(context types.Context, output types.StreamOutput) error {
	raw, isHTTP := context.Response().Raw().(http.ResponseWriter)
	if !isHTTP {
		var buf bytes.Buffer
		if err := output.StreamCallback()(func(chunk []byte) error {
			_, err := buf.Write(chunk)
			return err
		}); err != nil {
			return err
		}
		return context.Response().Write(types.SerialTypeBinary, buf.Bytes(), output.Code())
	}

	rc := http.NewResponseController(raw)
	_ = rc.SetWriteDeadline(time.Time{}) // F2: a stream must outlive any fixed WriteTimeout
	raw.WriteHeader(output.Code())
	var perWrite time.Duration
	if wt, ok := output.(types.WriteTimeoutStreamOutput); ok {
		perWrite = wt.StreamWriteTimeout()
	}
	write := func(chunk []byte) error {
		if perWrite > 0 {
			_ = rc.SetWriteDeadline(time.Now().Add(perWrite))
		}
		if _, err := raw.Write(chunk); err != nil {
			return err
		}
		if err := rc.Flush(); err != nil && err != http.ErrNotSupported {
			return err
		}
		return nil
	}

	if cs, ok := output.(types.ContextStreamOutput); ok {
		// Cancelled on client disconnect (request context) or kernel shutdown.
		parent := gocontext.Background()
		if rq, ok := context.Request().(interface{ Context() gocontext.Context }); ok {
			parent = rq.Context()
		}
		ctx, cancel := gocontext.WithCancel(parent)
		defer cancel()
		k.mu.Lock()
		shutdown := k.streamsCtx
		k.mu.Unlock()
		stop := gocontext.AfterFunc(shutdown, cancel)
		defer stop()
		return cs.StreamContextCallback()(ctx, write)
	}
	return output.StreamCallback()(write)
}

// shutdown gracefully shuts down all running apps
func (k *kernel) shutdown() error {
	var shutdownErr error
	k.shutdownOnce.Do(func() {
		k.mu.Lock()
		cancelStreams := k.cancelStreams
		apps := k.runningApps
		childKernels := k.childKernels
		k.mu.Unlock()

		// End open streams first so the servers below can actually drain.
		if cancelStreams != nil {
			cancelStreams()
		}

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

		// Reset mutable state back to what NewKernel() produces. Before
		// this, routes and the traverser's boot-hook/declaration graph
		// persisted on the kernel forever: a second Start() on the same
		// kernel (any test binary with more than one test that each boot a
		// server, e.g. cloud/server's own suite gaining a second such test)
		// re-ran every prior Start's boot hooks on top of the new one's —
		// traverser.collectHooks() only dedupes *within* a single
		// Traverse() call — which re-registered the same routes a second
		// time. AppendRoutes correctly rejects that as a genuine duplicate
		// (same method+path, both with a real handler), so the second
		// Start failed outright with DUPLICATE_ROUTE. shutdownOnce is
		// swapped for a fresh one too, or this reset would itself only
		// ever run once per kernel.
		k.mu.Lock()
		k.routes = []types.Route{}
		k.router = NewRouter()
		k.traverser = NewTraverser()
		k.runningApps = make([]types.App, 0)
		k.childKernels = make([]*kernel, 0)
		k.streamsCtx, k.cancelStreams = gocontext.WithCancel(gocontext.Background())
		k.mu.Unlock()
		k.shutdownOnce = &sync.Once{}
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
			return k.routes, errors.ErrInvalidRoute.WithMeta(map[string]any{"method": newRoute.Method, "path": newRoute.Path})
		}

		merged := false
		for i := range k.routes {
			existing := &k.routes[i]
			if !existing.Equals(&newRoute) {
				continue
			}

			// Two pure groups (no handler on either side) at the same
			// Method+Path are not a conflict — merge their children so a
			// router.Mount(prefix, ...) wrapping multiple staticRouters can
			// land all of them under a single shared prefix route.
			if existing.Handler == nil && newRoute.Handler == nil {
				existing.Children = append(existing.Children, newRoute.Children...)
				merged = true
				break
			}

			return k.routes, errors.ErrDuplicateRoute.WithMeta(map[string]any{"method": newRoute.Method, "path": newRoute.Path})
		}

		if !merged {
			k.routes = append(k.routes, newRoute)
		}
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
			return nil, errors.ErrInvalidHandlerFormat.WithMeta(handler)
		}

		controller := handler[0]
		methodName, ok := handler[1].(string)
		if !ok {
			return nil, errors.ErrMethodNameNotString.WithMeta(handler[1])
		}

		resolvedController, err := k.Container().Create(controller)
		if err != nil {
			return nil, errors.ErrFailedToMakeController.WithError(err)
		}

		controllerValue := reflect.ValueOf(resolvedController)
		method := controllerValue.MethodByName(methodName)

		if !method.IsValid() {
			return nil, errors.ErrMethodNotFound.WithMeta(map[string]any{"method": methodName, "controller": resolvedController})
		}

		methodType := method.Type()
		if methodType.NumIn() != 1 {
			return nil, errors.ErrMethodWrongArgCount.WithMeta(map[string]any{"method": methodName, "count": methodType.NumIn()})
		}

		argType := methodType.In(0)
		if argType.Kind() != reflect.Ptr || argType.Elem().Kind() != reflect.Struct {
			return nil, errors.ErrMethodArgNotPtrStruct.WithMeta(map[string]any{"method": methodName, "type": argType})
		}

		argPtr := reflect.New(argType.Elem())

		inp := input.NewInput(context)
		err = inp.Populate(argPtr.Interface())
		if err != nil {
			return nil, errors.ErrFailedToHydrateInput.WithError(err)
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
				return nil, errors.ErrFailedToHydrateInput.WithError(err)
			}

			results := handlerValue.Call([]reflect.Value{paramPtr})
			if len(results) > 0 {
				output = (results[0].Interface()).(types.Output)
			}
		} else {
			return nil, errors.ErrUnsupportedHandlerType.WithMeta(route.Handler)
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
