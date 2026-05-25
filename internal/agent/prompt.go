package agent

import (
	"nova/config"
	"nova/internal/book"
	"nova/internal/prompts"
)

// BuildInstruction 构建系统指令，包含基础 prompt + 作品状态注入。
// 实际的 Prompt 文本集中在 internal/prompts 包，这里只负责把 cfg/state 翻译成 prompts.SystemInstructionInput。
func BuildInstruction(cfg *config.Config, state *book.State) string {
	return prompts.BuildSystemInstruction(prompts.SystemInstructionInput{
		CreatorPrompt: state.ReadCreatorPrompt(),
		Workspace:     cfg.Workspace,
		StateContext:  state.CompactContext(),
	})
}

func BuildInteractiveStoryInstruction(cfg *config.Config, state *book.State) string {
	workspace := ""
	if cfg != nil {
		workspace = cfg.Workspace
	}
	creator := ""
	if state != nil {
		creator = state.ReadCreatorPrompt()
	}
	return prompts.BuildInteractiveStorySystemInstruction(prompts.InteractiveStorySystemInstructionInput{
		CreatorPrompt: creator,
		Workspace:     workspace,
	})
}
