package types

type Importable interface {
	Imports() []Module
}

type Exportable interface {
	Exports() []any
}

type Declarable interface {
	Declarations() []any
}

type Global interface {
	IsGlobal() bool
}

type Module interface {
	Importable
	Exportable
	Declarable
}
