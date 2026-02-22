package types

type Migration interface {
	Up() error
}
