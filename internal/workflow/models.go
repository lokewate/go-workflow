package workflow

import (
	"sync"
)

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

// Node represents a single step or gateway in a workflow.
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

// Workflow defines the static structure of a workflow.
type Workflow struct {
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

// GlobalContext provides an interface for accessing and modifying workflow-wide state.
// It allows for different backends (e.g., in-memory map, database) to store the state.
type GlobalContext interface {
	Get(key string) interface{}
	Set(key string, val interface{})
	AsMap() map[string]interface{}
}

// MapContext is an in-memory implementation of GlobalContext using a map.
type MapContext struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

// NewMapContext initializes a new MapContext.
func NewMapContext() *MapContext {
	return &MapContext{data: make(map[string]interface{})}
}

// Get retrieves the value for a given key from the context.
func (m *MapContext) Get(key string) interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.data[key]
}

// Set stores a value for a given key in the context.
func (m *MapContext) Set(key string, val interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = make(map[string]interface{})
	}
	m.data[key] = val
}

// AsMap returns a point-in-time copy of the internal data as a map.
func (m *MapContext) AsMap() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Return a copy to prevent external mutation
	res := make(map[string]interface{})
	for k, v := range m.data {
		res[k] = v
	}
	return res
}

// WorkflowInstance represents a single execution of a workflow definition.
type WorkflowInstance struct {
	ID              string  `json:"id"`
	GlobalContextID string  `json:"global_context_id"`
	Tokens          []Token `json:"tokens"`
}
