package agent

import (
	"context"
	"sort"
	"sync"
)

// HookPhase identifies when a lifecycle hook fires relative to the Agent run.
type HookPhase string

const (
	HookPhaseRunStart     HookPhase = "run_start"
	HookPhaseModelCall    HookPhase = "model_call"
	HookPhaseToolResult   HookPhase = "tool_result"
	HookPhaseRunComplete  HookPhase = "run_complete"
)

// RunContext carries immutable run metadata available to every hook invocation.
type RunContext struct {
	RunID       string
	TaskID      string
	AgentKind   string
	SessionID   string
	Workspace   string
	Mode        string
	UserMessage string
}

// RunResult describes the outcome of a completed Agent run.
type RunResult struct {
	Status         string // "success" | "error" | "aborted" | "panic"
	GeneratedBytes int
	Error          string
}

// ToolResultInfo carries tool execution details for HookPhaseToolResult.
type ToolResultInfo struct {
	ToolName    string
	ToolCallID  string
	Result      string
	AgentKind   string
}

// ModelCallInfo carries message context for HookPhaseModelCall.
type ModelCallInfo struct {
	MessageCount int
	AgentKind    string
}

// LifecycleHook is a single extension point in the Agent run lifecycle.
// Each method is optional — a hook may implement only the phases it cares about.
// Methods must be non-blocking; long work should be dispatched to a goroutine.
type LifecycleHook interface {
	Name() string
	OnRunStart(ctx context.Context, runCtx *RunContext) error
	OnModelCall(ctx context.Context, runCtx *RunContext, info ModelCallInfo) error
	OnToolResult(ctx context.Context, runCtx *RunContext, info ToolResultInfo) error
	OnRunComplete(ctx context.Context, runCtx *RunContext, result RunResult) error
}

// HookRegistry holds registered lifecycle hooks sorted by priority.
// Lower priority numbers execute first. The zero value is ready to use.
type HookRegistry struct {
	mu     sync.RWMutex
	hooks  []registeredHook
	sorted bool
}

type registeredHook struct {
	priority int
	hook     LifecycleHook
}

// Register adds a hook with the given priority (lower = earlier).
// If priority is 0, DefaultHookPriority is used.
func (r *HookRegistry) Register(hook LifecycleHook, priority int) {
	if priority == 0 {
		priority = DefaultHookPriority
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hooks = append(r.hooks, registeredHook{priority: priority, hook: hook})
	r.sorted = false
}

// Hooks returns a snapshot of registered hooks in execution order.
func (r *HookRegistry) Hooks() []LifecycleHook {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.sorted {
		sorted := make([]registeredHook, len(r.hooks))
		copy(sorted, r.hooks)
		sort.SliceStable(sorted, func(i, j int) bool {
			return sorted[i].priority < sorted[j].priority
		})
		result := make([]LifecycleHook, len(sorted))
		for i, h := range sorted {
			result[i] = h.hook
		}
		return result
	}
	result := make([]LifecycleHook, len(r.hooks))
	for i, h := range r.hooks {
		result[i] = h.hook
	}
	return result
}

// fireRunStart calls OnRunStart on every registered hook, collecting errors.
func (r *HookRegistry) fireRunStart(ctx context.Context, runCtx *RunContext) []error {
	var errs []error
	for _, h := range r.Hooks() {
		if err := h.OnRunStart(ctx, runCtx); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// fireModelCall calls OnModelCall on every registered hook.
func (r *HookRegistry) fireModelCall(ctx context.Context, runCtx *RunContext, info ModelCallInfo) []error {
	var errs []error
	for _, h := range r.Hooks() {
		if err := h.OnModelCall(ctx, runCtx, info); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// fireToolResult calls OnToolResult on every registered hook.
func (r *HookRegistry) fireToolResult(ctx context.Context, runCtx *RunContext, info ToolResultInfo) []error {
	var errs []error
	for _, h := range r.Hooks() {
		if err := h.OnToolResult(ctx, runCtx, info); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// fireRunComplete calls OnRunComplete on every registered hook.
func (r *HookRegistry) fireRunComplete(ctx context.Context, runCtx *RunContext, result RunResult) []error {
	var errs []error
	for _, h := range r.Hooks() {
		if err := h.OnRunComplete(ctx, runCtx, result); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// HookPriority constants define conventional priority bands.
const (
	HookPriorityHigh    = 10
	DefaultHookPriority = 50
	HookPriorityLow     = 100
)

// HookFunc is a function-based adapter implementing LifecycleHook.
// Set only the fields you need; unset functions are no-ops.
type HookFunc struct {
	NameVal         string
	OnRunStartFn    func(ctx context.Context, runCtx *RunContext) error
	OnModelCallFn   func(ctx context.Context, runCtx *RunContext, info ModelCallInfo) error
	OnToolResultFn  func(ctx context.Context, runCtx *RunContext, info ToolResultInfo) error
	OnRunCompleteFn func(ctx context.Context, runCtx *RunContext, result RunResult) error
}

func (h *HookFunc) Name() string {
	if h.NameVal != "" {
		return h.NameVal
	}
	return "hook-func"
}

func (h *HookFunc) OnRunStart(ctx context.Context, runCtx *RunContext) error {
	if h.OnRunStartFn != nil {
		return h.OnRunStartFn(ctx, runCtx)
	}
	return nil
}

func (h *HookFunc) OnModelCall(ctx context.Context, runCtx *RunContext, info ModelCallInfo) error {
	if h.OnModelCallFn != nil {
		return h.OnModelCallFn(ctx, runCtx, info)
	}
	return nil
}

func (h *HookFunc) OnToolResult(ctx context.Context, runCtx *RunContext, info ToolResultInfo) error {
	if h.OnToolResultFn != nil {
		return h.OnToolResultFn(ctx, runCtx, info)
	}
	return nil
}

func (h *HookFunc) OnRunComplete(ctx context.Context, runCtx *RunContext, result RunResult) error {
	if h.OnRunCompleteFn != nil {
		return h.OnRunCompleteFn(ctx, runCtx, result)
	}
	return nil
}

// lifecycleMiddleware is defined in lifecycle_middleware.go where it can embed
// adk.BaseChatModelAgentMiddleware without creating an import cycle.
