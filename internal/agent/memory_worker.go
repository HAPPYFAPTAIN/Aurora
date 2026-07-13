package agent

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// MemoryTaskStatus tracks the lifecycle of a memory task.
type MemoryTaskStatus string

const (
	MemoryTaskPending   MemoryTaskStatus = "pending"
	MemoryTaskRunning   MemoryTaskStatus = "running"
	MemoryTaskCompleted MemoryTaskStatus = "completed"
	MemoryTaskFailed    MemoryTaskStatus = "failed"
	MemoryTaskCancelled MemoryTaskStatus = "cancelled"
)

// MemoryTaskChunk represents one step in a chunked memory processing pipeline.
// Instead of generating all memory updates in one giant LLM call, tasks are
// broken into sequential chunks (e.g. current state → protagonist → important
// characters) to avoid token overflow and truncated output.
type MemoryTaskChunk struct {
	Name        string // human-readable step name (e.g. "current_state")
	Description string // what this chunk processes
}

// MemoryTask represents a unit of background memory work.
type MemoryTask struct {
	ID          string
	AgentKind   string
	SessionID   string
	Workspace   string
	StoryID     string
	BranchID    string
	TurnID      string
	Chunks      []MemoryTaskChunk
	Handler     MemoryTaskHandler
	ContextData map[string]interface{} // arbitrary data the handler needs
}

// MemoryTaskHandler processes a single chunk of a memory task.
// It receives the task, the chunk index, and an emit function for SSE events.
type MemoryTaskHandler func(ctx context.Context, task MemoryTask, chunkIndex int, emit func(Event)) error

// MemoryTaskInfo is the external view of a task's current state.
type MemoryTaskInfo struct {
	ID           string            `json:"id"`
	AgentKind    string            `json:"agent_kind"`
	SessionID    string            `json:"session_id"`
	StoryID      string            `json:"story_id,omitempty"`
	BranchID     string            `json:"branch_id,omitempty"`
	Status       MemoryTaskStatus  `json:"status"`
	TotalChunks  int               `json:"total_chunks"`
	DoneChunks   int               `json:"done_chunks"`
	CurrentChunk string            `json:"current_chunk,omitempty"`
	Error        string            `json:"error,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	StartedAt    *time.Time        `json:"started_at,omitempty"`
	CompletedAt  *time.Time        `json:"completed_at,omitempty"`
}

// MemoryWorkerConfig controls worker behavior.
type MemoryWorkerConfig struct {
	QueueSize      int           // channel buffer size (default 32)
	TaskTimeout    time.Duration // per-task timeout (default 10m)
	StopTimeout    time.Duration // graceful stop wait (default 5s)
}

func (c MemoryWorkerConfig) withDefaults() MemoryWorkerConfig {
	if c.QueueSize <= 0 {
		c.QueueSize = 32
	}
	if c.TaskTimeout <= 0 {
		c.TaskTimeout = 10 * time.Minute
	}
	if c.StopTimeout <= 0 {
		c.StopTimeout = 5 * time.Second
	}
	return c
}

// AsyncMemoryWorker processes memory tasks in the background using a single
// goroutine reading from a channel. Tasks are processed FIFO and serially to
// avoid concurrent LLM calls and file write contention.
//
// Each task is broken into chunks — the worker emits progress events after
// each chunk completes, so the frontend can show incremental progress instead
// of waiting for the entire task to finish.
type AsyncMemoryWorker struct {
	config   MemoryWorkerConfig
	queue    chan MemoryTask
	emit     func(Event)
	notify   func(Event) // external SSE emitter (may be nil)
	logger   *slog.Logger
	wg       sync.WaitGroup
	stopped  atomic.Bool
	counter  atomic.Int64
	tasks    sync.Map // taskID -> *MemoryTaskInfo
}

// NewAsyncMemoryWorker creates and starts a background worker.
// The emit function receives SSE events for task progress; pass nil to suppress.
func NewAsyncMemoryWorker(config MemoryWorkerConfig, emit func(Event)) *AsyncMemoryWorker {
	config = config.withDefaults()
	w := &AsyncMemoryWorker{
		config: config,
		queue:  make(chan MemoryTask, config.QueueSize),
		emit:   emit,
		notify: emit,
		logger: slog.Default().With("component", "async-memory-worker"),
	}
	w.wg.Add(1)
	go w.loop()
	return w
}

// Enqueue adds a memory task to the processing queue.
// Returns an error if the worker has been stopped or the queue is full.
func (w *AsyncMemoryWorker) Enqueue(task MemoryTask) error {
	if w.stopped.Load() {
		return fmt.Errorf("memory worker has been stopped")
	}
	info := &MemoryTaskInfo{
		ID:          task.ID,
		AgentKind:   task.AgentKind,
		SessionID:   task.SessionID,
		StoryID:     task.StoryID,
		BranchID:    task.BranchID,
		Status:      MemoryTaskPending,
		TotalChunks: len(task.Chunks),
		CreatedAt:   time.Now(),
	}
	w.tasks.Store(task.ID, info)

	select {
	case w.queue <- task:
		w.emitEvent(Event{
			Type: "memory_task_queued",
			Data: map[string]interface{}{
				"task_id":       task.ID,
				"agent_kind":    task.AgentKind,
				"session_id":    task.SessionID,
				"total_chunks":  len(task.Chunks),
			},
		})
		return nil
	default:
		w.tasks.Delete(task.ID)
		return fmt.Errorf("memory task queue is full (%d pending)", len(w.queue))
	}
}

// TaskInfo returns the current status of a task, or nil if not found.
func (w *AsyncMemoryWorker) TaskInfo(taskID string) *MemoryTaskInfo {
	if v, ok := w.tasks.Load(taskID); ok {
		info := v.(*MemoryTaskInfo)
		return info
	}
	return nil
}

// AllTasks returns a snapshot of all tracked tasks.
func (w *AsyncMemoryWorker) AllTasks() []MemoryTaskInfo {
	var result []MemoryTaskInfo
	w.tasks.Range(func(key, value interface{}) bool {
		info := value.(*MemoryTaskInfo)
		result = append(result, *info)
		return true
	})
	return result
}

// Stop gracefully shuts down the worker, waiting up to StopTimeout for
// the current task to finish. Pending tasks in the queue are cancelled.
func (w *AsyncMemoryWorker) Stop() {
	if w.stopped.CompareAndSwap(false, true) {
		close(w.queue)
		done := make(chan struct{})
		go func() {
			w.wg.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(w.config.StopTimeout):
			w.logger.Warn("stop_timeout_exceeded")
		}
	}
}

func (w *AsyncMemoryWorker) loop() {
	defer w.wg.Done()
	for task := range w.queue {
		w.processTask(task)
	}
}

func (w *AsyncMemoryWorker) processTask(task MemoryTask) {
	ctx, cancel := context.WithTimeout(context.Background(), w.config.TaskTimeout)
	defer cancel()

	now := time.Now()
	w.updateTaskInfo(task.ID, func(info *MemoryTaskInfo) {
		info.Status = MemoryTaskRunning
		info.StartedAt = &now
	})

	w.emitEvent(Event{
		Type: "memory_task_started",
		Data: map[string]interface{}{
			"task_id":      task.ID,
			"agent_kind":   task.AgentKind,
			"session_id":   task.SessionID,
			"total_chunks": len(task.Chunks),
		},
	})

	var taskErr error
	for i, chunk := range task.Chunks {
		if ctx.Err() != nil {
			taskErr = ctx.Err()
			break
		}
		w.updateTaskInfo(task.ID, func(info *MemoryTaskInfo) {
			info.CurrentChunk = chunk.Name
		})

		w.emitEvent(Event{
			Type: "memory_task_chunk_started",
			Data: map[string]interface{}{
				"task_id":      task.ID,
				"chunk_index":  i,
				"chunk_name":   chunk.Name,
				"description":  chunk.Description,
				"total_chunks": len(task.Chunks),
				"done_chunks":  i,
			},
		})

		chunkErr := task.Handler(ctx, task, i, w.emitEvent)
		if chunkErr != nil {
			taskErr = fmt.Errorf("chunk %q failed: %w", chunk.Name, chunkErr)
			break
		}

		w.updateTaskInfo(task.ID, func(info *MemoryTaskInfo) {
			info.DoneChunks = i + 1
		})

		w.emitEvent(Event{
			Type: "memory_task_chunk_completed",
			Data: map[string]interface{}{
				"task_id":      task.ID,
				"chunk_index":  i,
				"chunk_name":   chunk.Name,
				"total_chunks": len(task.Chunks),
				"done_chunks":  i + 1,
			},
		})
	}

	completedAt := time.Now()
	finalStatus := MemoryTaskCompleted
	errMsg := ""
	if taskErr != nil {
		if ctx.Err() != nil {
			finalStatus = MemoryTaskCancelled
		} else {
			finalStatus = MemoryTaskFailed
		}
		errMsg = taskErr.Error()
		w.logger.Error("task_failed",
			slog.String("task_id", task.ID),
			slog.String("error", errMsg),
		)
	}

	w.updateTaskInfo(task.ID, func(info *MemoryTaskInfo) {
		info.Status = finalStatus
		info.Error = errMsg
		info.CompletedAt = &completedAt
		info.CurrentChunk = ""
	})

	w.emitEvent(Event{
		Type: "memory_task_completed",
		Data: map[string]interface{}{
			"task_id":      task.ID,
			"agent_kind":   task.AgentKind,
			"session_id":   task.SessionID,
			"status":       string(finalStatus),
			"total_chunks": len(task.Chunks),
			"done_chunks":  func() int {
				if v, ok := w.tasks.Load(task.ID); ok {
					return v.(*MemoryTaskInfo).DoneChunks
				}
				return 0
			}(),
			"error": errMsg,
		},
	})
}

func (w *AsyncMemoryWorker) updateTaskInfo(taskID string, fn func(*MemoryTaskInfo)) {
	if v, ok := w.tasks.Load(taskID); ok {
		info := v.(*MemoryTaskInfo)
		// MemoryTaskInfo fields are only written by the single worker goroutine,
		// so no lock is needed for mutations. Reads from AllTasks/TaskInfo
		// may see slightly stale data, which is acceptable for status display.
		fn(info)
	}
}

func (w *AsyncMemoryWorker) emitEvent(ev Event) {
	if w.notify != nil {
		w.notify(ev)
	}
}

// GenerateTaskID creates a unique task ID using a monotonic counter.
func (w *AsyncMemoryWorker) GenerateTaskID(prefix string) string {
	if prefix == "" {
		prefix = "mem"
	}
	n := w.counter.Add(1)
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().Unix(), n)
}

// DefaultMemoryChunks returns the standard chunked processing pipeline
// for story memory tasks, addressing the user's request for incremental
// memory updates instead of one large LLM call.
//
// Order matters: current state is updated first (cheap, high value),
// then protagonist info, then important characters, then events/plot.
func DefaultMemoryChunks() []MemoryTaskChunk {
	return []MemoryTaskChunk{
		{
			Name:        "current_state",
			Description: "更新当前场景状态、主角状态和当前目标",
		},
		{
			Name:        "protagonist",
			Description: "整理主角的属性、关系和能力变化",
		},
		{
			Name:        "important_characters",
			Description: "更新重要 NPC 的状态、态度和关系",
		},
		{
			Name:        "events_and_plot",
			Description: "记录关键事件、伏笔和剧情进展",
		},
	}
}
