package expiremap

import (
	"sync"
	"time"
)

// item struct represents an individual element with a value and its expiration time.
type item[V any] struct {
	value     V
	expiresAt time.Time
}

// ExpireMap is a generic map structure that allows setting and retrieving items with expiration.
type ExpireMap[T comparable, V any] struct {
	m          map[T][]item[V] // Holds the actual data with their expiration details.
	lock       sync.RWMutex    // Mutex for ensuring concurrent access.
	cullPeriod time.Duration   // Duration to wait before cleaning expired items.
	defaultTTL time.Duration   // Default time to live for items if not specified.
}

// New creates a new instance of ExpireMap with default cullPeriod and defaultTTL set to 1 minute.
func New[T comparable, V any]() *ExpireMap[T, V] {
	return NewEx[T, V](time.Minute, time.Minute)
}

// NewEx creates a new instance of ExpireMap with specified cullPeriod and defaultTTL.
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

// Set adds a new item to the map with the default TTL.
func (m *ExpireMap[T, V]) Set(key T, value V) {
	m.SetEx(key, value, m.defaultTTL)
}

// SetEx adds a new item to the map with a specified TTL.
func (m *ExpireMap[T, V]) SetEx(key T, value V, ttl time.Duration) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.m[key] = append(m.m[key], item[V]{value: value, expiresAt: time.Now().Add(ttl)})
}

// LoadOrStore retrieves an item from the map by key. If it doesn't exist, stores the provided value with the default TTL.

func (m *ExpireMap[T, V]) LoadOrStore(key T, value V) (*V, bool) {
	return m.LoadOrStoreEx(key, value, m.defaultTTL)
}

// LoadOrStoreEx retrieves an item from the map by key. If it doesn't exist, stores the provided value with the specified TTL.
func (m *ExpireMap[T, V]) LoadOrStoreEx(key T, value V, ttl time.Duration) (*V, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	v, ok := m.getNewestValidItem(key)
	if ok {
		return v, ok
	}
	m.m[key] = append(m.m[key], item[V]{value: value, expiresAt: time.Now().Add(ttl)})
	return &value, false
}

// Get retrieves the newest valid item from the map by key.
func (m *ExpireMap[T, V]) Get(key T) (*V, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.getNewestValidItem(key)
}

// LoadAndDelete retrieves the newest valid item from the map by key and then deletes it.
func (m *ExpireMap[T, V]) LoadAndDelete(key T) (*V, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	v, ok := m.getNewestValidItem(key)
	if ok {
		m.deleteNewestValidItem(key)
	}
	return v, ok
}

// Load retrieves the newest valid item from the map by key.
func (m *ExpireMap[T, V]) Load(key T) (*V, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.getNewestValidItem(key)
}

// Delete removes all items associated with the provided key from the map.
func (m *ExpireMap[T, V]) Delete(key T) {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.m, key)
}

// cull periodically cleans up expired items from the map.
func (m *ExpireMap[T, V]) cull() {
	for {
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

// getNewestValidItem retrieves the newest valid item for a given key.
func (m *ExpireMap[T, V]) getNewestValidItem(key T) (*V, bool) {
	var newest item[V]
	found := false

	if items, ok := m.m[key]; ok {
		for _, currentItem := range items {
			if currentItem.expiresAt.After(time.Now()) && (!found || currentItem.expiresAt.After(newest.expiresAt)) {
				newest = currentItem
				found = true
			}
		}
	}

	if found {
		return &newest.value, true
	}
	return nil, false
}

// deleteNewestValidItem deletes the newest valid item for a given key.
func (m *ExpireMap[T, V]) deleteNewestValidItem(key T) {
	items := m.m[key]
	newestIndex := -1
	var newestExpiration time.Time

	for i, currentItem := range items {
		if currentItem.expiresAt.After(time.Now()) && (newestIndex == -1 || currentItem.expiresAt.After(newestExpiration)) {
			newestIndex = i
			newestExpiration = currentItem.expiresAt
		}
	}

	if newestIndex != -1 {
		m.m[key] = append(items[:newestIndex], items[newestIndex+1:]...)
	}
}

// Range iterates over each key-value pair in the map and calls the provided function until it returns false.
func (m *ExpireMap[T, V]) Range(f func(key T, value V) bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	for k, v := range m.m {
		for _, i := range v {
			if !f(k, i.value) {
				return
			}
		}
	}
}
