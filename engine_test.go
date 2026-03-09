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
	Workflow  workflow.Blueprint `json:"workflow"`
	Scenarios []TestScenario     `json:"scenarios"`
}

type TestScenario struct {
	Name  string `json:"name"`
	Steps []struct {
		NodeID  string                 `json:"node_id"`
		Results map[string]interface{} `json:"results"`
	} `json:"steps"`
	ExpectedTokens []string `json:"expected_tokens"`
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
				Repo:      repo,
				Blueprint: &testFile.Workflow,
			}

			// Initialize instance at the 'start' node
			instID := "test-instance"
			repo.Save(ctx, &workflow.Instance{
				ID:      instID,
				Payload: make(map[string]interface{}),
				Tokens:  []workflow.Token{{NodeID: "start", Status: workflow.TokenActive}},
			})

			// Run steps
			for _, step := range scenario.Steps {
				err := engine.CompleteTask(ctx, instID, step.NodeID, step.Results)
				assert.NoError(t, err)
			}

			// Assert final state
			finalInst, _ := repo.Get(ctx, instID)
			var activeTokenNodes []string
			for _, tok := range finalInst.Tokens {
				activeTokenNodes = append(activeTokenNodes, tok.NodeID)
			}

			assert.ElementsMatch(t, scenario.ExpectedTokens, activeTokenNodes)
		})
	}
}
