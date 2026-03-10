package workflow

import (
	"encoding/json"
	"os"
)

func LoadWorkflow(path string) (*Workflow, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var wf Workflow
	return &wf, json.Unmarshal(b, &wf)
}
