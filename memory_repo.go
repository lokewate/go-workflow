package workflow

import (
	"context"
	"errors"
	"sync"
	"workflow-engine/state"
)

// MemoryRepo implements Repo using an in-memory map.
type MemoryRepo struct {
	mu sync.RWMutex
	// instances maps instance IDs to their workflow instance objects.
	instances map[string]*WorkflowInstance
	// data stores workflow-wide variables, indexed by instance ID.
	data map[string]map[string]interface{}
	// tokens tracks current execution markers for each instance.
	tokens map[string][]state.Token
}

// NewMemoryRepo initializes a new MemoryRepo.
func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		instances: make(map[string]*WorkflowInstance),
		data:      make(map[string]map[string]interface{}),
		tokens:    make(map[string][]state.Token),
	}
}

// Get retrieves a workflow instance by its ID.
func (r *MemoryRepo) Get(c context.Context, id string) (*WorkflowInstance, error) {
	r.mu.RLock()
	inst, ok := r.instances[id]
	r.mu.RUnlock()

	if !ok {
		return nil, errors.New("instance not found")
	}

	// Initialize the context
	inst.Context = state.NewMapContext(
		r.loadState(id),
		r.saveState(id),
	)
	if err := inst.Context.Load(id); err != nil {
		return nil, err
	}

	return inst, nil
}

// Save persists a workflow instance.
func (r *MemoryRepo) Save(c context.Context, inst *WorkflowInstance) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.instances[inst.ID] = inst
	return nil
}

func (r *MemoryRepo) loadState(id string) func(string) (map[string]interface{}, []state.Token, error) {
	return func(id string) (map[string]interface{}, []state.Token, error) {
		r.mu.RLock()
		defer r.mu.RUnlock()
		return r.data[id], r.tokens[id], nil
	}
}

func (r *MemoryRepo) saveState(id string) func(string, map[string]interface{}, []state.Token) error {
	return func(id string, data map[string]interface{}, tokens []state.Token) error {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.data[id] = data
		r.tokens[id] = tokens
		return nil
	}
}
