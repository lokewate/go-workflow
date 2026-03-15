package workflow

import (
	"context"
	"sync"

	"github.com/lokewate/go-workflow/state"
)

// MemoryRepo implements Repo using an in-memory map.
type MemoryRepo struct {
	mu sync.RWMutex
	// instances maps instance IDs to their workflow instance objects.
	instances map[string]*WorkflowInstance
	// data stores workflow-wide variables, indexed by instance ID.
	data map[string]map[string]any
	// tokens tracks current execution markers for each instance.
	tokens map[string][]state.Token
}

// NewMemoryRepo initializes a new MemoryRepo.
func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		instances: make(map[string]*WorkflowInstance),
		data:      make(map[string]map[string]any),
		tokens:    make(map[string][]state.Token),
	}
}

// NewContext creates a GlobalContext wired to this repo's persistence layer.
func (r *MemoryRepo) NewContext(instID string) state.GlobalContext {
	return state.NewMapContextWithID(
		instID,
		r.loadState(instID),
		r.saveState(instID),
	)
}

// Get retrieves a workflow instance by its ID.
func (r *MemoryRepo) Get(ctx context.Context, id string) (*WorkflowInstance, error) {
	r.mu.RLock()
	inst, ok := r.instances[id]
	r.mu.RUnlock()

	if !ok {
		return nil, ErrInstanceNotFound
	}

	// Wire context to the repo's persistence
	inst.Context = state.NewMapContext(
		r.loadState(id),
		r.saveState(id),
	)
	if err := inst.Context.Load(ctx, id); err != nil {
		return nil, err
	}

	return inst, nil
}

// Save persists a workflow instance.
func (r *MemoryRepo) Save(_ context.Context, inst *WorkflowInstance) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.instances[inst.ID] = inst
	return nil
}

func (r *MemoryRepo) loadState(_ string) func(string) (map[string]any, []state.Token, error) {
	return func(id string) (map[string]any, []state.Token, error) {
		r.mu.RLock()
		defer r.mu.RUnlock()
		return r.data[id], r.tokens[id], nil
	}
}

func (r *MemoryRepo) saveState(_ string) func(string, map[string]any, []state.Token) error {
	return func(id string, data map[string]any, tokens []state.Token) error {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.data[id] = data
		r.tokens[id] = tokens
		return nil
	}
}
