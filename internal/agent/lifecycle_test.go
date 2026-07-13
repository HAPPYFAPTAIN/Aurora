package agent

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestHookRegistryFireOrder(t *testing.T) {
	registry := &HookRegistry{}
	var mu sync.Mutex
	var order []string

	// Register hooks with different priorities.
	registry.Register(&HookFunc{
		NameVal: "low",
		OnRunStartFn: func(_ context.Context, _ *RunContext) error {
			mu.Lock()
			order = append(order, "low")
			mu.Unlock()
			return nil
		},
	}, HookPriorityLow)

	registry.Register(&HookFunc{
		NameVal: "high",
		OnRunStartFn: func(_ context.Context, _ *RunContext) error {
			mu.Lock()
			order = append(order, "high")
			mu.Unlock()
			return nil
		},
	}, HookPriorityHigh)

	registry.Register(&HookFunc{
		NameVal: "default",
		OnRunStartFn: func(_ context.Context, _ *RunContext) error {
			mu.Lock()
			order = append(order, "default")
			mu.Unlock()
			return nil
		},
	}, DefaultHookPriority)

	runCtx := &RunContext{RunID: "test-run"}
	_ = registry.fireRunStart(context.Background(), runCtx)

	if len(order) != 3 {
		t.Fatalf("expected 3 hook calls, got %d", len(order))
	}
	if order[0] != "high" || order[1] != "default" || order[2] != "low" {
		t.Fatalf("expected [high, default, low], got %v", order)
	}
}

func TestHookRegistryAllPhases(t *testing.T) {
	registry := &HookRegistry{}
	var phases []string
	var mu sync.Mutex

	hook := &HookFunc{
		NameVal: "test-all-phases",
		OnRunStartFn: func(_ context.Context, _ *RunContext) error {
			mu.Lock()
			phases = append(phases, "run_start")
			mu.Unlock()
			return nil
		},
		OnModelCallFn: func(_ context.Context, _ *RunContext, _ ModelCallInfo) error {
			mu.Lock()
			phases = append(phases, "model_call")
			mu.Unlock()
			return nil
		},
		OnToolResultFn: func(_ context.Context, _ *RunContext, _ ToolResultInfo) error {
			mu.Lock()
			phases = append(phases, "tool_result")
			mu.Unlock()
			return nil
		},
		OnRunCompleteFn: func(_ context.Context, _ *RunContext, _ RunResult) error {
			mu.Lock()
			phases = append(phases, "run_complete")
			mu.Unlock()
			return nil
		},
	}
	registry.Register(hook, DefaultHookPriority)

	runCtx := &RunContext{RunID: "test-run"}

	_ = registry.fireRunStart(context.Background(), runCtx)
	_ = registry.fireModelCall(context.Background(), runCtx, ModelCallInfo{})
	_ = registry.fireToolResult(context.Background(), runCtx, ToolResultInfo{})
	_ = registry.fireRunComplete(context.Background(), runCtx, RunResult{Status: "success"})

	expected := []string{"run_start", "model_call", "tool_result", "run_complete"}
	if len(phases) != len(expected) {
		t.Fatalf("expected %d phases, got %d: %v", len(expected), len(phases), phases)
	}
	for i, want := range expected {
		if phases[i] != want {
			t.Errorf("phase[%d] = %q, want %q", i, phases[i], want)
		}
	}
}

func TestHookRegistryEmpty(t *testing.T) {
	registry := &HookRegistry{}
	runCtx := &RunContext{RunID: "test"}

	// Should not panic with no registered hooks.
	errs := registry.fireRunStart(context.Background(), runCtx)
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d", len(errs))
	}
	errs = registry.fireRunComplete(context.Background(), runCtx, RunResult{})
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d", len(errs))
	}
}

func TestAsyncMemoryWorkerEnqueueAndProcess(t *testing.T) {
	var mu sync.Mutex
	var processedChunks []string

	worker := NewAsyncMemoryWorker(MemoryWorkerConfig{
		QueueSize:   4,
		TaskTimeout: 30 * time.Second,
	}, nil)
	defer worker.Stop()

	taskID := worker.GenerateTaskID("test")
	chunks := []MemoryTaskChunk{
		{Name: "step1", Description: "first"},
		{Name: "step2", Description: "second"},
	}

	err := worker.Enqueue(MemoryTask{
		ID:        taskID,
		AgentKind: "test",
		Chunks:    chunks,
		Handler: func(_ context.Context, _ MemoryTask, chunkIndex int, _ func(Event)) error {
			mu.Lock()
			processedChunks = append(processedChunks, chunks[chunkIndex].Name)
			mu.Unlock()
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	// Wait for task to complete.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		info := worker.TaskInfo(taskID)
		if info != nil && (info.Status == MemoryTaskCompleted || info.Status == MemoryTaskFailed) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	info := worker.TaskInfo(taskID)
	if info == nil {
		t.Fatal("task info not found")
	}
	if info.Status != MemoryTaskCompleted {
		t.Fatalf("expected completed, got %s (error: %s)", info.Status, info.Error)
	}
	if info.DoneChunks != 2 {
		t.Fatalf("expected 2 done chunks, got %d", info.DoneChunks)
	}
	if len(processedChunks) != 2 || processedChunks[0] != "step1" || processedChunks[1] != "step2" {
		t.Fatalf("expected [step1, step2], got %v", processedChunks)
	}
}

func TestAsyncMemoryWorkerChunkFailure(t *testing.T) {
	worker := NewAsyncMemoryWorker(MemoryWorkerConfig{
		QueueSize:   4,
		TaskTimeout: 30 * time.Second,
	}, nil)
	defer worker.Stop()

	taskID := worker.GenerateTaskID("test-fail")
	chunks := []MemoryTaskChunk{
		{Name: "ok", Description: "succeeds"},
		{Name: "fail", Description: "fails"},
		{Name: "skipped", Description: "should be skipped"},
	}

	err := worker.Enqueue(MemoryTask{
		ID:        taskID,
		AgentKind: "test",
		Chunks:    chunks,
		Handler: func(_ context.Context, _ MemoryTask, chunkIndex int, _ func(Event)) error {
			if chunkIndex == 1 {
				return context.DeadlineExceeded
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		info := worker.TaskInfo(taskID)
		if info != nil && (info.Status == MemoryTaskCompleted || info.Status == MemoryTaskFailed) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	info := worker.TaskInfo(taskID)
	if info == nil {
		t.Fatal("task info not found")
	}
	if info.Status != MemoryTaskFailed {
		t.Fatalf("expected failed, got %s", info.Status)
	}
	if info.DoneChunks != 1 {
		t.Fatalf("expected 1 done chunk (first succeeded), got %d", info.DoneChunks)
	}
}

func TestDefaultMemoryChunks(t *testing.T) {
	chunks := DefaultMemoryChunks()
	if len(chunks) != 4 {
		t.Fatalf("expected 4 default chunks, got %d", len(chunks))
	}
	expected := []string{"current_state", "protagonist", "important_characters", "events_and_plot"}
	for i, want := range expected {
		if chunks[i].Name != want {
			t.Errorf("chunk[%d].Name = %q, want %q", i, chunks[i].Name, want)
		}
	}
}
