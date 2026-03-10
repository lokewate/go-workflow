package workflow

import (
	"workflow-engine/internal/workflow/context"
)

// NodeType defines the kind of node in a workflow (e.g., TASK, GATEWAY).
type NodeType string

const (
	// NodeTypeTask represents a manual or automated task.
	NodeTypeTask NodeType = "TASK"
	// NodeTypeGateway represents a decision point or merge point.
	NodeTypeGateway NodeType = "GATEWAY"
	// NodeTypeEvent represents a start or end event.
	NodeTypeEvent NodeType = "EVENT"
)

// EventType defines the kind of event node.
type EventType string

const (
	// StartEvent marks the entry point of a workflow path.
	StartEvent EventType = "START"
	// EndEvent marks the termination point of a workflow path.
	EndEvent EventType = "END"
)

// GatewayType defines the behavior of a gateway node.
type GatewayType string

const (
	// ExclusiveSplit selects exactly one outgoing path based on conditions.
	ExclusiveSplit GatewayType = "EXCLUSIVE_SPLIT"
	// ExclusiveJoin merges multiple paths into one (pass-through).
	ExclusiveJoin GatewayType = "EXCLUSIVE_JOIN"
	// ParallelSplit activates all outgoing paths.
	ParallelSplit GatewayType = "PARALLEL_SPLIT"
	// ParallelJoin waits for all incoming paths to complete.
	ParallelJoin GatewayType = "PARALLEL_JOIN"
)

// Node represents a single step or gateway in a workflow.
type Node struct {
	ID          string            `json:"id"`
	Type        NodeType          `json:"type"`
	Name        string            `json:"name"`
	TaskType    string            `json:"task_type,omitempty"`
	EventType   EventType         `json:"event_type,omitempty"`
	GatewayType GatewayType       `json:"gateway_type,omitempty"`
	Outputs     map[string]string `json:"outputs,omitempty"`
	X           int               `json:"x"`
	Y           int               `json:"y"`
}

// Edge represents a directed connection between two nodes.
type Edge struct {
	ID        string  `json:"id"`
	SourceID  string  `json:"source_id"`
	TargetID  string  `json:"target_id"`
	Condition *string `json:"condition,omitempty"`
}

// Workflow defines the static structure of a workflow.
type Workflow struct {
	ID      string `json:"workflow_id"`
	Name    string `json:"name"`
	Version int    `json:"version"`
	Nodes   []Node `json:"nodes"`
	Edges   []Edge `json:"edges"`
}

// WorkflowInstance represents a single execution of a workflow definition.
type WorkflowInstance struct {
	ID      string                `json:"id"`
	Status  string                `json:"status"` // e.g., "ACTIVE", "COMPLETED"
	Context context.GlobalContext `json:"-"`
}
