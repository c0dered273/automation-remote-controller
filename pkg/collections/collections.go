package collections

import "sync"

type Signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

type Unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

type Integer interface {
	Signed | Unsigned
}

type Float interface {
	~float32 | ~float64
}

type Ordered interface {
	Integer | Float | ~string
}

type ConcurrentMap[T Ordered, E any] struct {
	mx         *sync.RWMutex
	storageMap map[T]E
}

func (m *ConcurrentMap[T, E]) Get(key T) (E, bool) {
	m.mx.RLock()
	defer m.mx.RUnlock()
	v, ok := m.storageMap[key]
	return v, ok
}

func (m *ConcurrentMap[T, E]) Put(key T, value E) {
	m.mx.Lock()
	defer m.mx.Unlock()
	m.storageMap[key] = value
}

func (m *ConcurrentMap[T, E]) IterateKeys() <-chan T {
	c := make(chan T)
	go func() {
		m.mx.RLock()
		defer m.mx.RUnlock()
		for k := range m.storageMap {
			c <- k
		}
		close(c)
	}()
	return c
}

func (m *ConcurrentMap[T, E]) IterateValues() <-chan E {
	c := make(chan E)
	go func() {
		m.mx.RLock()
		defer m.mx.RUnlock()
		for _, v := range m.storageMap {
			c <- v
		}
		close(c)
	}()
	return c
}

func NewConcurrentMap[T Ordered, E any]() *ConcurrentMap[T, E] {
	return &ConcurrentMap[T, E]{
		mx:         new(sync.RWMutex),
		storageMap: make(map[T]E),
	}
}
