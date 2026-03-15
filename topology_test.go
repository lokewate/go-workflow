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
	Results map[string]any `json:"results"`
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

			wfBytes, err := json.Marshal(&testFile.Workflow)
			assert.NoError(t, err)

			// Capture task activations
			var mu sync.Mutex
			activations := make(map[string][]string) // NodeID -> []ExecutionID

			mgr.RegisterTaskHandler(func(ctx context.Context, payload TaskPayload) error {
				mu.Lock()
				defer mu.Unlock()
				activations[payload.NodeID()] = append(activations[payload.NodeID()], payload.ExecutionID)
				return nil
			})

			// Start workflow
			instID, err := mgr.StartWorkflow(ctx, wfBytes, nil)
			assert.NoError(t, err)

			// Process steps
			completedSteps := make(map[string]int) // NodeID -> Count

			for _, step := range scenario.Steps {
				// Find an activation for this nodeID that hasn't been used yet
				var execID string
				mu.Lock()
				ids := activations[step.NodeID]
				available := len(ids) - completedSteps[step.NodeID]
				if available > 0 {
					execID = ids[completedSteps[step.NodeID]]
					completedSteps[step.NodeID]++
				}
				mu.Unlock()

				if execID == "" {
					t.Fatalf("No activation found for node %s", step.NodeID)
				}

				err := mgr.TaskDone(ctx, execID, step.Results)
				assert.NoError(t, err)
			}

			// Verify final state
			finalInst, err := mgr.GetStatus(ctx, instID)
			assert.NoError(t, err)

			// Check expected status
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
