package git

import (
	"sync"
)

/* TODO
Remove this, and replace it with two channels. One for downloading, second for processing files.

The problem here is, what processing files adds new paths into the download queue,
so it it would be processed in one, we could have a deadlock.
*/

type FetchQueue struct {
	mu          sync.RWMutex
	files1Todo  map[string]string
	files2Doing map[string]string
	files3Done  map[string]string
	changesCnt  int
}

func NewFetchQueue() *FetchQueue {
	return &FetchQueue{
		mu:          sync.RWMutex{},
		files1Todo:  make(map[string]string),
		files2Doing: make(map[string]string),
		files3Done:  make(map[string]string),
	}
}

// Add adds a file to the queue. If it's done or doing, it's not re-added.
func (fq *FetchQueue) Add(file string) {
	fq.mu.Lock()
	defer fq.mu.Unlock()

	_, is2Doing := fq.files2Doing[file]
	_, is3Done := fq.files3Done[file]

	if !is2Doing && !is3Done {
		fq.files1Todo[file] = file
		fq.changesCnt++
	}

	return
}

func (fq *FetchQueue) Get() (file string, ok bool) {
	fq.mu.Lock()
	defer fq.mu.Unlock()

	if len(fq.files1Todo) == 0 {
		return "", false
	}

	for file = range fq.files1Todo {
		break
	}

	fq.files2Doing[file] = file
	delete(fq.files1Todo, file)

	fq.changesCnt++

	return file, true
}

func (fq *FetchQueue) Done(file string) {
	fq.mu.Lock()
	defer fq.mu.Unlock()

	fq.files3Done[file] = file
	delete(fq.files1Todo, file) // to be sure
	delete(fq.files2Doing, file)

	fq.changesCnt++

	return
}

func (fq *FetchQueue) HasThingsToDo() bool {
	fq.mu.RLock()
	defer fq.mu.RUnlock()

	return len(fq.files1Todo) > 0 || len(fq.files2Doing) > 0
}

func (fq *FetchQueue) DoneCnt() int {
	fq.mu.RLock()
	defer fq.mu.RUnlock()

	return len(fq.files3Done)
}

func (fq *FetchQueue) TodoCnt() any {
	fq.mu.RLock()
	defer fq.mu.RUnlock()

	return len(fq.files1Todo)
}

func (fq *FetchQueue) ChangesCnt() int {
	fq.mu.RLock()
	defer fq.mu.RUnlock()

	return fq.changesCnt
}
