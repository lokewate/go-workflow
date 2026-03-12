package workflow

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"workflow-engine/state"

	"github.com/google/uuid"
)

type manager struct {
	Repo       Repo
	Blueprints map[string]*Workflow
	handler    TaskActivationHandler
	mu         sync.Mutex // For coordinating atomic TaskDone
}

// NewWorkflowManager creates a new instance of the workflow manager.
func NewWorkflowManager(repo Repo) Manager {
	return &manager{
		Repo:       repo,
		Blueprints: make(map[string]*Workflow),
	}
}

// AddWorkflow registers a workflow blueprint.
func (m *manager) AddWorkflow(wf *Workflow) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Blueprints[wf.ID] = wf
}

func (m *manager) RegisterTaskHandler(handler TaskActivationHandler) {
	m.handler = handler
}

func (m *manager) getNode(wf *Workflow, id string) (Node, bool) {
	for _, n := range wf.Nodes {
		if n.ID == id {
			return n, true
		}
	}
	return Node{}, false
}

func (m *manager) StartWorkflow(ctx context.Context, workflowID string, initialCtx map[string]any) (string, error) {
	m.mu.Lock()
	wf, ok := m.Blueprints[workflowID]
	m.mu.Unlock()
	if !ok {
		return "", fmt.Errorf("workflow definition %s not found", workflowID)
	}

	instID := uuid.NewString()
	inst := &WorkflowInstance{
		ID:         instID,
		WorkflowID: workflowID,
		Status:     StatusActive,
	}

	// Initialize context
	inst.Context = state.NewMapContext(nil, nil) // Repo will provide load/save later if persistent
	for k, v := range initialCtx {
		inst.Context.Set(k, v)
	}

	if err := m.Repo.Save(ctx, inst); err != nil {
		return "", err
	}

	// Find START node
	var startNode *Node
	for _, n := range wf.Nodes {
		if n.Type == NodeTypeInternal && n.InternalType == InternalTypeEvent && n.EventType == StartEvent {
			startNode = &n
			break
		}
	}

	if startNode == nil {
		return "", fmt.Errorf("workflow %s has no START event", workflowID)
	}

	log.Printf("[Manager] Starting workflow %s (instance: %s)", workflowID, instID)
	if err := m.processTarget(ctx, wf, inst, startNode.ID); err != nil {
		return "", err
	}

	return instID, m.Repo.Save(ctx, inst)
}

// TaskDone is called when a task worker finishes.
func (m *manager) TaskDone(ctx context.Context, executionID string, outputs map[string]any) error {
	m.mu.Lock() // Ensure atomicity as per System Integrity Rules
	defer m.mu.Unlock()

	parts := strings.Split(executionID, ":")
	if len(parts) < 2 {
		return fmt.Errorf("invalid executionID: %s", executionID)
	}
	instID := parts[0]
	nodeID := parts[1]

	log.Printf("[Manager] TaskDone: instID=%s, nodeID=%s, execID=%s", instID, nodeID, executionID)
	inst, err := m.Repo.Get(ctx, instID)
	if err != nil {
		return err
	}

	// Idempotency: Verify that the token still exists at this node with the matching ExecutionID
	found := false
	tokens := inst.Context.GetTokens()
	for _, t := range tokens {
		if t.NodeID == nodeID && t.ID == executionID {
			found = true
			break
		}
	}
	if !found {
		log.Printf("[Manager] TaskDone: executionID %s already processed or invalid, ignoring", executionID)
		return nil
	}

	// Resolve the correct version of the blueprint pinned to this instance
	wf, ok := m.Blueprints[inst.WorkflowID]
	if !ok {
		return fmt.Errorf("blueprint %s not found for instance %s", inst.WorkflowID, instID)
	}

	node, ok := m.getNode(wf, nodeID)
	if !ok {
		return fmt.Errorf("node %s not found in workflow", nodeID)
	}

	// Output Mapping
	for local, global := range node.OutputMapping {
		if val, ok := outputs[local]; ok {
			inst.Context.Set(global, val)
		}
	}

	m.removeToken(inst, executionID)
	err = m.transition(ctx, wf, inst, nodeID)
	if err != nil {
		return err
	}

	return m.Repo.Save(ctx, inst)
}

func (m *manager) GetStatus(ctx context.Context, instanceID string) (*WorkflowInstance, error) {
	return m.Repo.Get(ctx, instanceID)
}

// transition moves tokens from a source node to its targets based on gateway logic.
func (m *manager) transition(c context.Context, wf *Workflow, inst *WorkflowInstance, sourceID string) error {
	sourceNode, ok := m.getNode(wf, sourceID)
	if !ok {
		return fmt.Errorf("source node %s not found", sourceID)
	}
	log.Printf("[Manager] transition: sourceID=%s Type=%s", sourceID, sourceNode.Type)
	edges := m.getOutgoing(wf, sourceID)
	var targets []string

	if sourceNode.Type == NodeTypeInternal && sourceNode.InternalType == InternalTypeGateway && sourceNode.GatewayType == ExclusiveSplit {
		for _, edge := range edges {
			if edge.Condition == nil || state.EvaluateCondition(*edge.Condition, inst.Context) {
				targets = append(targets, edge.TargetID)
				break
			}
		}
	} else if sourceNode.Type == NodeTypeInternal && sourceNode.InternalType == InternalTypeGateway && sourceNode.GatewayType == ParallelSplit {
		for _, edge := range edges {
			targets = append(targets, edge.TargetID)
		}
	} else {
		// Default: pass through (for events or simple tasks)
		for _, edge := range edges {
			targets = append(targets, edge.TargetID)
		}
	}

	for _, tID := range targets {
		if err := m.processTarget(c, wf, inst, tID); err != nil {
			return err
		}
	}
	return nil
}

// processTarget determines how to handle a specific node reached during a transition.
func (m *manager) processTarget(c context.Context, wf *Workflow, inst *WorkflowInstance, nodeID string) error {
	node, ok := m.getNode(wf, nodeID)
	if !ok {
		return fmt.Errorf("node %s not found", nodeID)
	}

	log.Printf("[Manager] processTarget: nodeID=%s Type=%s", nodeID, node.Type)

	if node.Type == NodeTypeInternal {
		switch node.InternalType {
		case InternalTypeEvent:
			if node.EventType == StartEvent {
				return m.transition(c, wf, inst, nodeID)
			}
			if node.EventType == EndEvent {
				inst.Status = StatusCompleted
				inst.Context.SetTokens(nil)
				return nil
			}
		case InternalTypeGateway:
			if node.GatewayType == ParallelJoin {
				tokens := inst.Context.GetTokens()
				tokens = append(tokens, state.Token{ID: uuid.NewString(), NodeID: nodeID, Status: state.TokenWaiting})
				inst.Context.SetTokens(tokens)
				if len(m.getTokensAt(inst, nodeID)) >= len(m.getIncoming(wf, nodeID)) {
					m.removeTokensAt(inst, nodeID)
					return m.transition(c, wf, inst, nodeID)
				}
				return nil
			}
			// Other gateways (joins) behave like pass-through if not ParallelJoin
			return m.transition(c, wf, inst, nodeID)
		}
	}

	if node.Type == NodeTypeTask {
		// Input Mapping
		inputs := make(map[string]any)
		for local, global := range node.InputMapping {
			inputs[local] = inst.Context.Get(global)
		}

		executionID := fmt.Sprintf("%s:%s:%s", inst.ID, nodeID, uuid.NewString())
		tokens := inst.Context.GetTokens()
		tokens = append(tokens, state.Token{ID: executionID, NodeID: nodeID, Status: state.TokenActive})
		inst.Context.SetTokens(tokens)

		if m.handler != nil {
			payload := TaskPayload{
				ExecutionID: executionID,
				TaskID:      node.TaskID,
				Inputs:      inputs,
			}
			return m.handler(c, payload)
		}
		return nil
	}

	return nil
}

// getOutgoing retrieves all edges where the specified node is the source.
func (m *manager) getOutgoing(wf *Workflow, id string) (res []Edge) {
	for _, edge := range wf.Edges {
		if edge.SourceID == id {
			res = append(res, edge)
		}
	}
	return
}

// getIncoming retrieves all edges where the specified node is the target.
func (m *manager) getIncoming(wf *Workflow, id string) (res []Edge) {
	for _, edge := range wf.Edges {
		if edge.TargetID == id {
			res = append(res, edge)
		}
	}
	return
}

// getTokensAt returns all active or waiting tokens currently at the specified node.
func (m *manager) getTokensAt(inst *WorkflowInstance, id string) (res []state.Token) {
	for _, t := range inst.Context.GetTokens() {
		if t.NodeID == id {
			res = append(res, t)
		}
	}
	return
}

// removeToken deletes a specific token by its unique ExecutionID.
func (m *manager) removeToken(inst *WorkflowInstance, executionID string) {
	var next []state.Token
	for _, t := range inst.Context.GetTokens() {
		if t.ID != executionID {
			next = append(next, t)
		}
	}
	inst.Context.SetTokens(next)
}

// removeTokensAt deletes all tokens from the instance that are currently at the specified node.
func (m *manager) removeTokensAt(inst *WorkflowInstance, nodeID string) {
	var next []state.Token
	for _, t := range inst.Context.GetTokens() {
		if t.NodeID != nodeID {
			next = append(next, t)
		}
	}
	inst.Context.SetTokens(next)
}
