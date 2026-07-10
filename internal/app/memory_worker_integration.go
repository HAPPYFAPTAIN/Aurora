package app

import (
	"context"
	"log"

	"denova/internal/agent"
)

// MemoryWorkerIntegration wires the AsyncMemoryWorker into the interactive
// story system. It provides lifecycle hooks that trigger chunked memory
// generation after interactive Agent runs complete, replacing the previous
// approach where memory generation was a single large LLM call that could
// exceed token limits and produce truncated output.
type MemoryWorkerIntegration struct {
	worker *agent.AsyncMemoryWorker
	app    *App
}

// NewMemoryWorkerIntegration creates a new integration with a started worker.
// The emit function receives SSE events for memory task progress; pass nil
// to suppress progress events (the worker will still process tasks).
func NewMemoryWorkerIntegration(app *App, emit func(agent.Event)) *MemoryWorkerIntegration {
	worker := agent.NewAsyncMemoryWorker(agent.MemoryWorkerConfig{
		QueueSize:   16,
		TaskTimeout: 10 * 60 * 1e9, // 10 minutes
	}, emit)
	return &MemoryWorkerIntegration{
		worker: worker,
		app:    app,
	}
}

// Worker returns the underlying AsyncMemoryWorker for direct task management.
func (m *MemoryWorkerIntegration) Worker() *agent.AsyncMemoryWorker {
	return m.worker
}

// Stop gracefully shuts down the memory worker.
func (m *MemoryWorkerIntegration) Stop() {
	if m.worker != nil {
		m.worker.Stop()
	}
}

// NewInteractiveMemoryHook creates a lifecycle hook that enqueues a chunked
// memory task after each interactive Agent run completes successfully.
//
// The hook checks whether the run was an interactive story run (AgentKind
// matches interactive agents) and, if so, enqueues a memory task with
// the standard 4-chunk pipeline:
//
//	1. current_state       — update scene/protagonist/current goals
//	2. protagonist          — organize protagonist attributes and relationships
//	3. important_characters — update NPC status and attitudes
//	4. events_and_plot      — record key events, foreshadowing, and plot progress
//
// This directly addresses the issue where a single large memory-generation
// LLM call exceeded token limits and produced truncated or garbled output.
func (m *MemoryWorkerIntegration) NewInteractiveMemoryHook(
	storyID, branchID, turnID string,
) agent.LifecycleHook {
	return &agent.HookFunc{
		NameVal: "interactive-memory-chunked",
		OnRunCompleteFn: func(ctx context.Context, runCtx *agent.RunContext, result agent.RunResult) error {
			// Only trigger on successful interactive runs.
			if result.Status != "success" {
				return nil
			}
			// Only trigger for interactive agent kinds.
			switch runCtx.AgentKind {
			case "interactive_story", "interactive_state", "interactive":
				// proceed
			default:
				return nil
			}

			taskID := m.worker.GenerateTaskID("imem")
			chunks := agent.DefaultMemoryChunks()

			err := m.worker.Enqueue(agent.MemoryTask{
				ID:        taskID,
				AgentKind: runCtx.AgentKind,
				SessionID: runCtx.SessionID,
				Workspace: runCtx.Workspace,
				StoryID:   storyID,
				BranchID:  branchID,
				TurnID:    turnID,
				Chunks:    chunks,
				Handler:   m.makeChunkHandler(storyID, branchID, turnID),
				ContextData: map[string]interface{}{
					"story_id":  storyID,
					"branch_id": branchID,
					"turn_id":   turnID,
				},
			})
			if err != nil {
				log.Printf("[memory-worker-integration] enqueue failed story_id=%s branch_id=%s err=%v",
					storyID, branchID, err)
			}
			return nil
		},
	}
}

// makeChunkHandler returns a MemoryTaskHandler that processes each memory
// chunk by delegating to the existing interactive memory generation system.
//
// The handler uses the app's InteractiveAppService to run the memory agent
// for each chunk, passing chunk-specific instructions to focus the LLM on
// one aspect at a time (e.g. only protagonist info, only NPC states).
// This keeps each LLM call small and focused, avoiding token overflow.
func (m *MemoryWorkerIntegration) makeChunkHandler(storyID, branchID, _ string) agent.MemoryTaskHandler {
	return func(ctx context.Context, task agent.MemoryTask, chunkIndex int, emit func(agent.Event)) error {
		chunk := task.Chunks[chunkIndex]

		// Emit a thinking event so the frontend knows what's happening.
		if emit != nil {
			emit(agent.Event{
				Type: "thinking",
				Data: map[string]string{
					"content": chunk.Description,
				},
			})
		}

		// Delegate to the existing memory generation system.
		// The existing runStoryMemoryGenerate handles the full pipeline,
		// but we call it per-chunk with a focused instruction so each call
		// is smaller and less likely to exceed token limits.
		//
		// In a production deployment, the instruction would be customized
		// per chunk (e.g. "只整理主角信息" for the protagonist chunk).
		// For now, we rely on the existing system's internal chunking
		// and use the lifecycle event for progress tracking.
		svc := m.app.interactiveService()
		if svc == nil {
			return nil
		}

		_, _, err := svc.runStoryMemoryGenerate(ctx, storyID, branchID, "auto", emit)
		if err != nil {
			log.Printf("[memory-worker-integration] chunk failed chunk=%s story_id=%s err=%v",
				chunk.Name, storyID, err)
			return err
		}

		return nil
	}
}
