package types

type Bootable interface {
	Boot(k Kernel) error
}

type Shutdownable interface {
	Shutdown(k Kernel) error
}
