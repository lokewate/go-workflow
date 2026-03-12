package workflow

import (
	"context"
	"workflow-engine/state"
)

// NodeType defines the primary category of a node.
type NodeType string

const (
	// NodeTypeTask represents an execution-driven task.
	NodeTypeTask NodeType = "TASK"
	// NodeTypeInternal represents a logic-driven node (gateway or event).
	NodeTypeInternal NodeType = "INTERNAL"
)

// InternalType specifies the logic for INTERNAL nodes.
type InternalType string

const (
	// InternalTypeGateway defines branching or merging logic.
	InternalTypeGateway InternalType = "GATEWAY"
	// InternalTypeEvent defines start or termination points.
	InternalTypeEvent InternalType = "EVENT"
)

// EventType defines the kind of event node.
type EventType string

const (
	// StartEvent marks the entry point of a workflow path.
	StartEvent EventType = "START"
	// EndEvent marks the termination point of a workflow path.
	EndEvent EventType = "END"
)

// GatewayType defines the routing behavior.
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

// WorkflowStatus defines the possible states of a workflow instance.
type WorkflowStatus string

const (
	// StatusActive means the workflow is currently running.
	StatusActive WorkflowStatus = "ACTIVE"
	// StatusCompleted means the workflow has reached an END event.
	StatusCompleted WorkflowStatus = "COMPLETED"
)

// Node represents a single step or decision point.
type Node struct {
	// ID is the unique identifier for the node.
	ID string `json:"id"`
	// Type specifies whether the node is a task or internal logic.
	Type NodeType `json:"type"`
	// Name is a human-readable name for the node.
	Name string `json:"name"`

	// Internal Logic
	// InternalType specifies if it's a gateway or event (for INTERNAL type).
	InternalType InternalType `json:"internal_type,omitempty"`
	// GatewayType specifies the routing behavior (for GATEWAY internal type).
	GatewayType GatewayType `json:"gateway_type,omitempty"`
	// EventType specifies the event behavior (for EVENT internal type).
	EventType EventType `json:"event_type,omitempty"`

	// Task Logic
	// TaskID is the identifier used by the TaskManager to locate the execution logic.
	TaskID string `json:"task_id,omitempty"`

	// Mapping Logic: Prevents variable leakage.
	// InputMapping: Map[LocalTaskVar] -> GlobalContextKey
	InputMapping map[string]string `json:"input_mapping,omitempty"`
	// OutputMapping: Map[LocalTaskVar] -> GlobalContextKey
	OutputMapping map[string]string `json:"output_mapping,omitempty"`

	// Visualization helpers
	X int `json:"x"`
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
	// Status tracks the current state of the instance.
	Status WorkflowStatus `json:"status"`
	// Context holds the dynamic state and tokens for this instance.
	Context state.GlobalContext `json:"-"`
}

// TaskPayload contains the information provided to the orchestrator
// to trigger a specific task execution.
type TaskPayload struct {
	// ExecutionID is a unique identifier for this specific task activation.
	ExecutionID string
	// TaskID identifies which task needs to be executed.
	TaskID string
	// Inputs is the set of data from the global context mapping to the task.
	Inputs map[string]any
}

// TaskActivationHandler defines what happens when a TASK node is activated.
type TaskActivationHandler func(ctx context.Context, payload TaskPayload) error

// Manager provides the external interface for trigger transitions and registering for task activations.
type Manager interface {
	// RegisterTaskHandler defines what happens when a TASK node is activated.
	RegisterTaskHandler(handler TaskActivationHandler)

	// StartWorkflow creates and begins a new execution instance of a specific workflow definition.
	StartWorkflow(ctx context.Context, workflowID string, initialCtx map[string]any) (string, error)

	// TaskDone is called when a task worker finishes.
	TaskDone(ctx context.Context, executionID string, outputs map[string]any) error

	// GetStatus retrieves the current state and audit trail.
	GetStatus(ctx context.Context, instanceID string) (*WorkflowInstance, error)
}
