package state

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapContext(t *testing.T) {
	var savedData map[string]any
	var savedTokens []Token

	loadFn := func(id string) (map[string]any, []Token, error) {
		return savedData, savedTokens, nil
	}
	saveFn := func(id string, data map[string]any, tokens []Token) error {
		savedData = data
		savedTokens = tokens
		return nil
	}

	ctx := context.Background()
	gctx := NewMapContext(loadFn, saveFn)
	err := gctx.Load(ctx, "test-id")
	assert.NoError(t, err)

	t.Run("Set and Get", func(t *testing.T) {
		gctx.Set(ctx, "foo", "bar")
		assert.Equal(t, "bar", gctx.Get("foo"))
		assert.Equal(t, "bar", savedData["foo"], "Should trigger implicit save")
	})

	t.Run("Tokens", func(t *testing.T) {
		tokens := []Token{{ID: "t1", NodeID: "n1", Status: TokenActive}}
		gctx.SetTokens(ctx, tokens)
		assert.Equal(t, tokens, gctx.GetTokens())
		assert.Equal(t, tokens, savedTokens, "Should trigger implicit save")
	})

	t.Run("Concurrent access", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			go func(val int) {
				gctx.Set(ctx, "concurrent", val)
				_ = gctx.Get("concurrent")
			}(i)
		}
	})
}
