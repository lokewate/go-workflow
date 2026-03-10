package context

import (
	"sync"
)

// GlobalContext provides an interface for accessing and modifying workflow-wide state.
// It allows for different backends (e.g., in-memory map, database) to store the state.
type GlobalContext interface {
	// Get retrieves the value for a given key from the context.
	Get(key string) interface{}
	// Set stores a value for a given key in the context.
	Set(key string, val interface{})
	// AsMap returns a point-in-time copy of the internal data as a map.
	AsMap() map[string]interface{}
}

// MapContext is an in-memory implementation of GlobalContext using a map.
type MapContext struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

// NewMapContext initializes a new MapContext.
func NewMapContext() *MapContext {
	return &MapContext{data: make(map[string]interface{})}
}

// Get retrieves the value for a given key from the context.
func (m *MapContext) Get(key string) interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.data[key]
}

// Set stores a value for a given key in the context.
func (m *MapContext) Set(key string, val interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = make(map[string]interface{})
	}
	m.data[key] = val
}

// AsMap returns a point-in-time copy of the internal data as a map.
func (m *MapContext) AsMap() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Return a copy to prevent external mutation
	res := make(map[string]interface{})
	for k, v := range m.data {
		res[k] = v
	}
	return res
}
