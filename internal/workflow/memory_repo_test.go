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
		inst := &WorkflowInstance{ID: "inst-1", GlobalContextID: "ctx-1"}
		err := repo.Save(c, inst)
		assert.NoError(t, err)

		retrieved, err := repo.Get(c, "inst-1")
		assert.NoError(t, err)
		assert.Equal(t, inst, retrieved)
	})

	t.Run("Get Non-existent Instance", func(t *testing.T) {
		retrieved, err := repo.Get(c, "missing")
		assert.Error(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("Save and Get Context", func(t *testing.T) {
		gctx := context.NewMapContext()
		gctx.Set("key", "value")
		err := repo.SaveContext(c, "ctx-1", gctx)
		assert.NoError(t, err)

		retrieved, err := repo.GetContext(c, "ctx-1")
		assert.NoError(t, err)
		assert.Equal(t, gctx, retrieved)
		assert.Equal(t, "value", retrieved.Get("key"))
	})

	t.Run("Get Non-existent Context", func(t *testing.T) {
		retrieved, err := repo.GetContext(c, "missing")
		assert.Error(t, err)
		assert.Nil(t, retrieved)
	})

}
