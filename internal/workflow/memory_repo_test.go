package workflow

import (
	ctx "context"
	"testing"
	"workflow-engine/internal/workflow/context"

	"github.com/stretchr/testify/assert"
)

func TestMemoryRepo(t *testing.T) {
	c := ctx.Background()
	repo := NewMemoryRepo()

	t.Run("Save and Get Instance", func(t *testing.T) {
		inst := &WorkflowInstance{ID: "inst-1"}
		err := repo.Save(c, inst)
		assert.NoError(t, err)

		retrieved, err := repo.Get(c, "inst-1")
		assert.NoError(t, err)
		assert.Equal(t, inst.ID, retrieved.ID)
		assert.NotNil(t, retrieved.Context)
	})

	t.Run("Context Persistence", func(t *testing.T) {
		inst, _ := repo.Get(c, "inst-1")
		inst.Context.Set("key", "value")
		inst.Context.SetTokens([]context.Token{{ID: "t1", NodeID: "n1", Status: context.TokenActive}})

		// Reload instance
		retrieved, _ := repo.Get(c, "inst-1")
		assert.Equal(t, "value", retrieved.Context.Get("key"))
		assert.Len(t, retrieved.Context.GetTokens(), 1)
	})

	t.Run("Get Non-existent Instance", func(t *testing.T) {
		retrieved, err := repo.Get(c, "missing")
		assert.Error(t, err)
		assert.Nil(t, retrieved)
	})
}
