package expiremap

import (
	"sync"
	"time"
)

// item is a value with an expiration time.
type item[V any] struct {
	value     V
	expiresAt time.Time
}

// ExpireMap is a thread-safe map, that automatically deletes entries after a given time.
type ExpireMap[T comparable, V any] struct {
	m          map[T][]item[V]
	lock       sync.RWMutex
	cullPeriod time.Duration
	defaultTTL time.Duration
}

// New creates a new ExpireMap with a default cull period of 1 minute and a default TTL of 1 minute.
func New[T comparable, V any]() *ExpireMap[T, V] {
	return NewEx[T, V](time.Minute, time.Minute)
}

// NewEx creates a new ExpireMap with the given cull period and default TTL.
func NewEx[T comparable, V any](cullPeriod, defaultTTL time.Duration) *ExpireMap[T, V] {
	var m = ExpireMap[T, V]{
		m:          make(map[T][]item[V]),
		cullPeriod: cullPeriod,
		defaultTTL: defaultTTL,
		lock:       sync.RWMutex{},
	}
	go m.cull()
	return &m
}

// Set sets a value with the default TTL.
func (m *ExpireMap[T, V]) Set(key T, value V) {
	m.SetEx(key, value, m.defaultTTL)
}

// SetEx sets a value with the given TTL.
func (m *ExpireMap[T, V]) SetEx(key T, value V, ttl time.Duration) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if _, ok := m.m[key]; !ok {
		m.m[key] = []item[V]{}
	}
	m.m[key] = append(m.m[key], item[V]{value: value, expiresAt: time.Now().Add(ttl)})
}

// Get returns the newest value for the given key, if it exists and is not expired.
func (m *ExpireMap[T, V]) Get(key T) (interface{}, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	var newest item[V]
	found := false

	if items, ok := m.m[key]; ok {
		for _, i := range items {
			if i.expiresAt.After(time.Now()) && (!found || i.expiresAt.After(newest.expiresAt)) {
				newest = i
				found = true
			}
		}
	}

	if found {
		return newest.value, true
	}
	return nil, false
}

// LoadAndDelete returns the newest value for the given key, if it exists and is not expired, and deletes it.
func (m *ExpireMap[T, V]) LoadAndDelete(key T) (interface{}, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if _, ok := m.m[key]; !ok {
		return nil, false
	}
	for i, v := range m.m[key] {
		if v.expiresAt.After(time.Now()) {
			m.m[key] = append(m.m[key][:i], m.m[key][i+1:]...)
			return v.value, true
		}
	}
	return nil, false
}

// Load returns the newest value for the given key, if it exists and is not expired.
func (m *ExpireMap[T, V]) Load(key T) (interface{}, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if _, ok := m.m[key]; !ok {
		return nil, false
	}
	for _, v := range m.m[key] {
		if v.expiresAt.After(time.Now()) {
			return v.value, true
		}
	}
	return nil, false
}

// Delete deletes the given key.
func (m *ExpireMap[T, V]) Delete(key T) {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.m, key)
}

// cull periodically removes expired items.
func (m *ExpireMap[T, V]) cull() {
	for {
		// Periodically cull expired items
		time.Sleep(m.cullPeriod)
		now := time.Now()
		m.lock.Lock()
		for k, v := range m.m {
			valid := 0
			for _, i := range v {
				if i.expiresAt.After(now) {
					v[valid] = i
					valid++
				}
			}
			if valid == 0 {
				delete(m.m, k)
			} else {
				m.m[k] = v[:valid]
			}
		}
		m.lock.Unlock()
	}
}
