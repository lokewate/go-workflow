package workflow

import (
	"context"
	"fmt"
	"log"
	"workflow-engine/state"

	"github.com/google/uuid"
)

// Engine handles the execution and transitions of workflow instances based on a workflow definition.
type Engine struct {
	Repo     Repo
	Workflow *Workflow
}

// getNode finds a node by its ID within the current workflow.
func (e *Engine) getNode(id string) (Node, bool) {
	for _, n := range e.Workflow.Nodes {
		if n.ID == id {
			return n, true
		}
	}
	return Node{}, false
}

// StartWorkflow begins workflow execution at the specified START event node.
func (e *Engine) StartWorkflow(c context.Context, inst *WorkflowInstance, startNodeID string) error {
	log.Printf("[Engine] StartWorkflow: instID=%s, startNodeID=%s", inst.ID, startNodeID)
	node, ok := e.getNode(startNodeID)
	if !ok || node.Type != NodeTypeEvent || node.EventType != StartEvent {
		return fmt.Errorf("start node %s not found or is not a START event", startNodeID)
	}

	// Status can be set to ACTIVE upon starting
	inst.Status = "ACTIVE"

	// Process the start node which will trigger transitions to the next nodes.
	err := e.processTarget(c, inst, startNodeID)
	if err != nil {
		return err
	}

	return e.Repo.Save(c, inst)
}

// CompleteTask marks a task as finished and triggers the next transitions.
func (e *Engine) CompleteTask(c context.Context, instID string, nodeID string, results map[string]interface{}) error {
	log.Printf("[Engine] CompleteTask: instID=%s, nodeID=%s", instID, nodeID)
	inst, err := e.Repo.Get(c, instID)
	if err != nil {
		log.Printf("[Engine] CompleteTask: failed to get instance: %v", err)
		return err
	}

	node, ok := e.getNode(nodeID)
	if !ok {
		log.Printf("[Engine] CompleteTask: node %s not found in workflow", nodeID)
		return fmt.Errorf("node %s not found", nodeID)
	}
	log.Printf("[Engine] CompleteTask: processing output mappings for node %s", nodeID)

	for global, local := range node.Outputs {
		if val, ok := results[local]; ok {
			inst.Context.Set(global, val)
		}
	}

	e.removeTokensAt(inst, nodeID)
	err = e.transition(c, inst, nodeID)
	if err != nil {
		return err
	}

	return e.Repo.Save(c, inst)
}

// transition moves tokens from a source node to its targets based on gateway logic.
func (e *Engine) transition(c context.Context, inst *WorkflowInstance, sourceID string) error {
	sourceNode, ok := e.getNode(sourceID)
	if !ok {
		log.Printf("[Engine] transition: source node %s not found", sourceID)
		return fmt.Errorf("source node %s not found", sourceID)
	}
	log.Printf("[Engine] transition: sourceID=%s sourceType=%s", sourceID, sourceNode.Type)
	edges := e.getOutgoing(sourceID)
	log.Printf("[Engine] transition: found %d outgoing edges", len(edges))
	var targets []string

	if sourceNode.Type == NodeTypeGateway && sourceNode.GatewayType == ExclusiveSplit {
		log.Printf("[Engine] transition: sourceID=%s sourceType=%s ExclusiveSplit", sourceID, sourceNode.Type)
		for _, edge := range edges {
			log.Printf("[Engine] transition: sourceID=%s sourceType=%s ExclusiveSplit edge.Condition=%v", sourceID, sourceNode.Type, edge.Condition)
			if edge.Condition == nil || state.EvaluateCondition(*edge.Condition, inst.Context) {
				targets = append(targets, edge.TargetID)
				break
			}
		}
		log.Printf("[Engine] transition: sourceID=%s sourceType=%s ExclusiveSplit targets=%v", sourceID, sourceNode.Type, targets)
	} else if sourceNode.Type == NodeTypeGateway && sourceNode.GatewayType == ParallelSplit {
		log.Printf("[Engine] transition: sourceID=%s sourceType=%s ParallelSplit", sourceID, sourceNode.Type)
		for _, edge := range edges {
			targets = append(targets, edge.TargetID)
		}
	} else {
		log.Printf("[Engine] transition: sourceID=%s sourceType=%s Default", sourceID, sourceNode.Type)
		for _, edge := range edges {
			targets = append(targets, edge.TargetID)
		}
	}

	for _, tID := range targets {
		log.Printf("[Engine] transition: processing target %s", tID)
		if err := e.processTarget(c, inst, tID); err != nil {
			return err
		}
	}
	return nil
}

// processTarget determines how to handle a specific node reached during a transition.
func (e *Engine) processTarget(c context.Context, inst *WorkflowInstance, nodeID string) error {
	node, ok := e.getNode(nodeID)
	if !ok {
		log.Printf("[Engine] processTarget: node %s not found", nodeID)
		return fmt.Errorf("node %s not found", nodeID)
	}
	log.Printf("[Engine] processTarget: nodeID=%s nodeType=%s", nodeID, node.Type)

	tokens := inst.Context.GetTokens()

	if node.Type == NodeTypeEvent {
		switch node.EventType {
		case StartEvent:
			// Automatically transition to the next connected node(s).
			return e.transition(c, inst, nodeID)
		case EndEvent:
			// Mark instance as completed and clear all tokens.
			inst.Status = "COMPLETED"
			inst.Context.SetTokens(nil)
			return nil
		}
	}

	if node.Type == NodeTypeTask {
		tokens = append(tokens, state.Token{ID: uuid.NewString(), NodeID: nodeID, Status: state.TokenActive})
		inst.Context.SetTokens(tokens)
		return nil
	}

	if node.GatewayType == ParallelJoin {
		tokens = append(tokens, state.Token{ID: uuid.NewString(), NodeID: nodeID, Status: state.TokenWaiting})
		inst.Context.SetTokens(tokens)
		if len(e.getTokensAt(inst, nodeID)) >= len(e.getIncoming(nodeID)) {
			e.removeTokensAt(inst, nodeID)
			return e.transition(c, inst, nodeID)
		}
		return nil
	}

	tokens = append(tokens, state.Token{ID: uuid.NewString(), NodeID: nodeID, Status: state.TokenActive})
	inst.Context.SetTokens(tokens)
	err := e.transition(c, inst, nodeID)
	e.removeTokensAt(inst, nodeID)
	return err
}

// getOutgoing retrieves all edges where the specified node is the source.
func (e *Engine) getOutgoing(id string) (res []Edge) {
	for _, edge := range e.Workflow.Edges {
		if edge.SourceID == id {
			res = append(res, edge)
		}
	}
	return
}

// getIncoming retrieves all edges where the specified node is the target.
func (e *Engine) getIncoming(id string) (res []Edge) {
	for _, edge := range e.Workflow.Edges {
		if edge.TargetID == id {
			res = append(res, edge)
		}
	}
	return
}

// getTokensAt returns all active or waiting tokens currently at the specified node.
func (e *Engine) getTokensAt(inst *WorkflowInstance, id string) (res []state.Token) {
	for _, t := range inst.Context.GetTokens() {
		if t.NodeID == id {
			res = append(res, t)
		}
	}
	return
}

// removeTokensAt deletes all tokens from the instance that are currently at the specified node.
func (e *Engine) removeTokensAt(inst *WorkflowInstance, nodeID string) {
	var next []state.Token
	for _, t := range inst.Context.GetTokens() {
		if t.NodeID != nodeID {
			next = append(next, t)
		}
	}
	inst.Context.SetTokens(next)
}
