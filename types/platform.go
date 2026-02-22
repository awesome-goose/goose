package types

type Platform interface {
	Boot(container Container) (App, error)
}
