package workflow

import (
	"context"
	"errors"
	"sync"
)

// MemoryRepo implements InstanceRepository using an in-memory map.
// MemoryRepo is an in-memory implementation of InstanceRepository.
// It is intended for testing and demonstration purposes.
type MemoryRepo struct {
	mu   sync.RWMutex
	data map[string]*Instance
}

// NewMemoryRepo initializes and returns a new MemoryRepo.
func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{data: make(map[string]*Instance)}
}

// Get retrieves a workflow instance by its ID.
func (r *MemoryRepo) Get(ctx context.Context, id string) (*Instance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	inst, ok := r.data[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return inst, nil
}

// Save persists a workflow instance in the in-memory map.
func (r *MemoryRepo) Save(ctx context.Context, inst *Instance) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[inst.ID] = inst
	return nil
}
