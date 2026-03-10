package context

import (
	"sync"
)

// TokenStatus defines the current state of a workflow token.
type TokenStatus string

const (
	// TokenActive indicates the token is currently at a task being performed.
	TokenActive TokenStatus = "ACTIVE"
	// TokenWaiting indicates the token is waiting at a join gateway.
	TokenWaiting TokenStatus = "WAITING"
)

// Token tracks the execution progress within a workflow instance.
type Token struct {
	ID     string      `json:"id"`
	NodeID string      `json:"node_id"`
	Status TokenStatus `json:"status"`
}

// GlobalContext provides an interface for accessing and modifying workflow-wide state.
// It allows for different backends (e.g., in-memory map, database) to store the state.
type GlobalContext interface {
	// Load retrieves the context for a given ID.
	Load(id string) error
	// Get retrieves the value for a given key from the context.
	Get(key string) interface{}
	// Set stores a value for a given key in the context and triggers an implicit save.
	// TODO: we might want to avoid implicit saving for perf reasons, and instead have a Save() method.
	// otherwise, for example, each Set call will trigger a DB write for a DB based GlobalContext.
	Set(key string, val interface{})
	// GetTokens returns the current tokens in the context.
	GetTokens() []Token
	// SetTokens updates the tokens in the context and triggers an implicit save.
	SetTokens(tokens []Token)
	// Data returns all dynamic state as a map (e.g. for evaluation).
	Data() map[string]interface{}
}

// MapContext is an in-memory implementation of GlobalContext using a map.
type MapContext struct {
	mu     sync.RWMutex
	id     string
	data   map[string]interface{}
	tokens []Token
	// saveFn handles persistence. It's called internally on every mutation.
	saveFn func(id string, data map[string]interface{}, tokens []Token) error
	// loadFn retrieves data for a given ID.
	loadFn func(id string) (map[string]interface{}, []Token, error)
}

// NewMapContext initializes a new MapContext with load and save functions.
func NewMapContext(loadFn func(string) (map[string]interface{}, []Token, error), saveFn func(string, map[string]interface{}, []Token) error) *MapContext {
	return &MapContext{
		loadFn: loadFn,
		saveFn: saveFn,
	}
}

// Load retrieves the context for a given ID.
func (m *MapContext) Load(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.id = id
	if m.loadFn != nil {
		data, tokens, err := m.loadFn(id)
		if err != nil {
			return err
		}
		m.data = data
		if m.data == nil {
			m.data = make(map[string]interface{})
		}
		m.tokens = tokens
	}
	return nil
}

// Get retrieves the value for a given key from the context.
func (m *MapContext) Get(key string) interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.data[key]
}

// Set stores a value for a given key in the context and triggers an implicit save.
func (m *MapContext) Set(key string, val interface{}) {
	m.mu.Lock()
	if m.data == nil {
		m.data = make(map[string]interface{})
	}
	m.data[key] = val
	m.mu.Unlock()
	m.triggerSave()
}

// GetTokens returns the current tokens in the context.
func (m *MapContext) GetTokens() []Token {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]Token, len(m.tokens))
	copy(res, m.tokens)
	return res
}

// SetTokens updates the tokens in the context and triggers an implicit save.
func (m *MapContext) SetTokens(tokens []Token) {
	m.mu.Lock()
	m.tokens = make([]Token, len(tokens))
	copy(m.tokens, tokens)
	m.mu.Unlock()
	m.triggerSave()
}

func (m *MapContext) triggerSave() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.saveFn != nil {
		m.saveFn(m.id, m.data, m.tokens)
	}
}

// Data returns all data from the context as a map.
func (m *MapContext) Data() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make(map[string]interface{})
	for k, v := range m.data {
		res[k] = v
	}
	return res
}
