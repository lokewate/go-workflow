package workflow

import "context"

type InstanceRepository interface {
	Get(ctx context.Context, id string) (*Instance, error)
	Save(ctx context.Context, inst *Instance) error
}
