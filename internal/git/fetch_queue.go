package git

import (
	"sync"
	"sync/atomic"
)

const todoChanSize = 10_000
const doneMapSize = 1_000

type FetchQueue struct {
	mu      sync.Mutex
	todo    chan string
	done    map[string]bool // value itself is not used, for now.
	cntTodo atomic.Uint32
	cntDone atomic.Uint32
}

func NewFetchQueue() *FetchQueue {
	return &FetchQueue{
		mu:   sync.Mutex{},
		todo: make(chan string, todoChanSize),
		done: make(map[string]bool, doneMapSize),
	}
}

func (fq *FetchQueue) Add(path string) {
	fq.mu.Lock()
	_, isDone := fq.done[path]

	if !isDone {
		fq.done[path] = false
		fq.cntTodo.Add(1)
	}
	fq.mu.Unlock()

	if !isDone {
		// Add after unlock to avoid deadlock.
		fq.todo <- path
	}
}

func (fq *FetchQueue) MarkDone(path string) {
	fq.mu.Lock()
	fq.done[path] = true
	fq.cntDone.Add(1)
	fq.mu.Unlock()
}

func (fq *FetchQueue) Todo() <-chan string {
	return fq.todo
}

func (fq *FetchQueue) CountersEqual() (equals bool) {
	fq.mu.Lock()
	equals = fq.cntTodo.Load() == fq.cntDone.Load()
	fq.mu.Unlock()

	return
}

func (fq *FetchQueue) CntDone() int {
	return int(fq.cntDone.Load())
}

func (fq *FetchQueue) CntQueued() int {
	return int(fq.cntTodo.Load() - fq.cntDone.Load())
}

func (fq *FetchQueue) Close() {
	close(fq.todo)
}
