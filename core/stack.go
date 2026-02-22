package core

import "sync"

// stack is a thread-safe generic stack implementation
type stack[T any] struct {
	mu    sync.RWMutex
	items []T
}

func NewStack[T any]() *stack[T] {
	return &stack[T]{
		items: make([]T, 0),
	}
}

func (s *stack[T]) Push(item T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, item)
}

func (s *stack[T]) Pop() (T, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.items) == 0 {
		var zero T
		return zero, false
	}
	lastIndex := len(s.items) - 1
	item := s.items[lastIndex]
	s.items = s.items[:lastIndex]
	return item, true
}

func (s *stack[T]) Peek() (T, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.items) == 0 {
		var zero T
		return zero, false
	}
	return s.items[len(s.items)-1], true
}

func (s *stack[T]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

// ExecuteAll executes the function on all items in LIFO order (last pushed, first executed)
func (s *stack[T]) ExecuteAll(fn func(T) error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := len(s.items) - 1; i >= 0; i-- {
		if err := fn(s.items[i]); err != nil {
			return err
		}
	}

	return nil
}
