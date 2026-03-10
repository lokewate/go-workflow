package context

import (
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
		for i := 0; i < 100; i++ {
			go func(val int) {
				gctx.Set("concurrent", val)
				_ = gctx.Get("concurrent")
			}(i)
		}
	})
}
