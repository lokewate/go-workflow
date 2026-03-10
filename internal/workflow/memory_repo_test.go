package workflow

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemoryRepo(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepo()

	t.Run("Save and Get Instance", func(t *testing.T) {
		inst := &WorkflowInstance{ID: "inst-1", GlobalContextID: "ctx-1"}
		err := repo.Save(ctx, inst)
		assert.NoError(t, err)

		retrieved, err := repo.Get(ctx, "inst-1")
		assert.NoError(t, err)
		assert.Equal(t, inst, retrieved)
	})

	t.Run("Get Non-existent Instance", func(t *testing.T) {
		retrieved, err := repo.Get(ctx, "missing")
		assert.Error(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("Save and Get Context", func(t *testing.T) {
		gctx := NewMapContext()
		gctx.Set("key", "value")
		err := repo.SaveContext(ctx, "ctx-1", gctx)
		assert.NoError(t, err)

		retrieved, err := repo.GetContext(ctx, "ctx-1")
		assert.NoError(t, err)
		assert.Equal(t, gctx, retrieved)
		assert.Equal(t, "value", retrieved.Get("key"))
	})

	t.Run("Get Non-existent Context", func(t *testing.T) {
		retrieved, err := repo.GetContext(ctx, "missing")
		assert.Error(t, err)
		assert.Nil(t, retrieved)
	})
}
