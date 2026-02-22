package types

type App interface {
	Run(fn func(c Context) error) error
}
