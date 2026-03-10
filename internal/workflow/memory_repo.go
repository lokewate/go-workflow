package workflow

import (
	ctx "context"
	"errors"
	"sync"
	"workflow-engine/internal/workflow/context"
)

// MemoryRepo implements Repo using an in-memory map.
type MemoryRepo struct {
	mu        sync.RWMutex
	instances map[string]*WorkflowInstance
	// We store data and tokens separately to simulate persistent storage
	data   map[string]map[string]interface{}
	tokens map[string][]context.Token
}

// NewMemoryRepo initializes a new MemoryRepo.
func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		instances: make(map[string]*WorkflowInstance),
		data:      make(map[string]map[string]interface{}),
		tokens:    make(map[string][]context.Token),
	}
}

// Get retrieves a workflow instance by its ID.
func (r *MemoryRepo) Get(c ctx.Context, id string) (*WorkflowInstance, error) {
	r.mu.RLock()
	inst, ok := r.instances[id]
	r.mu.RUnlock()

	if !ok {
		return nil, errors.New("instance not found")
	}

	// Initialize the context
	inst.Context = context.NewMapContext(
		r.loadState(id),
		r.saveState(id),
	)
	if err := inst.Context.Load(id); err != nil {
		return nil, err
	}

	return inst, nil
}

// Save persists a workflow instance.
func (r *MemoryRepo) Save(c ctx.Context, inst *WorkflowInstance) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.instances[inst.ID] = inst
	return nil
}

func (r *MemoryRepo) loadState(id string) func(string) (map[string]interface{}, []context.Token, error) {
	return func(id string) (map[string]interface{}, []context.Token, error) {
		r.mu.RLock()
		defer r.mu.RUnlock()
		return r.data[id], r.tokens[id], nil
	}
}

func (r *MemoryRepo) saveState(id string) func(string, map[string]interface{}, []context.Token) error {
	return func(id string, data map[string]interface{}, tokens []context.Token) error {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.data[id] = data
		r.tokens[id] = tokens
		return nil
	}
}
