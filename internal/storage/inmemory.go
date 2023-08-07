package storage

type Pair[K, V any] struct {
	Key   K
	Value V
}

type Storage[K, V comparable] interface {

	// Get returns the value associated with the key.
	// If the key is not found, it returns the zero value of the value type and false.
	Get(key K) (V, bool)

	// Set sets/updates the value associated with the key.
	Set(key K, value V)

	// Delete deletes the value associated with the key.
	Delete(key K)

	// Len returns the number of elements in the storage.
	Len() int

	// GetAll returns all elements in the storage.
	GetAll() []Pair[K, V]
}

type InMemoryStorage[K, V comparable] struct {
	storage map[K]V
}

func NewInMemoryStorage[K, V comparable]() Storage[K, V] {
	return &InMemoryStorage[K, V]{storage: make(map[K]V)}
}

func (i *InMemoryStorage[K, V]) Get(key K) (V, bool) {
	value, ok := i.storage[key]
	return value, ok
}

func (i *InMemoryStorage[K, V]) Set(key K, value V) {
	i.storage[key] = value
}

func (i *InMemoryStorage[K, V]) Delete(key K) {
	if _, ok := i.storage[key]; !ok {
		return
	}

	delete(i.storage, key)
}

func (i *InMemoryStorage[K, V]) Len() int {
	return len(i.storage)
}

func (i *InMemoryStorage[K, V]) GetAll() []Pair[K, V] {
	pairs := make([]Pair[K, V], 0, len(i.storage))
	for key, value := range i.storage {
		pairs = append(pairs, Pair[K, V]{Key: key, Value: value})
	}

	return pairs
}
