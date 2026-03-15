package workflow

import (
	"context"
	"testing"

	"github.com/lokewate/go-workflow/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDBRepo(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	repo, err := NewDBRepo(db)
	require.NoError(t, err)

	ctx := context.Background()
	inst := &WorkflowInstance{
		ID:         "inst-1",
		WorkflowID: "wf-1",
		Status:     StatusActive,
	}

	t.Run("Save and Get Instance", func(t *testing.T) {
		err := repo.Save(ctx, inst)
		assert.NoError(t, err)

		retrieved, err := repo.Get(ctx, inst.ID)
		require.NoError(t, err)
		assert.Equal(t, inst.ID, retrieved.ID)
		assert.Equal(t, inst.WorkflowID, retrieved.WorkflowID)
		assert.Equal(t, inst.Status, retrieved.Status)
	})

	t.Run("Context Persistence", func(t *testing.T) {
		wfCtx := repo.NewContext(inst.ID)
		wfCtx.Set(ctx, "key1", "val1")
		wfCtx.SetTokens(ctx, []state.Token{{ID: "t1", NodeID: "n1", Status: state.TokenActive}})

		// Reload instance and check context
		retrieved, err := repo.Get(ctx, inst.ID)
		require.NoError(t, err)
		
		assert.Equal(t, "val1", retrieved.Context.Get("key1"))
		tokens := retrieved.Context.GetTokens()
		assert.Len(t, tokens, 1)
		assert.Equal(t, "t1", tokens[0].ID)
	})

	t.Run("Instance Not Found", func(t *testing.T) {
		_, err := repo.Get(ctx, "non-existent")
		assert.ErrorIs(t, err, ErrInstanceNotFound)
	})
}
