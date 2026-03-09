package workflow

import (
	"context"
	"errors"
	"sync"
)

type MemoryRepo struct {
	mu   sync.RWMutex
	data map[string]*Instance
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{data: make(map[string]*Instance)}
}

func (r *MemoryRepo) Get(ctx context.Context, id string) (*Instance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	inst, ok := r.data[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return inst, nil
}

func (r *MemoryRepo) Save(ctx context.Context, inst *Instance) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[inst.ID] = inst
	return nil
}
