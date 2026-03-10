package workflow

import (
	"testing"
	"workflow-engine/internal/workflow/context"

	"github.com/stretchr/testify/assert"
)

func TestEvaluateCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition string
		setup     func(gctx context.GlobalContext)
		expected  bool
	}{
		{
			name:      "simple true boolean",
			condition: "payload.ok == true",
			setup: func(gctx context.GlobalContext) {
				gctx.Set("ok", true)
			},
			expected: true,
		},
		{
			name:      "simple false boolean",
			condition: "payload.ok == true",
			setup: func(gctx context.GlobalContext) {
				gctx.Set("ok", false)
			},
			expected: false,
		},
		{
			name:      "numeric comparison",
			condition: "payload.count > 5",
			setup: func(gctx context.GlobalContext) {
				gctx.Set("count", 10)
			},
			expected: true,
		},
		{
			name:      "string comparison",
			condition: "payload.status == 'completed'",
			setup: func(gctx context.GlobalContext) {
				gctx.Set("status", "completed")
			},
			expected: true,
		},
		{
			name:      "missing key",
			condition: "payload.missing == true",
			setup:     func(gctx context.GlobalContext) {},
			expected:  false,
		},
		{
			name:      "invalid syntax",
			condition: "!!!invalid!!!",
			setup:     func(gctx context.GlobalContext) {},
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loadFn := func(id string) (map[string]interface{}, []context.Token, error) {
				return make(map[string]interface{}), nil, nil
			}
			saveFn := func(id string, data map[string]interface{}, tokens []context.Token) error {
				return nil
			}
			gctx := context.NewMapContext(loadFn, saveFn)
			_ = gctx.Load("test")
			tt.setup(gctx)
			result := context.EvaluateCondition(tt.condition, gctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}
