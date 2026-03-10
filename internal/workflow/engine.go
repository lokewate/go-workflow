package workflow

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
)

// Engine handles the execution and transitions of workflow instances based on a blueprint.
type Engine struct {
	Repo      InstanceRepository
	Blueprint *Blueprint
}

// getNode finds a node by its ID within the current blueprint.
func (e *Engine) getNode(id string) (Node, bool) {
	for _, n := range e.Blueprint.Nodes {
		if n.ID == id {
			return n, true
		}
	}
	return Node{}, false
}

// CompleteTask marks a task as finished, maps results to the payload, and triggers transitions.
func (e *Engine) CompleteTask(ctx context.Context, instID, nodeID string, results map[string]interface{}) error {
	log.Printf("[Engine] CompleteTask: instID=%s, nodeID=%s", instID, nodeID)
	inst, err := e.Repo.Get(ctx, instID)
	if err != nil {
		log.Printf("[Engine] CompleteTask: failed to get instance: %v", err)
		return err
	}

	node, ok := e.getNode(nodeID)
	if !ok {
		log.Printf("[Engine] CompleteTask: node %s not found in blueprint", nodeID)
		return fmt.Errorf("node %s not found", nodeID)
	}
	log.Printf("[Engine] CompleteTask: processing output mappings for node %s", nodeID)
	if inst.Payload == nil {
		inst.Payload = make(map[string]interface{})
	}
	for global, local := range node.Outputs {
		if val, ok := results[local]; ok {
			inst.Payload[global] = val
		}
	}

	e.removeTokensAt(inst, nodeID)
	err = e.transition(ctx, inst, nodeID)
	if err != nil {
		return err
	}

	return e.Repo.Save(ctx, inst)
}

// transition moves tokens from a source node to its targets based on gateway logic.
func (e *Engine) transition(ctx context.Context, inst *Instance, sourceID string) error {
	log.Printf("[Engine] transition: sourceID=%s", sourceID)
	sourceNode, ok := e.getNode(sourceID)
	if !ok {
		log.Printf("[Engine] transition: source node %s not found", sourceID)
		return nil
	}
	edges := e.getOutgoing(sourceID)
	log.Printf("[Engine] transition: found %d outgoing edges", len(edges))
	var targets []string

	if sourceNode.Type == NodeTypeGateway && sourceNode.GatewayType == ExclusiveSplit {
		for _, edge := range edges {
			if edge.Condition == nil || EvaluateCondition(*edge.Condition, inst.Payload) {
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
		if err := e.processTarget(ctx, inst, tID); err != nil {
			return err
		}
	}
	return nil
}

// processTarget determines how to handle a specific node reached during a transition.
func (e *Engine) processTarget(ctx context.Context, inst *Instance, nodeID string) error {
	log.Printf("[Engine] processTarget: nodeID=%s", nodeID)
	node, ok := e.getNode(nodeID)
	if !ok {
		log.Printf("[Engine] processTarget: node %s not found", nodeID)
		return nil
	}

	if node.Type == NodeTypeTask {
		inst.Tokens = append(inst.Tokens, Token{ID: uuid.NewString(), NodeID: nodeID, Status: TokenActive})
		return nil
	}

	if node.GatewayType == ParallelJoin {
		inst.Tokens = append(inst.Tokens, Token{ID: uuid.NewString(), NodeID: nodeID, Status: TokenWaiting})
		if len(e.getTokensAt(inst, nodeID)) >= len(e.getIncoming(nodeID)) {
			e.removeTokensAt(inst, nodeID)
			return e.transition(ctx, inst, nodeID)
		}
		return nil
	}

	return e.transition(ctx, inst, nodeID)
}

func (e *Engine) getOutgoing(id string) (res []Edge) {
	for _, edge := range e.Blueprint.Edges {
		if edge.SourceID == id {
			res = append(res, edge)
		}
	}
	return
}

func (e *Engine) getIncoming(id string) (res []Edge) {
	for _, edge := range e.Blueprint.Edges {
		if edge.TargetID == id {
			res = append(res, edge)
		}
	}
	return
}

func (e *Engine) getTokensAt(inst *Instance, id string) (res []Token) {
	for _, t := range inst.Tokens {
		if t.NodeID == id {
			res = append(res, t)
		}
	}
	return
}

func (e *Engine) removeTokensAt(inst *Instance, id string) {
	var next []Token
	for _, t := range inst.Tokens {
		if t.NodeID != id {
			next = append(next, t)
		}
	}
	inst.Tokens = next
}
