package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"workflow-engine/internal/workflow"

	"github.com/stretchr/testify/assert"
)

type Suite struct {
	Blueprint workflow.Blueprint `json:"blueprint"`
	Scenarios []struct {
		Name  string `json:"name"`
		Steps []struct {
			NodeID  string                 `json:"node_id"`
			Results map[string]interface{} `json:"results"`
		} `json:"steps"`
		ExpectedTokens []string `json:"expected_tokens"`
	} `json:"scenarios"`
}

func TestE2E(t *testing.T) {
	data, _ := os.ReadFile("test_suite.json")
	var suite Suite
	json.Unmarshal(data, &suite)

	for _, s := range suite.Scenarios {
		t.Run(s.Name, func(t *testing.T) {
			repo := workflow.NewMemoryRepo()
			eng := &workflow.Engine{Repo: repo, Blueprint: &suite.Blueprint}
			ctx := context.Background()

			inst := &workflow.Instance{
				ID:      "1",
				Tokens:  []workflow.Token{{NodeID: "start"}},
				Payload: make(map[string]interface{}),
			}
			repo.Save(ctx, inst)

			for _, step := range s.Steps {
				eng.CompleteTask(ctx, "1", step.NodeID, step.Results)
			}

			final, _ := repo.Get(ctx, "1")
			var ids []string
			for _, tok := range final.Tokens {
				ids = append(ids, tok.NodeID)
			}
			assert.ElementsMatch(t, s.ExpectedTokens, ids)
		})
	}
}
