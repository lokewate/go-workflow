package workflow

import (
	ctx "context"
	"fmt"
	"log"

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

// CompleteTask marks a task as finished and triggers the next transitions.
func (e *Engine) CompleteTask(c ctx.Context, instID string, nodeID string, results map[string]interface{}) error {
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
	gctx, err := e.Repo.GetContext(c, inst.GlobalContextID)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	for global, local := range node.Outputs {
		if val, ok := results[local]; ok {
			gctx.Set(global, val)
		}
	}

	if err := e.Repo.SaveContext(c, inst.GlobalContextID, gctx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	e.removeTokensAt(inst, nodeID)
	err = e.transition(c, inst, nodeID)
	if err != nil {
		return err
	}

	return e.Repo.Save(c, inst)
}

// transition moves tokens from a source node to its targets based on gateway logic.
func (e *Engine) transition(c ctx.Context, inst *WorkflowInstance, sourceID string) error {
	log.Printf("[Engine] transition: sourceID=%s", sourceID)
	sourceNode, ok := e.getNode(sourceID)
	if !ok {
		log.Printf("[Engine] transition: source node %s not found", sourceID)
		return fmt.Errorf("source node %s not found", sourceID)
	}
	edges := e.getOutgoing(sourceID)
	log.Printf("[Engine] transition: found %d outgoing edges", len(edges))
	var targets []string

	if sourceNode.Type == NodeTypeGateway && sourceNode.GatewayType == ExclusiveSplit {
		gctx, err := e.Repo.GetContext(c, inst.GlobalContextID)
		if err != nil {
			return fmt.Errorf("failed to load context for transition: %w", err)
		}

		for _, edge := range edges {
			if edge.Condition == nil || EvaluateCondition(*edge.Condition, gctx) {
				targets = append(targets, edge.TargetID)
				break
			}
		}
	} else if sourceNode.Type == NodeTypeGateway && sourceNode.GatewayType == ParallelSplit {
		for _, edge := range edges {
			targets = append(targets, edge.TargetID)
		}
	} else {
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
func (e *Engine) processTarget(c ctx.Context, inst *WorkflowInstance, nodeID string) error {
	log.Printf("[Engine] processTarget: nodeID=%s", nodeID)
	node, ok := e.getNode(nodeID)
	if !ok {
		log.Printf("[Engine] processTarget: node %s not found", nodeID)
		return fmt.Errorf("node %s not found", nodeID)
	}

	if node.Type == NodeTypeTask {
		inst.Tokens = append(inst.Tokens, Token{ID: uuid.NewString(), NodeID: nodeID, Status: TokenActive})
		return nil
	}

	if node.GatewayType == ParallelJoin {
		inst.Tokens = append(inst.Tokens, Token{ID: uuid.NewString(), NodeID: nodeID, Status: TokenWaiting})
		if len(e.getTokensAt(inst, nodeID)) >= len(e.getIncoming(nodeID)) {
			e.removeTokensAt(inst, nodeID)
			return e.transition(c, inst, nodeID)
		}
		return nil
	}

	inst.Tokens = append(inst.Tokens, Token{ID: uuid.NewString(), NodeID: nodeID, Status: TokenActive})
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
func (e *Engine) getTokensAt(inst *WorkflowInstance, id string) (res []Token) {
	for _, t := range inst.Tokens {
		if t.NodeID == id {
			res = append(res, t)
		}
	}
	return
}

// removeTokensAt deletes all tokens from the instance that are currently at the specified node.
func (e *Engine) removeTokensAt(inst *WorkflowInstance, nodeID string) {
	log.Printf("DEBUG: removing tokens at node %s", nodeID)
	var next []Token
	for _, t := range inst.Tokens {
		if t.NodeID != nodeID {
			next = append(next, t)
		}
	}
	inst.Tokens = next
}
