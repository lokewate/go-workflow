package engine

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/lokewate/go-workflow"
	"github.com/lokewate/go-workflow/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Helpers ---

// simpleLinearWorkflow: START -> task_a -> END
func simpleLinearWorkflow() *workflow.Workflow {
	return &workflow.Workflow{
		ID:      "linear",
		Name:    "Simple Linear",
		Version: 1,
		Nodes: []workflow.Node{
			{ID: "start", Type: workflow.NodeTypeInternal, InternalType: workflow.InternalTypeEvent, EventType: workflow.StartEvent},
			{ID: "task_a", Type: workflow.NodeTypeTask, TaskID: "do_a",
				InputMapping:  map[string]string{"local_in": "global_input"},
				OutputMapping: map[string]string{"local_out": "global_output"},
			},
			{ID: "end", Type: workflow.NodeTypeInternal, InternalType: workflow.InternalTypeEvent, EventType: workflow.EndEvent},
		},
		Edges: []workflow.Edge{
			{ID: "e1", SourceID: "start", TargetID: "task_a"},
			{ID: "e2", SourceID: "task_a", TargetID: "end"},
		},
	}
}

// exclusiveSplitWorkflow: START -> task_init -> EXCLUSIVE_SPLIT -> (ok: task_yes | !ok: task_no) -> END
func exclusiveSplitWorkflow() *workflow.Workflow {
	condOK := "ok == true"
	condNotOK := "ok == false"
	return &workflow.Workflow{
		ID:      "exclusive",
		Name:    "Exclusive Split",
		Version: 1,
		Nodes: []workflow.Node{
			{ID: "start", Type: workflow.NodeTypeInternal, InternalType: workflow.InternalTypeEvent, EventType: workflow.StartEvent},
			{ID: "task_init", Type: workflow.NodeTypeTask, TaskID: "init",
				OutputMapping: map[string]string{"ok": "ok"},
			},
			{ID: "split", Type: workflow.NodeTypeInternal, InternalType: workflow.InternalTypeGateway, GatewayType: workflow.ExclusiveSplit},
			{ID: "task_yes", Type: workflow.NodeTypeTask, TaskID: "yes_task"},
			{ID: "task_no", Type: workflow.NodeTypeTask, TaskID: "no_task"},
			{ID: "join", Type: workflow.NodeTypeInternal, InternalType: workflow.InternalTypeGateway, GatewayType: workflow.ExclusiveJoin},
			{ID: "end", Type: workflow.NodeTypeInternal, InternalType: workflow.InternalTypeEvent, EventType: workflow.EndEvent},
		},
		Edges: []workflow.Edge{
			{ID: "e1", SourceID: "start", TargetID: "task_init"},
			{ID: "e2", SourceID: "task_init", TargetID: "split"},
			{ID: "e3", SourceID: "split", TargetID: "task_yes", Condition: &condOK},
			{ID: "e4", SourceID: "split", TargetID: "task_no", Condition: &condNotOK},
			{ID: "e5", SourceID: "task_yes", TargetID: "join"},
			{ID: "e6", SourceID: "task_no", TargetID: "join"},
			{ID: "e7", SourceID: "join", TargetID: "end"},
		},
	}
}

// parallelWorkflow: START -> PARALLEL_SPLIT -> (task_a, task_b) -> PARALLEL_JOIN -> END
func parallelWorkflow() *workflow.Workflow {
	return &workflow.Workflow{
		ID:      "parallel",
		Name:    "Parallel Flow",
		Version: 1,
		Nodes: []workflow.Node{
			{ID: "start", Type: workflow.NodeTypeInternal, InternalType: workflow.InternalTypeEvent, EventType: workflow.StartEvent},
			{ID: "p_split", Type: workflow.NodeTypeInternal, InternalType: workflow.InternalTypeGateway, GatewayType: workflow.ParallelSplit},
			{ID: "task_a", Type: workflow.NodeTypeTask, TaskID: "work_a"},
			{ID: "task_b", Type: workflow.NodeTypeTask, TaskID: "work_b"},
			{ID: "p_join", Type: workflow.NodeTypeInternal, InternalType: workflow.InternalTypeGateway, GatewayType: workflow.ParallelJoin},
			{ID: "end", Type: workflow.NodeTypeInternal, InternalType: workflow.InternalTypeEvent, EventType: workflow.EndEvent},
		},
		Edges: []workflow.Edge{
			{ID: "e1", SourceID: "start", TargetID: "p_split"},
			{ID: "e2", SourceID: "p_split", TargetID: "task_a"},
			{ID: "e3", SourceID: "p_split", TargetID: "task_b"},
			{ID: "e4", SourceID: "task_a", TargetID: "p_join"},
			{ID: "e5", SourceID: "task_b", TargetID: "p_join"},
			{ID: "e6", SourceID: "p_join", TargetID: "end"},
		},
	}
}

// newTestManager creates a manager with a handler that captures activations.
func newTestManager() (workflow.Manager, *activationTracker) {
	repo := repository.NewMemoryRepo()
	mgr := NewWorkflowManager(repo)

	tracker := &activationTracker{
		payloads: make(map[string][]workflow.TaskPayload),
	}

	mgr.RegisterTaskHandler(func(ctx context.Context, payload workflow.TaskPayload) error {
		tracker.mu.Lock()
		defer tracker.mu.Unlock()
		tracker.payloads[payload.NodeID()] = append(tracker.payloads[payload.NodeID()], payload)
		return nil
	})

	return mgr, tracker
}

func wfJSON(t *testing.T, wf *workflow.Workflow) []byte {
	b, err := json.Marshal(wf)
	require.NoError(t, err)
	return b
}

type activationTracker struct {
	mu       sync.Mutex
	payloads map[string][]workflow.TaskPayload // NodeID -> []workflow.TaskPayload
}

func (t *activationTracker) getExecID(nodeID string, index int) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	payloads := t.payloads[nodeID]
	if index < len(payloads) {
		return payloads[index].ExecutionID
	}
	return ""
}

func (t *activationTracker) getPayload(nodeID string, index int) workflow.TaskPayload {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.payloads[nodeID][index]
}

// ============================================================
// Error Path Tests
// ============================================================

func TestStartWorkflow_ErrInvalidJSON(t *testing.T) {
	repo := repository.NewMemoryRepo()
	mgr := NewWorkflowManager(repo)

	_, err := mgr.StartWorkflow(context.Background(), []byte("invalid-json"), nil)
	require.Error(t, err)
}

func TestStartWorkflow_ErrNoStartEvent(t *testing.T) {
	repo := repository.NewMemoryRepo()
	mgr := NewWorkflowManager(repo)

	// Workflow with no START event
	wf := &workflow.Workflow{
		ID:   "no_start",
		Name: "Missing Start",
		Nodes: []workflow.Node{
			{ID: "task_a", Type: workflow.NodeTypeTask, TaskID: "a"},
			{ID: "end", Type: workflow.NodeTypeInternal, InternalType: workflow.InternalTypeEvent, EventType: workflow.EndEvent},
		},
		Edges: []workflow.Edge{
			{ID: "e1", SourceID: "task_a", TargetID: "end"},
		},
	}
	_, err := mgr.StartWorkflow(context.Background(), wfJSON(t, wf), nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, workflow.ErrNoStartEvent))
}

func TestStartWorkflow_ErrHandlerNotRegistered(t *testing.T) {
	repo := repository.NewMemoryRepo()
	mgr := NewWorkflowManager(repo)

	// Don't register a handler — StartWorkflow should fail when it hits the task node
	_, err := mgr.StartWorkflow(context.Background(), wfJSON(t, simpleLinearWorkflow()), nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, workflow.ErrHandlerNotRegistered))
}

func TestTaskDone_ErrInvalidExecutionID(t *testing.T) {
	mgr, _ := newTestManager()

	err := mgr.TaskDone(context.Background(), "bad-format-no-colon", nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, workflow.ErrInvalidExecutionID))
}

func TestTaskDone_ErrInstanceNotFound(t *testing.T) {
	mgr, _ := newTestManager()

	err := mgr.TaskDone(context.Background(), "nonexistent-instance:some-node:some-uuid", nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, workflow.ErrInstanceNotFound))
}

func TestGetStatus_ErrInstanceNotFound(t *testing.T) {
	mgr, _ := newTestManager()

	inst, err := mgr.GetStatus(context.Background(), "does-not-exist")
	require.Error(t, err)
	assert.Nil(t, inst)
	assert.True(t, errors.Is(err, workflow.ErrInstanceNotFound))
}

func TestExclusiveSplit_ErrNoMatchingCondition(t *testing.T) {
	ctx := context.Background()

	// Create a workflow where the exclusive split conditions won't match
	condA := "status == 'approved'"
	condB := "status == 'rejected'"
	wf := &workflow.Workflow{
		ID:   "no_match",
		Name: "No Match",
		Nodes: []workflow.Node{
			{ID: "start", Type: workflow.NodeTypeInternal, InternalType: workflow.InternalTypeEvent, EventType: workflow.StartEvent},
			{ID: "task_init", Type: workflow.NodeTypeTask, TaskID: "init",
				OutputMapping: map[string]string{"status": "status"},
			},
			{ID: "split", Type: workflow.NodeTypeInternal, InternalType: workflow.InternalTypeGateway, GatewayType: workflow.ExclusiveSplit},
			{ID: "task_a", Type: workflow.NodeTypeTask, TaskID: "a"},
			{ID: "task_b", Type: workflow.NodeTypeTask, TaskID: "b"},
		},
		Edges: []workflow.Edge{
			{ID: "e1", SourceID: "start", TargetID: "task_init"},
			{ID: "e2", SourceID: "task_init", TargetID: "split"},
			{ID: "e3", SourceID: "split", TargetID: "task_a", Condition: &condA},
			{ID: "e4", SourceID: "split", TargetID: "task_b", Condition: &condB},
		},
	}

	mgr, tracker := newTestManager()
	instID, err := mgr.StartWorkflow(ctx, wfJSON(t, wf), nil)
	require.NoError(t, err)

	// Complete task_init with status="pending" — neither condition matches
	execID := tracker.getExecID("task_init", 0)
	err = mgr.TaskDone(ctx, execID, map[string]any{"status": "pending"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, workflow.ErrNoMatchingCondition))

	// Verify instance is marked as FAILED
	inst, err := mgr.GetStatus(ctx, instID)
	require.NoError(t, err)
	assert.Equal(t, workflow.StatusFailed, inst.Status)
}

// ============================================================
// Behavioral Tests
// ============================================================

func TestIdempotency_DuplicateTaskDone(t *testing.T) {
	ctx := context.Background()
	mgr, tracker := newTestManager()

	_, err := mgr.StartWorkflow(ctx, wfJSON(t, simpleLinearWorkflow()), map[string]any{"global_input": "hello"})
	require.NoError(t, err)

	execID := tracker.getExecID("task_a", 0)
	require.NotEmpty(t, execID)

	// First call should succeed
	err = mgr.TaskDone(ctx, execID, map[string]any{"local_out": "result"})
	require.NoError(t, err)

	// Second call with the same executionID should be a no-op (not an error)
	err = mgr.TaskDone(ctx, execID, map[string]any{"local_out": "result"})
	require.NoError(t, err, "Duplicate TaskDone should not return an error")
}

func TestInputMapping(t *testing.T) {
	ctx := context.Background()
	wf := simpleLinearWorkflow()
	repo := repository.NewMemoryRepo()
	mgr := NewWorkflowManager(repo)

	var capturedPayload workflow.TaskPayload
	mgr.RegisterTaskHandler(func(ctx context.Context, payload workflow.TaskPayload) error {
		capturedPayload = payload
		return nil
	})

	_, err := mgr.StartWorkflow(ctx, wfJSON(t, wf), map[string]any{
		"global_input": "mapped_value",
	})
	require.NoError(t, err)

	// Verify the handler received the correctly mapped input
	assert.Equal(t, "do_a", capturedPayload.TaskID)
	assert.Equal(t, "mapped_value", capturedPayload.Inputs["local_in"],
		"Input mapping should map global_input -> local_in")
}

func TestOutputMapping(t *testing.T) {
	ctx := context.Background()
	mgr, tracker := newTestManager()

	instID, err := mgr.StartWorkflow(ctx, wfJSON(t, simpleLinearWorkflow()), map[string]any{
		"global_input": "hello",
	})
	require.NoError(t, err)

	execID := tracker.getExecID("task_a", 0)
	err = mgr.TaskDone(ctx, execID, map[string]any{
		"local_out": "task_result_value",
	})
	require.NoError(t, err)

	// Verify the output was mapped into global context
	inst, err := mgr.GetStatus(ctx, instID)
	require.NoError(t, err)
	assert.Equal(t, "task_result_value", inst.Context.Get("global_output"),
		"Output mapping should map local_out -> global_output")
	assert.Equal(t, workflow.StatusCompleted, inst.Status)
}

func TestOutputMapping_IgnoresUnmappedKeys(t *testing.T) {
	ctx := context.Background()
	mgr, tracker := newTestManager()

	instID, err := mgr.StartWorkflow(ctx, wfJSON(t, simpleLinearWorkflow()), map[string]any{
		"global_input": "hello",
	})
	require.NoError(t, err)

	execID := tracker.getExecID("task_a", 0)
	err = mgr.TaskDone(ctx, execID, map[string]any{
		"local_out":   "mapped_value",
		"extra_field": "should_not_appear", // Not in OutputMapping
	})
	require.NoError(t, err)

	inst, err := mgr.GetStatus(ctx, instID)
	require.NoError(t, err)
	assert.Equal(t, "mapped_value", inst.Context.Get("global_output"))
	assert.Nil(t, inst.Context.Get("extra_field"),
		"Unmapped output keys should not leak into global context")
}

func TestConcurrentTaskDone_SameInstance(t *testing.T) {
	ctx := context.Background()
	mgr, tracker := newTestManager()

	instID, err := mgr.StartWorkflow(ctx, wfJSON(t, parallelWorkflow()), nil)
	require.NoError(t, err)

	execA := tracker.getExecID("task_a", 0)
	execB := tracker.getExecID("task_b", 0)
	require.NotEmpty(t, execA)
	require.NotEmpty(t, execB)

	// Complete both tasks concurrently
	var wg sync.WaitGroup
	var errA, errB error

	wg.Add(2)
	go func() {
		defer wg.Done()
		errA = mgr.TaskDone(ctx, execA, nil)
	}()
	go func() {
		defer wg.Done()
		errB = mgr.TaskDone(ctx, execB, nil)
	}()
	wg.Wait()

	assert.NoError(t, errA, "Concurrent TaskDone A should not error")
	assert.NoError(t, errB, "Concurrent TaskDone B should not error")

	inst, err := mgr.GetStatus(ctx, instID)
	require.NoError(t, err)
	assert.Equal(t, workflow.StatusCompleted, inst.Status,
		"Workflow should complete after both parallel tasks finish")
	assert.Empty(t, inst.Context.GetTokens(),
		"No tokens should remain after completion")
}

func TestConcurrentTaskDone_DifferentInstances(t *testing.T) {
	ctx := context.Background()
	wf := simpleLinearWorkflow()
	repo := repository.NewMemoryRepo()
	mgr := NewWorkflowManager(repo)

	var mu sync.Mutex
	activations := make(map[string]string) // instanceID -> executionID (since each only has one task)

	mgr.RegisterTaskHandler(func(ctx context.Context, payload workflow.TaskPayload) error {
		mu.Lock()
		defer mu.Unlock()
		// Store by the instance portion of the executionID
		activations[payload.ExecutionID] = payload.ExecutionID
		return nil
	})

	// Start multiple instances
	const numInstances = 20
	instIDs := make([]string, numInstances)
	execIDs := make([]string, numInstances)

	wfBytes := wfJSON(t, wf)
	for i := 0; i < numInstances; i++ {
		id, err := mgr.StartWorkflow(ctx, wfBytes, map[string]any{
			"global_input": i,
		})
		require.NoError(t, err)
		instIDs[i] = id
	}

	// Collect execIDs
	mu.Lock()
	i := 0
	for _, eid := range activations {
		execIDs[i] = eid
		i++
	}
	mu.Unlock()

	// Complete all tasks concurrently
	var wg sync.WaitGroup
	var errorCount int32

	for _, eid := range execIDs[:numInstances] {
		wg.Add(1)
		go func(execID string) {
			defer wg.Done()
			if err := mgr.TaskDone(ctx, execID, map[string]any{"local_out": "done"}); err != nil {
				atomic.AddInt32(&errorCount, 1)
			}
		}(eid)
	}
	wg.Wait()

	assert.Equal(t, int32(0), errorCount, "No errors expected across concurrent instance completions")

	// All should be completed
	for _, id := range instIDs {
		inst, err := mgr.GetStatus(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, workflow.StatusCompleted, inst.Status,
			"Instance %s should be completed", id)
	}
}

// ============================================================
// Edge Case Tests
// ============================================================

func TestStartWorkflow_NilInitialContext(t *testing.T) {
	ctx := context.Background()
	mgr, tracker := newTestManager()

	instID, err := mgr.StartWorkflow(ctx, wfJSON(t, simpleLinearWorkflow()), nil)
	require.NoError(t, err)

	// Should still activate the task, just with nil/empty inputs
	execID := tracker.getExecID("task_a", 0)
	require.NotEmpty(t, execID)

	// Complete it
	err = mgr.TaskDone(ctx, execID, map[string]any{"local_out": "val"})
	require.NoError(t, err)

	inst, err := mgr.GetStatus(ctx, instID)
	require.NoError(t, err)
	assert.Equal(t, workflow.StatusCompleted, inst.Status)
}

func TestTaskDone_EmptyOutputs(t *testing.T) {
	ctx := context.Background()
	mgr, tracker := newTestManager()

	instID, err := mgr.StartWorkflow(ctx, wfJSON(t, simpleLinearWorkflow()), map[string]any{"global_input": "v"})
	require.NoError(t, err)

	execID := tracker.getExecID("task_a", 0)
	err = mgr.TaskDone(ctx, execID, nil) // nil outputs
	require.NoError(t, err)

	inst, err := mgr.GetStatus(ctx, instID)
	require.NoError(t, err)
	assert.Equal(t, workflow.StatusCompleted, inst.Status,
		"Workflow should complete even with nil outputs")
	assert.Nil(t, inst.Context.Get("global_output"),
		"Output should not be set when outputs map is nil")
}

func TestTaskDone_EmptyOutputsMap(t *testing.T) {
	ctx := context.Background()
	mgr, tracker := newTestManager()

	instID, err := mgr.StartWorkflow(ctx, wfJSON(t, simpleLinearWorkflow()), map[string]any{"global_input": "v"})
	require.NoError(t, err)

	execID := tracker.getExecID("task_a", 0)
	err = mgr.TaskDone(ctx, execID, map[string]any{}) // empty map
	require.NoError(t, err)

	inst, err := mgr.GetStatus(ctx, instID)
	require.NoError(t, err)
	assert.Equal(t, workflow.StatusCompleted, inst.Status)
}

func TestExclusiveSplit_DefaultEdge(t *testing.T) {
	ctx := context.Background()

	// Workflow where one edge has no condition (acts as default)
	condA := "status == 'approved'"
	wf := &workflow.Workflow{
		ID:   "default_edge",
		Name: "Default Edge",
		Nodes: []workflow.Node{
			{ID: "start", Type: workflow.NodeTypeInternal, InternalType: workflow.InternalTypeEvent, EventType: workflow.StartEvent},
			{ID: "split", Type: workflow.NodeTypeInternal, InternalType: workflow.InternalTypeGateway, GatewayType: workflow.ExclusiveSplit},
			{ID: "task_a", Type: workflow.NodeTypeTask, TaskID: "approved_task"},
			{ID: "task_default", Type: workflow.NodeTypeTask, TaskID: "default_task"},
			{ID: "end", Type: workflow.NodeTypeInternal, InternalType: workflow.InternalTypeEvent, EventType: workflow.EndEvent},
		},
		Edges: []workflow.Edge{
			{ID: "e1", SourceID: "start", TargetID: "split"},
			{ID: "e2", SourceID: "split", TargetID: "task_a", Condition: &condA},
			{ID: "e3", SourceID: "split", TargetID: "task_default"}, // No condition = default
			{ID: "e4", SourceID: "task_a", TargetID: "end"},
			{ID: "e5", SourceID: "task_default", TargetID: "end"},
		},
	}

	mgr, tracker := newTestManager()

	// Start — no matching condition, but default edge should fire
	instID, err := mgr.StartWorkflow(ctx, wfJSON(t, wf), map[string]any{
		"status": "pending",
	})
	require.NoError(t, err)

	// Should have activated task_a (condition match) or task_default.
	// Since "pending" != "approved", e2 fails, then e3 (no condition) fires as default
	execID := tracker.getExecID("task_default", 0)
	require.NotEmpty(t, execID, "Default edge should fire when no condition matches")

	err = mgr.TaskDone(ctx, execID, nil)
	require.NoError(t, err)

	inst, err := mgr.GetStatus(ctx, instID)
	require.NoError(t, err)
	assert.Equal(t, workflow.StatusCompleted, inst.Status)
}

func TestMultipleBlueprints(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemoryRepo()
	mgr := NewWorkflowManager(repo)

	wfA := simpleLinearWorkflow()
	wfB := parallelWorkflow()

	var mu sync.Mutex
	taskIDs := make(map[string]int) // TaskID -> count

	mgr.RegisterTaskHandler(func(ctx context.Context, payload workflow.TaskPayload) error {
		mu.Lock()
		defer mu.Unlock()
		taskIDs[payload.TaskID]++
		return nil
	})

	// Start both workflows
	_, err := mgr.StartWorkflow(ctx, wfJSON(t, wfA), map[string]any{"global_input": "v"})
	require.NoError(t, err)

	_, err = mgr.StartWorkflow(ctx, wfJSON(t, wfB), nil)
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()

	// Linear activates TaskID "do_a"
	assert.Equal(t, 1, taskIDs["do_a"], "Linear workflow should activate do_a once")
	// Parallel activates TaskIDs "work_a" and "work_b"
	assert.Equal(t, 1, taskIDs["work_a"], "Parallel workflow should activate work_a once")
	assert.Equal(t, 1, taskIDs["work_b"], "Parallel workflow should activate work_b once")
}

func TestStatusFailed_PersistedToRepo(t *testing.T) {
	ctx := context.Background()

	// Create workflow that will always fail at the split (no default, no matching condition)
	condA := "x == 1"
	condB := "x == 2"
	wf := &workflow.Workflow{
		ID:   "will_fail",
		Name: "Will Fail",
		Nodes: []workflow.Node{
			{ID: "start", Type: workflow.NodeTypeInternal, InternalType: workflow.InternalTypeEvent, EventType: workflow.StartEvent},
			{ID: "task_init", Type: workflow.NodeTypeTask, TaskID: "init",
				OutputMapping: map[string]string{"x": "x"},
			},
			{ID: "split", Type: workflow.NodeTypeInternal, InternalType: workflow.InternalTypeGateway, GatewayType: workflow.ExclusiveSplit},
			{ID: "task_a", Type: workflow.NodeTypeTask, TaskID: "a"},
			{ID: "task_b", Type: workflow.NodeTypeTask, TaskID: "b"},
		},
		Edges: []workflow.Edge{
			{ID: "e1", SourceID: "start", TargetID: "task_init"},
			{ID: "e2", SourceID: "task_init", TargetID: "split"},
			{ID: "e3", SourceID: "split", TargetID: "task_a", Condition: &condA},
			{ID: "e4", SourceID: "split", TargetID: "task_b", Condition: &condB},
		},
	}

	mgr, tracker := newTestManager()
	instID, err := mgr.StartWorkflow(ctx, wfJSON(t, wf), nil)
	require.NoError(t, err)

	execID := tracker.getExecID("task_init", 0)
	err = mgr.TaskDone(ctx, execID, map[string]any{"x": 999}) // Neither 1 nor 2
	require.Error(t, err)

	// Verify FAILED status is persisted and retrievable
	inst, err := mgr.GetStatus(ctx, instID)
	require.NoError(t, err)
	assert.Equal(t, workflow.StatusFailed, inst.Status,
		"Instance should be persisted as FAILED after transition error")
}

func TestTaskPayload_NodeID(t *testing.T) {
	tests := []struct {
		name     string
		execID   string
		expected string
	}{
		{"normal format", "inst-123:node-abc:uuid-456", "node-abc"},
		{"two parts", "inst:node", "node"},
		{"single part", "invalid", ""},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := workflow.TaskPayload{ExecutionID: tt.execID}
			assert.Equal(t, tt.expected, p.NodeID())
		})
	}
}

func TestLinearWorkflow_FullLifecycle(t *testing.T) {
	ctx := context.Background()
	mgr, tracker := newTestManager()

	// Start
	instID, err := mgr.StartWorkflow(ctx, wfJSON(t, simpleLinearWorkflow()), map[string]any{
		"global_input": "test_value",
	})
	require.NoError(t, err)

	// Verify ACTIVE status
	inst, err := mgr.GetStatus(ctx, instID)
	require.NoError(t, err)
	assert.Equal(t, workflow.StatusActive, inst.Status)
	assert.Len(t, inst.Context.GetTokens(), 1, "Should have one token at task_a")

	// Verify input was correctly mapped
	payload := tracker.getPayload("task_a", 0)
	assert.Equal(t, "test_value", payload.Inputs["local_in"])
	assert.Equal(t, "do_a", payload.TaskID)

	// Complete the task
	err = mgr.TaskDone(ctx, payload.ExecutionID, map[string]any{
		"local_out": "output_value",
	})
	require.NoError(t, err)

	// Verify COMPLETED
	inst, err = mgr.GetStatus(ctx, instID)
	require.NoError(t, err)
	assert.Equal(t, workflow.StatusCompleted, inst.Status)
	assert.Empty(t, inst.Context.GetTokens())
	assert.Equal(t, "output_value", inst.Context.Get("global_output"))
}

func TestWorkflowCompletionHandler(t *testing.T) {
	ctx := context.Background()
	mgr, tracker := newTestManager()

	var completed atomic.Bool
	var finalStatus workflow.WorkflowStatus

	mgr.RegisterWorkflowCompletionHandler(func(ctx context.Context, inst *workflow.WorkflowInstance) error {
		completed.Store(true)
		finalStatus = inst.Status
		return nil
	})

	// 1. Success Case
	instID, err := mgr.StartWorkflow(ctx, wfJSON(t, simpleLinearWorkflow()), nil)
	require.NoError(t, err)
	require.NotEmpty(t, instID)

	execID := tracker.getExecID("task_a", 0)
	err = mgr.TaskDone(ctx, execID, nil)
	require.NoError(t, err)

	assert.True(t, completed.Load())
	assert.Equal(t, workflow.StatusCompleted, finalStatus)

	// 2. Failure Case
	completed.Store(false)
	// Create workflow that fails (no start event is easiest to fail during StartWorkflow)
	wfFail := &workflow.Workflow{ID: "fail", Nodes: []workflow.Node{{ID: "task", Type: workflow.NodeTypeTask}}} // No start event
	_, err = mgr.StartWorkflow(ctx, wfJSON(t, wfFail), nil)
	require.Error(t, err)

	assert.True(t, completed.Load())
	assert.Equal(t, workflow.StatusFailed, finalStatus)
}
