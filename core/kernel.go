package core

import (
	"fmt"
	"reflect"

	"github.com/awesome-goose/goose/input"
	"github.com/awesome-goose/goose/types"
)

// ErrDuplicateRoute is returned when attempting to add a route that already exists.
var ErrDuplicateRoute = fmt.Errorf("kernel: duplicate route detected")
var ErrInvalidRoute = fmt.Errorf("kernel: invalid route detected")

type kernel struct {
	router     types.RouterFinder
	routes     types.Routes
	serializer types.Serializer
	traverser  types.Traverser
}

func NewKernel() *kernel {
	return &kernel{
		router:     NewRouter(),
		routes:     []types.Route{},
		serializer: NewSerializer(),
		traverser:  NewTraverser(),
	}
}

func (k *kernel) Start(platform types.Platform, module types.Module, initializers []func(container types.Container) error) (func() error, error) {
	stop := func() error {
		return k.traverser.OnShutdownHooks().ExecuteAll(func(fn func(types.Kernel) error) error {
			return fn(k)
		})
	}

	container := k.traverser.Container()
	for _, fn := range services {
		container.Register(fn, "", true)
	}

	for _, initFn := range initializers {
		err := initFn(container)
		if err != nil {
			return stop, err
		}
	}

	err := k.traverser.Traverse(module)
	if err != nil {
		return stop, err
	}

	err = k.traverser.OnBootHooks().ExecuteAll(func(fn func(types.Kernel) error) error {
		return fn(k)
	})
	if err != nil {
		return stop, err
	}

	app, err := platform.Boot(container)
	if err != nil {
		return stop, err
	}

	err = app.Run(func(context types.Context) error {
		routes := k.Routes()
		route, params, err := k.router.Find(routes, context.Request().Method().String(), context.Request().Paths())
		if err != nil {
			return err
		}

		context.Request().PopulateParams(params)

		for _, middleware := range route.Middlewares {
			err := middleware.Handle(context)
			if err != nil {
				return err
			}
		}

		output, err := k.processHandler(route, context)
		if err != nil {
			return err
		}

		// Set custom headers from output (if any)
		if headers := output.Headers(); headers != nil {
			context.Response().SetHeaders(headers)
		}

		// Set content type if explicitly specified by output
		if contentType := output.ContentType(); contentType != "" {
			context.Response().SetHeader("Content-Type", contentType)
		}

		serialType, buf, err := k.serializer.Serialize(output.Data())
		if err != nil {
			return err
		}

		err = context.Response().Write(serialType, buf, output.Code())
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return stop, err
	}

	return stop, nil
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
