# Go Workflow Engine

A lightweight, decoupled Go workflow engine (state machine) designed for scalability and clear separation of concerns.

## Overview

The system is divided into three distinct layers:
1.  **Workflow Manager (WM)**: The pure state machine and blueprint registry. It manages tokens, evaluates gateways, updates global context, and tracks history.
2.  **Orchestrator (The Bridge)**: A thin layer that initializes the WM and Task workers, routing events between them.
3.  **Task Workers**: The execution environment that performs the actual work (API calls, DB writes, etc.).

## Getting Started

### 1. Initialize the Manager

The Manager requires a `Repository` to persist workflow instances. You can also configure a custom structured logger.

```go
import (
    "log/slog"
    "os"
    "github.com/lokewate/go-workflow"
)

repo := workflow.NewMemoryRepo() 

// Optional: Use a custom slog logger
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
wm := workflow.NewWorkflowManager(repo, workflow.WithLogger(logger))
```

### 2. Register a Task Handler (The Orchestrator)

The Orchestrator defines how the Manager interacts with your task execution system.

```go
wm.RegisterTaskHandler(func(ctx context.Context, payload workflow.TaskPayload) error {
    // The Manager provides a unique ExecutionID (format: instanceID:nodeID:uuid). 
    // Dispatch this task to your worker system (e.g., Temporal, Kafka, or simple Go routine).
    slog.Info("Executing task", "taskID", payload.TaskID, "execID", payload.ExecutionID)
    
    // Once the task is done, call wm.TaskDone(ctx, payload.ExecutionID, results)
    return nil
})
```

### 3. Loading Blueprints

Register your static workflow definitions (blueprints).

```go
blueprint := &workflow.Workflow{
    ID:   "user_onboarding",
    Name: "User Onboarding Flow",
    Nodes: []workflow.Node{
        {
            ID:   "start",
            Type: workflow.NodeTypeInternal,
            InternalType: workflow.InternalTypeEvent,
            EventType: workflow.StartEvent,
        },
        {
            ID:   "welcome_email",
            Type: workflow.NodeTypeTask,
            TaskID: "send_email",
            InputMapping: map[string]string{
                "email_address": "user_email", // Map GlobalContextKey to LocalTaskVar
            },
        },
    },
    Edges: []workflow.Edge{
        {ID: "e1", SourceID: "start", TargetID: "welcome_email"},
    },
}

wm.AddWorkflow(blueprint)
```

### 4. Running a Workflow

```go
initialCtx := map[string]any{
    "user_email": "hello@example.com",
}

// StartWorkflow returns the InstanceID
instanceID, err := wm.StartWorkflow(context.Background(), "user_onboarding", initialCtx)
```

### 5. Completing Tasks

When your task worker finishes its job, signal the Manager using the unique `ExecutionID`.

```go
results := map[string]any{
    "status": "sent",
}

err := wm.TaskDone(context.Background(), executionID, results)
```

## Workflow Statuses

- `StatusActive`: The workflow is currently running and has active tokens.
- `StatusCompleted`: The workflow has reached an `END` event.
- `StatusFailed`: The workflow encountered an error (e.g., no matching condition in an Exclusive Split) and has stopped.

## System Integrity Rules

- **Isolation**: Task workers operate only on provided inputs; they never touch `GlobalContext` directly.
- **Atomicity**: State updates are atomic and thread-safe.
- **Idempotency**: Duplicate signals for the same `ExecutionID` are ignored.

## License
MIT
