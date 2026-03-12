package workflow

import (
	"context"

	"github.com/lokewate/go-workflow/state"
)

// Repo defines the interface for persisting workflow instances.
type Repo interface {
	// Get retrieves a workflow instance by ID.
	Get(ctx context.Context, id string) (*WorkflowInstance, error)
	// Save persists a workflow instance.
	Save(ctx context.Context, inst *WorkflowInstance) error
	// NewContext creates a GlobalContext wired to this repo's persistence layer for the given instance ID.
	NewContext(instID string) state.GlobalContext
}
