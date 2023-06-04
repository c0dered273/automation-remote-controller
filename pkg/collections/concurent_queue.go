package collections

import (
	"container/list"
	"sync"
)

type ConcurrentQueue[T any] struct {
	mx   *sync.Mutex
	list *list.List
}

func (q *ConcurrentQueue[T]) add(v T) {
	q.mx.Lock()
	q.list.PushBack(v)
	q.mx.Unlock()
}

func (q *ConcurrentQueue[T]) poll() T {
	q.mx.Lock()
	v := q.list.Front()
	q.list.Remove(v)
	q.mx.Unlock()
	return v.Value.(T)
}

func NewConcurrentQueue[T any]() *ConcurrentQueue[T] {
	return &ConcurrentQueue[T]{
		mx:   new(sync.Mutex),
		list: list.New(),
	}
}
