package workflow

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapContext(t *testing.T) {
	gctx := NewMapContext()

	t.Run("Set and Get", func(t *testing.T) {
		gctx.Set("foo", "bar")
		assert.Equal(t, "bar", gctx.Get("foo"))
	})

	t.Run("AsMap returns copy", func(t *testing.T) {
		gctx.Set("a", 1)
		m := gctx.AsMap()
		assert.Equal(t, 1, m["a"])

		// Mutate map copy
		m["a"] = 2
		assert.Equal(t, 1, gctx.Get("a"), "Original should not be mutated")
	})

	t.Run("Concurrent access", func(t *testing.T) {
		// This is mostly to ensure no race condition is caught by -race
		for i := 0; i < 100; i++ {
			go func(val int) {
				gctx.Set("concurrent", val)
				_ = gctx.Get("concurrent")
			}(i)
		}
	})
}

func TestWorkflowInstance_JSON(t *testing.T) {
	inst := &WorkflowInstance{
		ID:              "test-id",
		GlobalContextID: "ctx-id",
		Tokens: []Token{
			{ID: "t1", NodeID: "n1", Status: TokenActive},
		},
	}

	data, err := json.Marshal(inst)
	assert.NoError(t, err)

	var inst2 WorkflowInstance
	err = json.Unmarshal(data, &inst2)
	assert.NoError(t, err)

	assert.Equal(t, inst.ID, inst2.ID)
	assert.Equal(t, inst.GlobalContextID, inst2.GlobalContextID)
	assert.Equal(t, inst.Tokens, inst2.Tokens)
}
