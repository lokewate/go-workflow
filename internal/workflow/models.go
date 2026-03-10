package workflow

// NodeType defines the kind of node in a workflow (e.g., TASK, GATEWAY).
type NodeType string

const (
	// NodeTypeTask represents a manual or automated task.
	NodeTypeTask NodeType = "TASK"
	// NodeTypeGateway represents a decision point or merge point.
	NodeTypeGateway NodeType = "GATEWAY"
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

// Node represents a single step or gateway in a workflow blueprint.
type Node struct {
	ID          string            `json:"id"`
	Type        NodeType          `json:"type"`
	Name        string            `json:"name"`
	TaskType    string            `json:"task_type,omitempty"`
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

// Blueprint defines the static structure of a workflow.
type Blueprint struct {
	ID      string `json:"workflow_id"`
	Name    string `json:"name"`
	Version int    `json:"version"`
	Nodes   []Node `json:"nodes"`
	Edges   []Edge `json:"edges"`
}

// TokenStatus defines the current state of a workflow token.
type TokenStatus string

const (
	// TokenActive indicates the token is currently at a task being performed.
	TokenActive TokenStatus = "ACTIVE"
	// TokenWaiting indicates the token is waiting at a join gateway.
	TokenWaiting TokenStatus = "WAITING"
)

// Token tracks the execution progress within a workflow instance.
type Token struct {
	ID     string      `json:"id"`
	NodeID string      `json:"node_id"`
	Status TokenStatus `json:"status"`
}

// Instance represents a single execution of a workflow blueprint.
type Instance struct {
	ID      string                 `json:"id"`
	Payload map[string]interface{} `json:"payload"`
	Tokens  []Token                `json:"tokens"`
}
