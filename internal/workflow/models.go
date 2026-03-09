package workflow

type NodeType string

const (
	NodeTypeTask    NodeType = "TASK"
	NodeTypeGateway NodeType = "GATEWAY"
)

type GatewayType string

const (
	ExclusiveSplit GatewayType = "EXCLUSIVE_SPLIT"
	ExclusiveJoin  GatewayType = "EXCLUSIVE_JOIN"
	ParallelSplit  GatewayType = "PARALLEL_SPLIT"
	ParallelJoin   GatewayType = "PARALLEL_JOIN"
)

type Node struct {
	ID          string            `json:"id"`
	Type        NodeType          `json:"type"`
	GatewayType GatewayType       `json:"gateway_type,omitempty"`
	Outputs     map[string]string `json:"outputs,omitempty"` // GlobalKey: LocalResultKey
}

type Edge struct {
	ID        string  `json:"id"`
	SourceID  string  `json:"source_id"`
	TargetID  string  `json:"target_id"`
	Condition *string `json:"condition,omitempty"`
}

type Blueprint struct {
	ID    string          `json:"id"`
	Nodes map[string]Node `json:"nodes"`
	Edges []Edge          `json:"edges"`
}

type TokenStatus string

const (
	TokenActive  TokenStatus = "ACTIVE"
	TokenWaiting TokenStatus = "WAITING"
)

type Token struct {
	ID     string      `json:"id"`
	NodeID string      `json:"node_id"`
	Status TokenStatus `json:"status"`
}

type Instance struct {
	ID      string                 `json:"id"`
	Payload map[string]interface{} `json:"payload"`
	Tokens  []Token                `json:"tokens"`
}
