package workflow

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestFile struct {
	Workflow  Workflow       `json:"workflow"`
	Scenarios []TestScenario `json:"scenarios"`
}

type TestScenario struct {
	Name           string         `json:"name"`
	Steps          []TestStep     `json:"steps"`
	ExpectedTokens []string       `json:"expected_tokens"`
	ExpectedStatus WorkflowStatus `json:"expected_status"`
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
			repo := NewMemoryRepo()
			engine := &Engine{
				Repo:     repo,
				Workflow: &testFile.Workflow,
			}

			// Initialize instance
			instID := "test-instance"
			inst := &WorkflowInstance{
				ID: instID,
			}
			repo.Save(ctx, inst)
			inst, _ = repo.Get(ctx, instID)

			// Start workflow from the START node
			err = engine.StartWorkflow(ctx, inst, "start")
			assert.NoError(t, err)

			// Execute mock task completions defined in JSON
			// (Any START nodes in steps should be skipped as they no longer accept CompleteTask)
			for _, step := range scenario.Steps {
				if step.NodeID == "start" {
					// We can put results into the context manually, simulating payload args
					for k, v := range step.Results {
						inst.Context.Set(k, v)
					}
					// Auto-save these results
					repo.Save(ctx, inst)
					continue
				}
				err := engine.CompleteTask(ctx, instID, step.NodeID, step.Results)
				assert.NoError(t, err, "Failed completing step: %s", step.NodeID)
			}

			// Verify final state
			finalInst, _ := repo.Get(ctx, instID)

			// 1. Check expected status
			if scenario.ExpectedStatus != "" {
				assert.Equal(t, scenario.ExpectedStatus, finalInst.Status, "Workflow status does not match")
			} else {
				// Fallback to expecting completed if "end" is in tokens
				expectComplete := false
				for _, expTok := range scenario.ExpectedTokens {
					if expTok == "end" {
						expectComplete = true
						break
					}
				}
				if expectComplete {
					assert.Equal(t, "COMPLETED", finalInst.Status, "Workflow should be COMPLETED")
				}
			}

			if finalInst.Status == "COMPLETED" {
				// 2. Check Tokens (should be empty for COMPLETED)
				assert.Empty(t, finalInst.Context.GetTokens(), "Tokens should be empty after completion")
			} else {
				// 1. Check Tokens for partial completion
				var actualTokenNodes []string
				for _, tok := range finalInst.Context.GetTokens() {
					actualTokenNodes = append(actualTokenNodes, tok.NodeID)
				}
				assert.ElementsMatch(t, scenario.ExpectedTokens, actualTokenNodes, "Tokens do not match expected end state")
			}
		})
	}
}
