package state

import (
	"context"
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
	// ID is a unique identifier for the token.
	ID string `json:"id"`
	// NodeID is the ID of the node where the token is currently located.
	NodeID string `json:"node_id"`
	// Status indicates if the token is active or waiting.
	Status TokenStatus `json:"status"`
}

// GlobalContext provides an interface for accessing and modifying workflow-wide state.
// It allows for different backends (e.g., in-memory map, database) to store the state.
type GlobalContext interface {
	// Load retrieves the context for a given ID.
	Load(ctx context.Context, id string) error
	// Get retrieves the value for a given key from the context.
	Get(key string) interface{}
	// Set stores a value for a given key in the context.
	Set(ctx context.Context, key string, val interface{})
	// GetTokens returns the current tokens in the context.
	GetTokens() []Token
	// SetTokens updates the tokens in the context.
	SetTokens(ctx context.Context, tokens []Token)
}

// MapContext is an in-memory implementation of GlobalContext using a map.
type MapContext struct {
	mu sync.RWMutex
	id string
	// data holds the workflow-wide variables and their values.
	data map[string]interface{}
	// tokens tracks the current execution markers in the workflow.
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

// NewMapContextWithID initializes a new MapContext pre-loaded with an instance ID and ready for use.
// This is used when creating a new instance that hasn't been persisted yet.
func NewMapContextWithID(id string, loadFn func(string) (map[string]interface{}, []Token, error), saveFn func(string, map[string]interface{}, []Token) error) *MapContext {
	return &MapContext{
		id:     id,
		data:   make(map[string]interface{}),
		loadFn: loadFn,
		saveFn: saveFn,
	}
}

// Load retrieves the context for a given ID.
func (m *MapContext) Load(_ context.Context, id string) error {
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
func (m *MapContext) Set(_ context.Context, key string, val interface{}) {
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
func (m *MapContext) SetTokens(_ context.Context, tokens []Token) {
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
