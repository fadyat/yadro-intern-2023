package storage

import "errors"

type Queue[T any] interface {
	// Push pushes an element to the queue.
	Push(T)

	// Pop pops an element from the queue.
	// It returns an error if the queue is empty.
	Pop() (T, error)

	// Len returns the number of elements in the queue.
	Len() int

	// Peek returns the first element of the queue without removing it.
	// It returns an error if the queue is empty.
	Peek() (T, error)

	// Clear removes all elements from the queue.
	Clear()
}

type InMemoryQueue[T any] struct {
	queue    []T
	nilValue T
}

func NewInMemoryQueue[T any](nilValue T) Queue[T] {
	return &InMemoryQueue[T]{
		queue:    make([]T, 0),
		nilValue: nilValue,
	}
}

func (i *InMemoryQueue[T]) Push(value T) {
	i.queue = append(i.queue, value)
}

func (i *InMemoryQueue[T]) Pop() (T, error) {
	top, err := i.Peek()
	if err != nil {
		return i.nilValue, err
	}

	i.queue = i.queue[1:]
	return top, nil
}

func (i *InMemoryQueue[T]) Len() int {
	return len(i.queue)
}

func (i *InMemoryQueue[T]) Peek() (T, error) {
	if len(i.queue) == 0 {
		return i.nilValue, errors.New("queue is empty")
	}

	top := i.queue[0]
	return top, nil
}

func (i *InMemoryQueue[T]) Clear() {
	i.queue = i.queue[:0]
}
