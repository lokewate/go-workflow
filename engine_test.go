package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"workflow-engine/internal/workflow"

	"github.com/stretchr/testify/assert"
)

type TestFile struct {
	Workflow  workflow.Workflow `json:"workflow"`
	Scenarios []TestScenario    `json:"scenarios"`
}

type TestScenario struct {
	Name            string                 `json:"name"`
	Steps           []TestStep             `json:"steps"`
	ExpectedTokens  []string               `json:"expected_tokens"`
	ExpectedPayload map[string]interface{} `json:"expected_payload"`
}

type TestStep struct {
	NodeID  string                 `json:"node_id"`
	Results map[string]interface{} `json:"results"`
}

func TestWorkflows(t *testing.T) {
	content, err := os.ReadFile("test_suite.json")
	assert.NoError(t, err)

	var testFile TestFile
	err = json.Unmarshal(content, &testFile)
	assert.NoError(t, err)

	for _, scenario := range testFile.Scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			ctx := context.Background()
			repo := workflow.NewMemoryRepo()
			engine := &workflow.Engine{
				Repo:     repo,
				Workflow: &testFile.Workflow,
			}

			// All scenarios start with a token on the 'start' node
			instID := "test-instance"
			repo.Save(ctx, &workflow.Instance{
				ID:      instID,
				Payload: make(map[string]interface{}),
				Tokens:  []workflow.Token{{ID: "init-token", NodeID: "start", Status: workflow.TokenActive}},
			})

			// Execute mock task completions defined in JSON
			for _, step := range scenario.Steps {
				err := engine.CompleteTask(ctx, instID, step.NodeID, step.Results)
				assert.NoError(t, err, "Failed completing step: %s", step.NodeID)
			}

			// Verify final state
			finalInst, _ := repo.Get(ctx, instID)

			// 1. Check Tokens
			var actualTokenNodes []string
			for _, tok := range finalInst.Tokens {
				actualTokenNodes = append(actualTokenNodes, tok.NodeID)
			}
			assert.ElementsMatch(t, scenario.ExpectedTokens, actualTokenNodes, "Tokens do not match expected end state")

			// 2. Check Payload mutations
			for key, expectedValue := range scenario.ExpectedPayload {
				assert.Equal(t, expectedValue, finalInst.Payload[key], "Payload mismatch for key: %s", key)
			}
		})
	}
}
