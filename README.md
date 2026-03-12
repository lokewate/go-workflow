# Go Workflow Engine

A lightweight, decoupled Go workflow engine (state machine) designed for scalability and clear separation of concerns.

## Overview

The system is divided into three distinct layers:
1.  **Workflow Manager (WM)**: The pure state machine and blueprint registry. It manages tokens, evaluates gateways, updates global context, and tracks history.
2.  **Orchestrator (The Bridge)**: A thin layer that initializes the WM and Task workers, routing events between them.
3.  **Task Workers**: The execution environment that performs the actual work (API calls, DB writes, etc.).

## Getting Started

### 1. Initialize the Manager

The Manager requires a `Repository` to persist workflow instances.

```go
import "github.com/your-org/go-workflow"

repo := workflow.NewMemoryRepo() // Or your custom DB implementation
wm := workflow.NewWorkflowManager(repo)
```

### 2. Register a Task Handler (The Orchestrator)

The Orchestrator defines how the Manager interacts with your task execution system.

```go
wm.RegisterTaskHandler(func(ctx context.Context, payload workflow.TaskPayload) error {
    // The Manager provides a unique ExecutionID. 
    // Dispatch this task to your worker system (e.g., Temporal, Kafka, or simple Go routine).
    log.Printf("Executing task %s with inputs %v", payload.TaskID, payload.Inputs)
    
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
                "email_address": "user_email",
            },
        },
        // ... more nodes
    },
    // ... edges
}

wm.AddWorkflow(blueprint)
```

### 4. Running a Workflow

```go
initialCtx := map[string]any{
    "user_email": "hello@example.com",
}

instanceID, err := wm.StartWorkflow(ctx, "user_onboarding", initialCtx)
```

### 5. Completing Tasks

When your task worker finishes its job, signal the Manager using the unique `ExecutionID`.

```go
results := map[string]any{
    "status": "sent",
}

err := wm.TaskDone(ctx, executionID, results)
```

## System Integrity Rules

- **Isolation**: Task workers operate only on provided inputs; they never touch `GlobalContext` directly.
- **Atomicity**: State updates are atomic and thread-safe.
- **Idempotency**: Duplicate signals for the same `ExecutionID` are ignored.

## License
MIT
