package workflow

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkflowInstance_JSON(t *testing.T) {
	inst := &WorkflowInstance{
		ID: "test-id",
	}

	data, err := json.Marshal(inst)
	assert.NoError(t, err)

	var inst2 WorkflowInstance
	err = json.Unmarshal(data, &inst2)
	assert.NoError(t, err)

	assert.Equal(t, inst.ID, inst2.ID)
}
