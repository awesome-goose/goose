package types

type Traverser interface {
	Traverse(root Module) error
	Container() Container
	Registry() Registry
	OnBootHooks() Stack[func(Kernel) error]
	OnShutdownHooks() Stack[func(Kernel) error]
}
