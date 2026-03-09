package workflow

import (
	"encoding/json"
	"os"
)

func LoadBlueprint(path string) (*Blueprint, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var bp Blueprint
	return &bp, json.Unmarshal(b, &bp)
}
