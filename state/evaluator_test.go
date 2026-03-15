package state

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvaluateCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition string
		setup     func(gctx GlobalContext)
		expected  bool
		wantErr   bool
	}{
		{
			name:      "simple true boolean",
			condition: "ok == true",
			setup: func(gctx GlobalContext) {
				gctx.Set(context.Background(), "ok", true)
			},
			expected: true,
		},
		{
			name:      "simple false boolean",
			condition: "ok == true",
			setup: func(gctx GlobalContext) {
				gctx.Set(context.Background(), "ok", false)
			},
			expected: false,
		},
		{
			name:      "numeric comparison",
			condition: "count > 5",
			setup: func(gctx GlobalContext) {
				gctx.Set(context.Background(), "count", 10)
			},
			expected: true,
		},
		{
			name:      "string comparison",
			condition: "status == 'completed'",
			setup: func(gctx GlobalContext) {
				gctx.Set(context.Background(), "status", "completed")
			},
			expected: true,
		},
		{
			name:      "missing key",
			condition: "missing == true",
			setup:     func(gctx GlobalContext) {},
			expected:  false,
			wantErr:   false, // expr evaluates nil == true as false without error
		},
		{
			name:      "invalid syntax",
			condition: "!!!invalid!!!",
			setup:     func(gctx GlobalContext) {},
			expected:  false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loadFn := func(id string) (map[string]any, []Token, error) {
				return make(map[string]any), nil, nil
			}
			saveFn := func(id string, data map[string]any, tokens []Token) error {
				return nil
			}
			gctx := NewMapContext(loadFn, saveFn)
			_ = gctx.Load(context.Background(), "test")
			tt.setup(gctx)
			result, err := EvaluateCondition(tt.condition, gctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

type trackMockContext struct {
	requestedKeys []string
	data          map[string]any
}

func (m *trackMockContext) Load(_ context.Context, _ string) error { return nil }
func (m *trackMockContext) Get(key string) any {
	m.requestedKeys = append(m.requestedKeys, key)
	return m.data[key]
}
func (m *trackMockContext) Set(_ context.Context, _ string, _ any) {}
func (m *trackMockContext) GetTokens() []Token                             { return nil }
func (m *trackMockContext) SetTokens(_ context.Context, _ []Token)         {}
func (m *trackMockContext) GetAll() map[string]any                 { return m.data }

func TestEvaluateCondition_DynamicExtraction(t *testing.T) {
	mock := &trackMockContext{
		data: map[string]any{
			"ok":     true,
			"count":  10,
			"unused": "ignored",
			"tax":    15,
		},
	}

	result, err := EvaluateCondition("ok == true && count > 5 && tax < 20", mock)
	assert.NoError(t, err)
	assert.True(t, result)

	assert.Contains(t, mock.requestedKeys, "ok")
	assert.Contains(t, mock.requestedKeys, "count")
	assert.Contains(t, mock.requestedKeys, "tax")
	assert.NotContains(t, mock.requestedKeys, "unused")
	assert.Len(t, mock.requestedKeys, 3)
}
