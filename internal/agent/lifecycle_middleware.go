package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// lifecycleMiddleware bridges the ADK middleware system to LifecycleHooks.
// It fires HookPhaseModelCall before each model call and HookPhaseToolResult
// after each tool result is collected.
type lifecycleMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
	registry *HookRegistry
	runCtx   *RunContext
}

func newLifecycleMiddleware(registry *HookRegistry, runCtx *RunContext) *lifecycleMiddleware {
	return &lifecycleMiddleware{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
		registry:                     registry,
		runCtx:                       runCtx,
	}
}

// BeforeModelRewriteState fires HookPhaseModelCall before each LLM call.
func (m *lifecycleMiddleware) BeforeModelRewriteState(
	ctx context.Context,
	state *adk.ChatModelAgentState,
	mc *adk.ModelContext,
) (context.Context, *adk.ChatModelAgentState, error) {
	if m.registry != nil && m.runCtx != nil && state != nil {
		info := ModelCallInfo{
			MessageCount: len(state.Messages),
			AgentKind:    m.runCtx.AgentKind,
		}
		// Hook errors are logged but do not block the model call.
		_ = m.registry.fireModelCall(context.Background(), m.runCtx, info)
	}
	return ctx, state, nil
}

// WrapInvokableToolCall fires HookPhaseToolResult after each synchronous tool call.
func (m *lifecycleMiddleware) WrapInvokableToolCall(
	_ context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	toolCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		result, err := endpoint(ctx, args, opts...)
		if m.registry != nil && m.runCtx != nil {
			info := ToolResultInfo{
				ToolName:   toolName(toolCtx),
				Result:     result,
				AgentKind:  m.runCtx.AgentKind,
			}
			_ = m.registry.fireToolResult(context.Background(), m.runCtx, info)
		}
		return result, err
	}, nil
}

// WrapStreamableToolCall fires HookPhaseToolResult after each streaming tool call
// completes. It wraps the reader to collect the full result before firing the hook.
func (m *lifecycleMiddleware) WrapStreamableToolCall(
	_ context.Context,
	endpoint adk.StreamableToolCallEndpoint,
	toolCtx *adk.ToolContext,
) (adk.StreamableToolCallEndpoint, error) {
	return func(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
		reader, err := endpoint(ctx, args, opts...)
		if err != nil {
			return reader, err
		}
		if m.registry == nil || m.runCtx == nil {
			return reader, nil
		}
		toolNameVal := toolName(toolCtx)
		agentKind := m.runCtx.AgentKind

		// Create a pipe: we read from the original reader, forward every chunk
		// to the caller, and collect content for the post-completion hook.
		outReader, outWriter := schema.Pipe[string](64)
		go func() {
			defer outWriter.Close()
			var collected strings.Builder
			for {
				chunk, recvErr := reader.Recv()
				if errors.Is(recvErr, io.EOF) {
					// Stream ended — fire the hook with collected content.
					if collected.Len() > 0 {
						info := ToolResultInfo{
							ToolName:  toolNameVal,
							Result:    collected.String(),
							AgentKind: agentKind,
						}
						_ = m.registry.fireToolResult(context.Background(), m.runCtx, info)
					}
					return
				}
				if recvErr != nil {
					// Error — fire the hook with whatever we collected so far.
					if collected.Len() > 0 {
						info := ToolResultInfo{
							ToolName:  toolNameVal,
							Result:    collected.String(),
							AgentKind: agentKind,
						}
						_ = m.registry.fireToolResult(context.Background(), m.runCtx, info)
					}
					_ = outWriter.Send(fmt.Sprintf("\n[tool error] %v", recvErr), nil)
					return
				}
				collected.WriteString(chunk)
				_ = outWriter.Send(chunk, nil)
			}
		}()
		return outReader, nil
	}, nil
}
