// (c) https://github.com/golobby/container/tree/master
// Package container is a lightweight yet powerful IoC container for Go projects.
// It provides an easy-to-use interface and performance-in-mind container to be your ultimate requirement.
package core

import (
	"reflect"
	"sync"
	"unsafe"

	"github.com/awesome-goose/goose/errors"
	"github.com/awesome-goose/goose/types"
)

// binding holds a binding resolver and an instance (for singleton bindings).
type binding struct {
	resolver any // resolver function that creates the appropriate implementation of the related abstraction
	instance any // instance stored for reusing in singleton bindings
}

// resolve creates an appropriate implementation of the related abstraction
func (b binding) resolve(c *Container) (any, error) {
	if b.instance != nil {
		return b.instance, nil
	}

	if b.resolver == nil {
		return nil, errors.ErrBindingNoResolverOrInstance
	}

	instance, err := c.invoke(b.resolver)
	if err != nil {
		return nil, err
	}

	c.onResolve(instance)

	return instance, nil
}

// Container holds all of the declared bindings
type Container struct {
	mu       sync.RWMutex
	bindings map[reflect.Type]map[string]binding
}

// NewContainer creates a new instance of the Container
func NewContainer() *Container {
	return &Container{
		bindings: make(map[reflect.Type]map[string]binding),
	}
}

// invoke calls a function and returns the yielded value.
// It only works for functions that return a single value.
func (c *Container) invoke(function any) (any, error) {
	if function == nil {
		return nil, errors.ErrCannotInvokeNilFunction
	}

	funcType := reflect.TypeOf(function)
	if funcType == nil || funcType.Kind() != reflect.Func {
		return nil, errors.ErrResolverMustBeFunction
	}

	args, err := c.arguments(function)
	if err != nil {
		return nil, err
	}

	numOut := funcType.NumOut()
	if numOut == 1 {
		return reflect.ValueOf(function).Call(args)[0].Interface(), nil
	} else if numOut == 2 {
		values := reflect.ValueOf(function).Call(args)
		if values[1].IsNil() {
			return values[0].Interface(), nil
		}
		if err, ok := values[1].Interface().(error); ok {
			return values[0].Interface(), err
		}
		return values[0].Interface(), errors.ErrSecondReturnNotError.WithMeta(values[1].Interface())
	}

	return nil, errors.ErrInvalidResolverSignature
}

// arguments returns container-resolved arguments of a function.
func (c *Container) arguments(function any) ([]reflect.Value, error) {
	reflectedFunction := reflect.TypeOf(function)
	if reflectedFunction == nil {
		return nil, errors.ErrCannotGetArgumentsForNil
	}

	argumentsCount := reflectedFunction.NumIn()
	arguments := make([]reflect.Value, argumentsCount)

	for i := 0; i < argumentsCount; i++ {
		abstraction := reflectedFunction.In(i)

		c.mu.RLock()
		bindingMap, exists := c.bindings[abstraction]
		var concrete binding
		if exists {
			concrete, exists = bindingMap[""]
		}
		c.mu.RUnlock()

		if exists {
			instance, err := concrete.resolve(c)
			if err != nil {
				return nil, errors.ErrFailedToResolveArgument.WithError(err).WithMeta(map[string]any{"index": i, "type": abstraction.String()})
			}
			arguments[i] = reflect.ValueOf(instance)
		} else {
			return nil, errors.ErrNoConcreteFound.WithMeta(abstraction.String())
		}
	}

	return arguments, nil
}

// create is the recursive helper for Create.
//
// Not safe for concurrent invocation on the same struct: it sets fields via
// unsafe pointers between mutex sections, which is fine for a single
// initializer per struct but races if two goroutines populate the same value.
func (c *Container) create(v reflect.Value, visited map[reflect.Type]bool) error {
	t := v.Type()
	if visited[t] {
		return errors.ErrCircularDependency.WithMeta(t.String())
	}
	visited[t] = true
	defer delete(visited, t)

	s := v.Elem()
	sType := s.Type()

	for i := 0; i < s.NumField(); i++ {
		field := s.Field(i)
		fieldType := field.Type()

		if tagValue, exist := sType.Field(i).Tag.Lookup("inject"); exist {
			var name string
			switch tagValue {
			case "type", "":
				name = ""
			case "name":
				name = sType.Field(i).Name
			default:
				name = tagValue
			}

			// Process only pointer to struct and interface fields.
			isPtrToStruct := fieldType.Kind() == reflect.Ptr && fieldType.Elem().Kind() == reflect.Struct
			isInterface := fieldType.Kind() == reflect.Interface

			if !isPtrToStruct && !isInterface {
				continue
			}

			// 0. Check for existing binding under a write lock to ensure atomicity.
			c.mu.Lock()
			bindingMap, mapExists := c.bindings[fieldType]
			if !mapExists {
				bindingMap = make(map[string]binding)
				c.bindings[fieldType] = bindingMap
			}
			existingBinding, bindingExists := bindingMap[name]

			if bindingExists && existingBinding.instance != nil {
				c.mu.Unlock() // Unlock before resolving to avoid deadlock
				instance, err := existingBinding.resolve(c)
				if err != nil {
					return err
				}
				ptr := reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
				ptr.Set(reflect.ValueOf(instance))
				continue
			}

			// No existing instance, so we'll create one.
			var instance any
			var err error

			// Unlock before potentially heavy operations or calls that might re-lock.
			c.mu.Unlock()

			// 1. Use a pre-registered resolver if one exists.
			if bindingExists && existingBinding.resolver != nil {
				instance, err = c.invoke(existingBinding.resolver)
				if err != nil {
					return errors.ErrFailedToResolveField.WithError(err).WithMeta(sType.Field(i).Name)
				}
			} else {
				if isInterface {
					return errors.ErrCannotCreateInterfaceField.WithMeta(map[string]any{"field": sType.Field(i).Name, "type": sType.Name()})
				}

				// 2. Use a zero value of the struct for pointer to struct fields.
				if isPtrToStruct {
					instance = reflect.New(fieldType.Elem()).Interface()
				}
			}

			// Re-lock to safely write the new instance (double-check pattern).
			c.mu.Lock()
			// Check again if another goroutine created the binding while we were working
			if existingB, exists := c.bindings[fieldType][name]; exists && existingB.instance != nil {
				// Another goroutine beat us to it, use the existing instance
				c.mu.Unlock()
				instance = existingB.instance
			} else {
				c.bindings[fieldType][name] = binding{instance: instance}
				c.mu.Unlock()

				// Recursively call create for the new instance if it's a struct pointer.
				// Only do this if WE created the instance (not if we're using an existing one).
				if isPtrToStruct {
					if err := c.create(reflect.ValueOf(instance), visited); err != nil {
						return err
					}
				}
			}

			// Set the resolved field on the struct.
			ptr := reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
			ptr.Set(reflect.ValueOf(instance))
		}
	}

	c.onRegister(v.Interface())

	return nil
}

// Register maps an abstraction to a concrete and sets an instance if it's a singleton binding.
func (c *Container) Register(resolver any, name string, singleton bool) error {
	if resolver == nil {
		return errors.ErrResolverCannotBeNil
	}

	reflectedResolver := reflect.TypeOf(resolver)
	if reflectedResolver == nil || reflectedResolver.Kind() != reflect.Func {
		return errors.ErrInvalidResolver
	}

	for i := 0; i < reflectedResolver.NumOut(); i++ {
		outType := reflectedResolver.Out(i)

		c.mu.Lock()
		if _, exist := c.bindings[outType]; !exist {
			c.bindings[outType] = make(map[string]binding)
		}

		if singleton {
			c.mu.Unlock() // Unlock before invoke to avoid deadlock
			instance, err := c.invoke(resolver)
			if err != nil {
				return err
			}

			c.onRegister(instance)

			c.mu.Lock()
			c.bindings[outType][name] = binding{resolver: resolver, instance: instance}
			c.mu.Unlock()
		} else {
			c.bindings[outType][name] = binding{resolver: resolver}
			c.mu.Unlock()
		}
	}

	return nil
}

// Resolve resolves a binding and sets it to the provided abstraction pointer.
func (c *Container) Resolve(abstraction any, name string) error {
	receiverType := reflect.TypeOf(abstraction)
	if receiverType == nil {
		return errors.ErrInvalidAbstraction
	}

	if receiverType.Kind() == reflect.Ptr {
		elem := receiverType.Elem()

		c.mu.RLock()
		bindingMap, mapExists := c.bindings[elem]
		var concrete binding
		var exists bool
		if mapExists {
			concrete, exists = bindingMap[name]
		}
		c.mu.RUnlock()

		if exists {
			instance, err := concrete.resolve(c)
			if err != nil {
				return err
			}
			reflect.ValueOf(abstraction).Elem().Set(reflect.ValueOf(instance))
			c.onResolve(instance)
			return nil
		}

		return errors.ErrNoConcreteFound.WithMeta(elem.String())
	}

	return errors.ErrInvalidAbstraction
}

// Call takes a function (receiver) with one or more arguments of the abstractions (interfaces).
// It invokes the function (receiver) and passes the related implementations.
func (c *Container) Call(function any) error {
	receiverType := reflect.TypeOf(function)
	if receiverType == nil || receiverType.Kind() != reflect.Func {
		return errors.ErrInvalidFunction
	}

	arguments, err := c.arguments(function)
	if err != nil {
		return err
	}

	reflect.ValueOf(function).Call(arguments)

	return nil
}

// Fill takes a struct and resolves the fields with the tag `inject:""`
func (c *Container) Fill(structure any) error {
	receiverType := reflect.TypeOf(structure)
	if receiverType == nil {
		return errors.ErrInvalidStructure
	}

	if receiverType.Kind() == reflect.Ptr {
		elem := receiverType.Elem()
		if elem.Kind() == reflect.Struct {
			s := reflect.ValueOf(structure).Elem()

			for i := 0; i < s.NumField(); i++ {
				f := s.Field(i)

				if tagValue, exist := s.Type().Field(i).Tag.Lookup("inject"); exist {
					var name string

					switch tagValue {
					case "type", "":
						name = ""
					case "name":
						name = s.Type().Field(i).Name
					default:
						name = tagValue
					}

					c.mu.RLock()
					bindingMap, mapExists := c.bindings[f.Type()]
					var concrete binding
					var exists bool
					if mapExists {
						concrete, exists = bindingMap[name]
					}
					c.mu.RUnlock()

					if exists {
						instance, err := concrete.resolve(c)
						if err != nil {
							return errors.ErrFailedToResolveField.WithError(err).WithMeta(s.Type().Field(i).Name)
						}

						ptr := reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
						ptr.Set(reflect.ValueOf(instance))

						continue
					}

					return errors.ErrCannotResolveField.WithMeta(s.Type().Field(i).Name)
				}
			}

			return nil
		}
	}

	return errors.ErrInvalidStructure
}

// Create creates and registers a struct and its dependencies recursively.
// It's purely singleton-based.
//
// Not safe for concurrent use on the same struct. Container population is a
// one-shot boot-time operation; callers must serialize Register/Create/Fill
// calls. The internal recursive helper writes resolved fields via unsafe
// pointers under the assumption that no other goroutine is touching the same
// struct concurrently.
func (c *Container) Create(value any) (any, error) {
	if value == nil {
		return nil, errors.ErrCreateValueCannotBeNil
	}

	v := reflect.ValueOf(value)
	t := v.Type()

	var ptrV reflect.Value

	if t.Kind() == reflect.Struct {
		ptrT := reflect.PointerTo(t)

		c.mu.RLock()
		bindingMap, mapExists := c.bindings[ptrT]
		var b binding
		var exists bool
		if mapExists {
			b, exists = bindingMap[""]
		}
		c.mu.RUnlock()

		if exists && b.instance != nil {
			return b.instance, nil
		}

		ptrV = reflect.New(t)
	} else if t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct {
		c.mu.RLock()
		bindingMap, mapExists := c.bindings[t]
		var b binding
		var exists bool
		if mapExists {
			b, exists = bindingMap[""]
		}
		c.mu.RUnlock()

		if exists && b.instance != nil {
			return b.instance, nil
		}

		ptrV = v
	} else {
		return nil, errors.ErrInvalidAbstraction
	}

	visited := make(map[reflect.Type]bool)

	if err := c.create(ptrV, visited); err != nil {
		return nil, err
	}

	return ptrV.Interface(), nil
}

// Reset deletes all the existing bindings and empties the container instance.
func (c *Container) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for k := range c.bindings {
		delete(c.bindings, k)
	}
}

// Close calls OnClose on all CloseAware instances and resets the container.
// Returns the first error encountered, but attempts to close all instances.
func (c *Container) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var firstErr error

	for _, bindingMap := range c.bindings {
		for _, b := range bindingMap {
			if b.instance != nil {
				if closer, ok := b.instance.(types.CloseAware); ok {
					if err := closer.OnClose(); err != nil && firstErr == nil {
						firstErr = err
					}
				}
			}
		}
	}

	// Clear all bindings
	for k := range c.bindings {
		delete(c.bindings, k)
	}

	return firstErr
}

func (c *Container) onRegister(instance any) {
	if registrar, ok := instance.(types.RegisterAware); ok {
		registrar.OnRegister()
	}
}

func (c *Container) onResolve(instance any) {
	if resolver, ok := instance.(types.ResolveAware); ok {
		resolver.OnResolve()
	}
}
