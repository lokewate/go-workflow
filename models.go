package workflow

import (
	"workflow-engine/state"
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
	// ID is the unique identifier for the node.
	ID string `json:"id"`
	// Type specifies whether the node is a task, gateway, or event.
	Type NodeType `json:"type"`
	// Name is a human-readable name for the node.
	Name string `json:"name"`
	// TaskType specifies the kind of task if Type is NodeTypeTask.
	TaskType string `json:"task_type,omitempty"`
	// EventType specifies the kind of event if Type is NodeTypeEvent.
	EventType EventType `json:"event_type,omitempty"`
	// GatewayType specifies the behavior if Type is NodeTypeGateway.
	GatewayType GatewayType `json:"gateway_type,omitempty"`
	// Outputs defines the mapping from local task results to global context keys.
	Outputs map[string]string `json:"outputs,omitempty"`
	// X is the horizontal position of the node in a designer.
	X int `json:"x"`
	// Y is the vertical position of the node in a designer.
	Y int `json:"y"`
}

// Edge represents a directed connection between two nodes.
type Edge struct {
	// ID is the unique identifier for the edge.
	ID string `json:"id"`
	// SourceID is the ID of the node where the edge starts.
	SourceID string `json:"source_id"`
	// TargetID is the ID of the node where the edge ends.
	TargetID string `json:"target_id"`
	// Condition is an optional boolean expression that determines if this edge is followed.
	Condition *string `json:"condition,omitempty"`
}

// Workflow defines the static structure of a workflow.
type Workflow struct {
	// ID is the unique identifier for the workflow definition.
	ID string `json:"workflow_id"`
	// Name is the human-readable name of the workflow.
	Name string `json:"name"`
	// Version is the version number of the workflow definition.
	Version int `json:"version"`
	// Nodes is the list of steps and gateways in the workflow.
	Nodes []Node `json:"nodes"`
	// Edges is the list of connections between nodes.
	Edges []Edge `json:"edges"`
}

// WorkflowInstance represents a single execution of a workflow definition.
type WorkflowInstance struct {
	// ID is the unique identifier for this specific execution instance.
	ID string `json:"id"`
	// Status tracks the current state of the instance (e.g., "ACTIVE", "COMPLETED").
	Status string `json:"status"`
	// Context holds the dynamic state and tokens for this instance.
	Context state.GlobalContext `json:"-"`
}
