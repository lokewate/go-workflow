package workflow

import "context"

// InstanceRepository provides an interface for persisting and retrieving workflow instances.
type InstanceRepository interface {
	Get(ctx context.Context, id string) (*Instance, error)
	Save(ctx context.Context, inst *Instance) error
}
