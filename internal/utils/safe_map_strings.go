package utils

import (
	"sync"
)

type SafeMapStrings struct {
	mu     sync.RWMutex
	values map[string]string
}

func NewSafeMapStrings() *SafeMapStrings {
	return &SafeMapStrings{
		mu:     sync.RWMutex{},
		values: make(map[string]string),
	}
}

func (sm *SafeMapStrings) Lock() {
	sm.mu.Lock()
}

func (sm *SafeMapStrings) Unlock() {
	sm.mu.Unlock()
}

func (sm *SafeMapStrings) Exists(key string) (exists bool) {
	_, exists = sm.Get(key)

	return
}

func (sm *SafeMapStrings) Get(key string) (value string, exists bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	value, exists = sm.values[key]

	return
}

func (sm *SafeMapStrings) AddKeyValue(key string, value string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.values[key] = value
}

func (sm *SafeMapStrings) Add(keyAndValue string) {
	sm.AddKeyValue(keyAndValue, keyAndValue)
}

func (sm *SafeMapStrings) Count() (count int) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return len(sm.values)
}

func (sm *SafeMapStrings) PullRand() (key string, value string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for key, value = range sm.values {
		break
	}

	delete(sm.values, key)

	return
}

/*
func (sm *SafeMapStrings) Delete(key string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.values, key)
}

func (sm *SafeMapStrings) Keys() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	keys := make([]string, 0, len(sm.values))

	for key := range sm.values {
		keys = append(keys, key)
	}

	return keys
}
*/
