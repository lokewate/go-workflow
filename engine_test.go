package workflow

import (
	"context"
	"encoding/json"
	"os"
	"sync"
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
			mgr := NewWorkflowManager(repo)

			// Cast to manager implementation to use AddWorkflow or use exported if I change it
			if mi, ok := mgr.(*manager); ok {
				mi.AddWorkflow(&testFile.Workflow)
			}

			// Capture task activations
			var mu sync.Mutex
			activations := make(map[string][]string) // NodeID -> []ExecutionID

			mgr.RegisterTaskHandler(func(ctx context.Context, payload TaskPayload) error {
				mu.Lock()
				defer mu.Unlock()
				activations[payload.NodeID()] = append(activations[payload.NodeID()], payload.ExecutionID)
				return nil
			})

			// Helper to get nodeID from executionID (since format is inst:node:uuid)
			// Actually, let's add a helper to TaskPayload or just split here.
			// I'll add a helper method to TaskPayload in an update if needed,
			// but for now I'll just split in the handler.

			// Start workflow
			instID, err := mgr.StartWorkflow(ctx, testFile.Workflow.ID, nil)
			assert.NoError(t, err)

			// Process steps
			completedSteps := make(map[string]int) // NodeID -> Count

			for _, step := range scenario.Steps {
				// Find an activation for this nodeID that hasn't been used yet
				var execID string
				for {
					mu.Lock()
					ids := activations[step.NodeID]
					available := len(ids) - completedSteps[step.NodeID]
					if available > 0 {
						execID = ids[completedSteps[step.NodeID]]
						completedSteps[step.NodeID]++
						mu.Unlock()
						break
					}
					mu.Unlock()
					// In a real test we'd have a timeout or wait channel,
					// but since it's synchronous manager for now, it should be there.
					break
				}

				if execID == "" {
					t.Fatalf("No activation found for node %s", step.NodeID)
				}

				err := mgr.TaskDone(ctx, execID, step.Results)
				assert.NoError(t, err)
			}

			// Verify final state
			finalInst, err := mgr.GetStatus(ctx, instID)
			assert.NoError(t, err)

			// 1. Check expected status
			assert.Equal(t, scenario.ExpectedStatus, finalInst.Status, "Workflow status does not match")

			if finalInst.Status == StatusCompleted {
				assert.Empty(t, finalInst.Context.GetTokens(), "Tokens should be empty after completion")
			} else {
				var actualTokenNodes []string
				for _, tok := range finalInst.Context.GetTokens() {
					actualTokenNodes = append(actualTokenNodes, tok.NodeID)
				}
				assert.ElementsMatch(t, scenario.ExpectedTokens, actualTokenNodes, "Tokens do not match expected end state")
			}
		})
	}
}

// Removed local helper
