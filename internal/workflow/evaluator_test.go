package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvaluateCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition string
		setup     func(gctx GlobalContext)
		expected  bool
	}{
		{
			name:      "simple true boolean",
			condition: "payload.ok == true",
			setup: func(gctx GlobalContext) {
				gctx.Set("ok", true)
			},
			expected: true,
		},
		{
			name:      "simple false boolean",
			condition: "payload.ok == true",
			setup: func(gctx GlobalContext) {
				gctx.Set("ok", false)
			},
			expected: false,
		},
		{
			name:      "numeric comparison",
			condition: "payload.count > 5",
			setup: func(gctx GlobalContext) {
				gctx.Set("count", 10)
			},
			expected: true,
		},
		{
			name:      "string comparison",
			condition: "payload.status == 'completed'",
			setup: func(gctx GlobalContext) {
				gctx.Set("status", "completed")
			},
			expected: true,
		},
		{
			name:      "missing key",
			condition: "payload.missing == true",
			setup:     func(gctx GlobalContext) {},
			expected:  false,
		},
		{
			name:      "invalid syntax",
			condition: "!!!invalid!!!",
			setup:     func(gctx GlobalContext) {},
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gctx := NewMapContext()
			tt.setup(gctx)
			result := EvaluateCondition(tt.condition, gctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}
