package types

type Stack[T any] interface {
	Push(T)
	Pop() (T, bool)
	Peek() (T, bool)
	Len() int
	ExecuteAll(func(T) error) error
}
