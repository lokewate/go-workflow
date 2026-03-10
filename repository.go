package workflow

import (
	"context"
)

// Repo defines the interface for persisting workflow instances.
type Repo interface {
	Get(c context.Context, id string) (*WorkflowInstance, error)
	Save(c context.Context, inst *WorkflowInstance) error
}
