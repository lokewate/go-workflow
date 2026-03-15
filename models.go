package workflow

import (
	"context"
	"errors"
	"strings"

	"github.com/lokewate/go-workflow/state"
)

// Sentinel errors for programmatic error handling.
var (
	// ErrWorkflowNotFound is returned when a workflow blueprint cannot be resolved.
	ErrWorkflowNotFound = errors.New("workflow definition not found")
	// ErrInstanceNotFound is returned when a workflow instance cannot be found.
	ErrInstanceNotFound = errors.New("instance not found")
	// ErrNodeNotFound is returned when a node ID cannot be resolved within a workflow.
	ErrNodeNotFound = errors.New("node not found")
	// ErrNoStartEvent is returned when a workflow has no START event node.
	ErrNoStartEvent = errors.New("workflow has no START event")
	// ErrInvalidExecutionID is returned when an execution ID has an invalid format.
	ErrInvalidExecutionID = errors.New("invalid execution ID format")
	// ErrNoMatchingCondition is returned when an ExclusiveSplit has no matching outgoing edge.
	ErrNoMatchingCondition = errors.New("exclusive split: no condition matched")
	// ErrHandlerNotRegistered is returned when a task is activated but no handler is registered.
	ErrHandlerNotRegistered = errors.New("task activation handler not registered")
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
	// StatusFailed means the workflow encountered an error during a transition.
	StatusFailed WorkflowStatus = "FAILED"
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
	// WorkflowID is the ID of the blueprint this instance is pinned to.
	WorkflowID string `json:"workflow_id"`
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

// NodeID returns the node part of the ExecutionID.
func (p TaskPayload) NodeID() string {
	parts := strings.Split(p.ExecutionID, ":")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// TaskActivationHandler defines what happens when a TASK node is activated.
type TaskActivationHandler func(ctx context.Context, payload TaskPayload) error

// WorkflowCompletionHandler is called when the workflow reaches an end state (i.e. COMPLETED or FAILED).
type WorkflowCompletionHandler func(ctx context.Context, instance *WorkflowInstance) error

// Manager provides the external interface for managing workflow execution.
type Manager interface {
	// RegisterTaskHandler allows a single global handler to be registered.
	// This handler will get called when any Task needs to be executed.
	RegisterTaskHandler(handler TaskActivationHandler)

	// RegisterWorkflowCompletionHandler registers a callback for when the entire workflow finishes.
	RegisterWorkflowCompletionHandler(handler WorkflowCompletionHandler)

	// StartWorkflow takes a workflow definition and executes it.
	// initialContext is used to populate the global state managed by the Manager.
	// Returns the ID assigned to this workflow instance.
	StartWorkflow(ctx context.Context, workflowJSON []byte, initialContext map[string]any) (string, error)

	// TaskDone should be called when a task worker finishes.
	// taskExecutionID is the unique reference provided in the TaskPayload when the task was activated.
	TaskDone(ctx context.Context, taskExecutionID string, outputs map[string]any) error

	// GetStatus retrieves the current state and audit trail.
	GetStatus(ctx context.Context, instanceID string) (*WorkflowInstance, error)
}
