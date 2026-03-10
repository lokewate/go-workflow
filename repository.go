package workflow

import (
	ctx "context"
)

// Repo defines the interface for persisting workflow instances.
type Repo interface {
	Get(c ctx.Context, id string) (*WorkflowInstance, error)
	Save(c ctx.Context, inst *WorkflowInstance) error
}
