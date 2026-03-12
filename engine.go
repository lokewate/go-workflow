package workflow

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/lokewate/go-workflow/state"

	"github.com/google/uuid"
)

type manager struct {
	repo       Repo
	blueprints map[string]*Workflow
	handler    TaskActivationHandler
	logger     *slog.Logger

	// Per-instance locks to avoid serializing all instances behind a single mutex.
	mu    sync.Mutex             // Protects blueprints map and instanceLocks map
	locks map[string]*sync.Mutex // Per-instance mutexes
}

// NewWorkflowManager creates a new instance of the workflow manager.
func NewWorkflowManager(repo Repo, opts ...ManagerOption) Manager {
	m := &manager{
		repo:       repo,
		blueprints: make(map[string]*Workflow),
		locks:      make(map[string]*sync.Mutex),
		logger:     slog.Default(),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// ManagerOption allows configuring the manager.
type ManagerOption func(*manager)

// WithLogger sets a custom structured logger.
func WithLogger(logger *slog.Logger) ManagerOption {
	return func(m *manager) {
		m.logger = logger
	}
}

// instanceLock returns the per-instance mutex, creating one if needed.
func (m *manager) instanceLock(instID string) *sync.Mutex {
	m.mu.Lock()
	defer m.mu.Unlock()
	if l, ok := m.locks[instID]; ok {
		return l
	}
	l := &sync.Mutex{}
	m.locks[instID] = l
	return l
}

// AddWorkflow registers a workflow blueprint.
func (m *manager) AddWorkflow(wf *Workflow) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blueprints[wf.ID] = wf
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

func (m *manager) getBlueprint(id string) (*Workflow, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	wf, ok := m.blueprints[id]
	return wf, ok
}

func (m *manager) StartWorkflow(ctx context.Context, workflowID string, initialCtx map[string]any) (string, error) {
	wf, ok := m.getBlueprint(workflowID)
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrWorkflowNotFound, workflowID)
	}

	instID := uuid.NewString()
	inst := &WorkflowInstance{
		ID:         instID,
		WorkflowID: workflowID,
		Status:     StatusActive,
	}

	// Wire context to the repo's persistence layer
	inst.Context = m.repo.NewContext(instID)

	for k, v := range initialCtx {
		inst.Context.Set(ctx, k, v)
	}

	if err := m.repo.Save(ctx, inst); err != nil {
		return "", fmt.Errorf("save new instance: %w", err)
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
		return "", fmt.Errorf("%w: %s", ErrNoStartEvent, workflowID)
	}

	m.logger.Info("starting workflow", "workflowID", workflowID, "instanceID", instID)
	if err := m.processTarget(ctx, wf, inst, startNode.ID); err != nil {
		inst.Status = StatusFailed
		_ = m.repo.Save(ctx, inst)
		return "", err
	}

	return instID, m.repo.Save(ctx, inst)
}

// TaskDone is called when a task worker finishes.
func (m *manager) TaskDone(ctx context.Context, executionID string, outputs map[string]any) error {
	parts := strings.Split(executionID, ":")
	if len(parts) < 2 {
		return fmt.Errorf("%w: %s", ErrInvalidExecutionID, executionID)
	}
	instID := parts[0]
	nodeID := parts[1]

	// Per-instance lock ensures atomicity without blocking other instances.
	lock := m.instanceLock(instID)
	lock.Lock()
	defer lock.Unlock()

	m.logger.Info("TaskDone", "instanceID", instID, "nodeID", nodeID, "executionID", executionID)
	inst, err := m.repo.Get(ctx, instID)
	if err != nil {
		return fmt.Errorf("get instance %s: %w", instID, err)
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
		m.logger.Warn("TaskDone: execution already processed, ignoring", "executionID", executionID)
		return nil
	}

	// Resolve the correct version of the blueprint pinned to this instance
	wf, ok := m.getBlueprint(inst.WorkflowID)
	if !ok {
		return fmt.Errorf("%w: %s (instance %s)", ErrWorkflowNotFound, inst.WorkflowID, instID)
	}

	node, ok := m.getNode(wf, nodeID)
	if !ok {
		return fmt.Errorf("%w: %s in workflow %s", ErrNodeNotFound, nodeID, inst.WorkflowID)
	}

	// Output Mapping
	for local, global := range node.OutputMapping {
		if val, ok := outputs[local]; ok {
			inst.Context.Set(ctx, global, val)
		}
	}

	m.removeToken(ctx, inst, executionID)
	if err := m.transition(ctx, wf, inst, nodeID); err != nil {
		inst.Status = StatusFailed
		_ = m.repo.Save(ctx, inst)
		return err
	}

	return m.repo.Save(ctx, inst)
}

func (m *manager) GetStatus(ctx context.Context, instanceID string) (*WorkflowInstance, error) {
	return m.repo.Get(ctx, instanceID)
}

// transition moves tokens from a source node to its targets based on gateway logic.
func (m *manager) transition(ctx context.Context, wf *Workflow, inst *WorkflowInstance, sourceID string) error {
	sourceNode, ok := m.getNode(wf, sourceID)
	if !ok {
		return fmt.Errorf("%w: %s", ErrNodeNotFound, sourceID)
	}
	m.logger.Debug("transition", "sourceID", sourceID, "type", sourceNode.Type)
	edges := m.getOutgoing(wf, sourceID)
	var targets []string

	if sourceNode.Type == NodeTypeInternal && sourceNode.InternalType == InternalTypeGateway && sourceNode.GatewayType == ExclusiveSplit {
		for _, edge := range edges {
			if edge.Condition == nil {
				targets = append(targets, edge.TargetID)
				break
			}
			matched, err := state.EvaluateCondition(*edge.Condition, inst.Context)
			if err != nil {
				return fmt.Errorf("evaluate condition on edge %s: %w", edge.ID, err)
			}
			if matched {
				targets = append(targets, edge.TargetID)
				break
			}
		}
		if len(targets) == 0 {
			return fmt.Errorf("%w: node %s", ErrNoMatchingCondition, sourceID)
		}
	} else if sourceNode.Type == NodeTypeInternal && sourceNode.InternalType == InternalTypeGateway && sourceNode.GatewayType == ParallelSplit {
		for _, edge := range edges {
			targets = append(targets, edge.TargetID)
		}
	} else {
		// Default: pass through (for events, tasks, joins)
		for _, edge := range edges {
			targets = append(targets, edge.TargetID)
		}
	}

	for _, tID := range targets {
		if err := m.processTarget(ctx, wf, inst, tID); err != nil {
			return err
		}
	}
	return nil
}

// processTarget determines how to handle a specific node reached during a transition.
func (m *manager) processTarget(ctx context.Context, wf *Workflow, inst *WorkflowInstance, nodeID string) error {
	node, ok := m.getNode(wf, nodeID)
	if !ok {
		return fmt.Errorf("%w: %s", ErrNodeNotFound, nodeID)
	}

	m.logger.Debug("processTarget", "nodeID", nodeID, "type", node.Type)

	if node.Type == NodeTypeInternal {
		switch node.InternalType {
		case InternalTypeEvent:
			if node.EventType == StartEvent {
				return m.transition(ctx, wf, inst, nodeID)
			}
			if node.EventType == EndEvent {
				inst.Status = StatusCompleted
				inst.Context.SetTokens(ctx, nil)
				return nil
			}
		case InternalTypeGateway:
			if node.GatewayType == ParallelJoin {
				tokens := inst.Context.GetTokens()
				tokens = append(tokens, state.Token{ID: uuid.NewString(), NodeID: nodeID, Status: state.TokenWaiting})
				inst.Context.SetTokens(ctx, tokens)
				if len(m.getTokensAt(inst, nodeID)) >= len(m.getIncoming(wf, nodeID)) {
					m.removeTokensAt(ctx, inst, nodeID)
					return m.transition(ctx, wf, inst, nodeID)
				}
				return nil
			}
			// Other gateways (ExclusiveJoin, etc.) pass through
			return m.transition(ctx, wf, inst, nodeID)
		}
	}

	if node.Type == NodeTypeTask {
		if m.handler == nil {
			return ErrHandlerNotRegistered
		}

		// Input Mapping
		inputs := make(map[string]any)
		for local, global := range node.InputMapping {
			inputs[local] = inst.Context.Get(global)
		}

		executionID := fmt.Sprintf("%s:%s:%s", inst.ID, nodeID, uuid.NewString())
		tokens := inst.Context.GetTokens()
		tokens = append(tokens, state.Token{ID: executionID, NodeID: nodeID, Status: state.TokenActive})
		inst.Context.SetTokens(ctx, tokens)

		payload := TaskPayload{
			ExecutionID: executionID,
			TaskID:      node.TaskID,
			Inputs:      inputs,
		}
		return m.handler(ctx, payload)
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
func (m *manager) removeToken(ctx context.Context, inst *WorkflowInstance, executionID string) {
	var next []state.Token
	for _, t := range inst.Context.GetTokens() {
		if t.ID != executionID {
			next = append(next, t)
		}
	}
	inst.Context.SetTokens(ctx, next)
}

// removeTokensAt deletes all tokens from the instance that are currently at the specified node.
func (m *manager) removeTokensAt(ctx context.Context, inst *WorkflowInstance, nodeID string) {
	var next []state.Token
	for _, t := range inst.Context.GetTokens() {
		if t.NodeID != nodeID {
			next = append(next, t)
		}
	}
	inst.Context.SetTokens(ctx, next)
}
