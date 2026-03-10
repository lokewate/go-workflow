package workflow

import "context"

// Repo defines the interface for persisting workflow instances and their global contexts.
type Repo interface {
	Get(ctx context.Context, id string) (*WorkflowInstance, error)
	Save(ctx context.Context, inst *WorkflowInstance) error
	GetContext(ctx context.Context, id string) (GlobalContext, error)
	SaveContext(ctx context.Context, id string, gctx GlobalContext) error
}
