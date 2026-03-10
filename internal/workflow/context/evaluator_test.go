package context

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
			condition: "ok == true",
			setup: func(gctx GlobalContext) {
				gctx.Set("ok", true)
			},
			expected: true,
		},
		{
			name:      "simple false boolean",
			condition: "ok == true",
			setup: func(gctx GlobalContext) {
				gctx.Set("ok", false)
			},
			expected: false,
		},
		{
			name:      "numeric comparison",
			condition: "count > 5",
			setup: func(gctx GlobalContext) {
				gctx.Set("count", 10)
			},
			expected: true,
		},
		{
			name:      "string comparison",
			condition: "status == 'completed'",
			setup: func(gctx GlobalContext) {
				gctx.Set("status", "completed")
			},
			expected: true,
		},
		{
			name:      "missing key",
			condition: "missing == true",
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
			loadFn := func(id string) (map[string]interface{}, []Token, error) {
				return make(map[string]interface{}), nil, nil
			}
			saveFn := func(id string, data map[string]interface{}, tokens []Token) error {
				return nil
			}
			gctx := NewMapContext(loadFn, saveFn)
			_ = gctx.Load("test")
			tt.setup(gctx)
			result := EvaluateCondition(tt.condition, gctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

type trackMockContext struct {
	requestedKeys []string
	data          map[string]interface{}
}

func (m *trackMockContext) Load(id string) error { return nil }
func (m *trackMockContext) Get(key string) interface{} {
	m.requestedKeys = append(m.requestedKeys, key)
	return m.data[key]
}
func (m *trackMockContext) Set(key string, val interface{}) {}
func (m *trackMockContext) GetTokens() []Token              { return nil }
func (m *trackMockContext) SetTokens(tokens []Token)        {}

func TestEvaluateCondition_DynamicExtraction(t *testing.T) {
	mock := &trackMockContext{
		data: map[string]interface{}{
			"ok":     true,
			"count":  10,
			"unused": "ignored",
			"tax":    15,
		},
	}

	result := EvaluateCondition("ok == true && count > 5 && tax < 20", mock)
	assert.True(t, result)

	assert.Contains(t, mock.requestedKeys, "ok")
	assert.Contains(t, mock.requestedKeys, "count")
	assert.Contains(t, mock.requestedKeys, "tax")
	assert.NotContains(t, mock.requestedKeys, "unused")
	assert.Len(t, mock.requestedKeys, 3)
}
