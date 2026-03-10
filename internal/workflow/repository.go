package workflow

import (
	ctx "context"
	"workflow-engine/internal/workflow/context"
)

// Repo defines the interface for persisting workflow instances and their global contexts.
type Repo interface {
	Get(c ctx.Context, id string) (*WorkflowInstance, error)
	Save(c ctx.Context, inst *WorkflowInstance) error
	GetContext(c ctx.Context, id string) (context.GlobalContext, error)
	SaveContext(c ctx.Context, id string, gctx context.GlobalContext) error
}
