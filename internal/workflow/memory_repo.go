package workflow

import (
	ctx "context"
	"errors"
	"sync"
	"workflow-engine/internal/workflow/context"
)

// MemoryRepo implements Repo using an in-memory map.
// MemoryRepo is an in-memory implementation of Repo.
type MemoryRepo struct {
	mu        sync.RWMutex
	instances map[string]*WorkflowInstance
	contexts  map[string]context.GlobalContext
}

// NewMemoryRepo initializes a new MemoryRepo.
func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		instances: make(map[string]*WorkflowInstance),
		contexts:  make(map[string]context.GlobalContext),
	}
}

// Get retrieves a workflow instance by its ID.
func (r *MemoryRepo) Get(c ctx.Context, id string) (*WorkflowInstance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	inst, ok := r.instances[id]
	if !ok {
		return nil, errors.New("instance not found")
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

// GetContext retrieves a global context by its ID.
func (r *MemoryRepo) GetContext(c ctx.Context, id string) (context.GlobalContext, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	gctx, ok := r.contexts[id]
	if !ok {
		return nil, errors.New("context not found")
	}
	return gctx, nil
}

// SaveContext persists a global context.
func (r *MemoryRepo) SaveContext(c ctx.Context, id string, gctx context.GlobalContext) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.contexts[id] = gctx
	return nil
}
