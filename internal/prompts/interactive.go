package prompts

import (
	"fmt"
	"strings"
)

type InteractiveStorySystemInstructionInput struct {
	CreatorPrompt string
	Workspace     string
}

type InteractiveStoryPromptInput struct {
	Title             string
	Origin            string
	StoryTellerID     string
	StoryTeller       string
	BranchID          string
	Characters        string
	WorldBuilding     string
	SnapshotStateJSON string
}

func BuildInteractiveStorySystemInstruction(in InteractiveStorySystemInstructionInput) string {
	var sb strings.Builder
	if creator := strings.TrimSpace(in.CreatorPrompt); creator != "" {
		sb.WriteString("# 创作者指令（最高优先级）\n\n")
		sb.WriteString(creator)
		sb.WriteString("\n\n---\n\n")
	}
	sb.WriteString("你是 Nova 的互动故事模式 Agent，只负责根据用户行动生成故事舞台上的下一回合内容。\n\n")
	sb.WriteString("## 模式边界\n")
	sb.WriteString("- 当前模式是互动故事模式，不是 IDE 写章节模式。\n")
	sb.WriteString("- 你的输出会流式展示到主屏幕的故事舞台，并由后端写入 interactive/story/story-{id}.jsonl。\n")
	sb.WriteString("- 禁止使用写文件工具，包括 write_file、edit_file、delete_file 以及任何会修改 workspace 文件的工具。\n")
	sb.WriteString("- 不要创建或修改 chapters、outline、progress、characters 等文件；互动状态只能通过 <STATE_DELTA> JSON 表达。\n")
	sb.WriteString("- 可以基于已注入的故事上下文、共享设定和当前快照继续剧情。\n\n")
	sb.WriteString("## 输出协议\n")
	sb.WriteString("必须只输出以下结构，不要输出计划、解释、工具说明或 Markdown 标题：\n")
	sb.WriteString("<NARRATIVE>\n本回合展示在故事舞台上的正文\n</NARRATIVE>\n")
	sb.WriteString("<STATE_DELTA>\n{\"ops\":[{\"op\":\"set\",\"path\":\"on_stage\",\"value\":[\"角色名\"]}]}\n</STATE_DELTA>\n")
	sb.WriteString("如果没有明确状态变化，可以省略整个 <STATE_DELTA> 块。\n")
	if ws := strings.TrimSpace(in.Workspace); ws != "" {
		sb.WriteString("\n## 作品工作目录\n")
		sb.WriteString(ws)
		sb.WriteString("\n")
	}
	return sb.String()
}

func InteractiveStoryContext(in InteractiveStoryPromptInput) string {
	var sb strings.Builder
	sb.WriteString("[互动故事模式]\n")
	sb.WriteString("你正在为 Nova 的互动 story 子模式生成下一回合内容。输出会直接流式显示到故事舞台，并在结束后写入 interactive/story/story-{id}.jsonl。\n\n")
	sb.WriteString("## 输出协议\n")
	sb.WriteString("必须严格输出以下结构，不要输出额外解释、计划、工具说明或 Markdown 标题：\n")
	sb.WriteString("<NARRATIVE>\n")
	sb.WriteString("本回合面向读者展示的故事正文\n")
	sb.WriteString("</NARRATIVE>\n")
	sb.WriteString("<STATE_DELTA>\n")
	sb.WriteString("{\"ops\":[{\"op\":\"set\",\"path\":\"on_stage\",\"value\":[\"角色名\"]}]}\n")
	sb.WriteString("</STATE_DELTA>\n\n")
	sb.WriteString("如果本回合没有明确状态变化，可以省略整个 <STATE_DELTA> 块。STATE_DELTA 只记录本回合已经发生、确定成立的变化，不要记录未来计划。\n")
	sb.WriteString("状态 path 仅允许 on_stage、characters.<角色名>、events 及其子路径；op 仅允许 set、merge、push、pull、inc、unset。\n\n")
	sb.WriteString("## 故事信息\n")
	writeField(&sb, "标题", in.Title)
	writeField(&sb, "开端", in.Origin)
	writeField(&sb, "当前分支", in.BranchID)
	writeField(&sb, "讲述者 ID", in.StoryTellerID)
	writeBlock(&sb, "讲述者提示词", in.StoryTeller)
	writeBlock(&sb, "角色设定", in.Characters)
	writeBlock(&sb, "世界观设定", in.WorldBuilding)
	writeBlock(&sb, "当前互动状态快照(JSON)", in.SnapshotStateJSON)
	return sb.String()
}

func InteractiveStoryTurnInstruction(message string) string {
	return fmt.Sprintf(`[互动输入]
用户本回合行动：
%s

请基于互动故事上下文续写下一回合。NARRATIVE 只写读者应看到的故事正文；STATE_DELTA 只写本回合造成的状态变化。
必须使用 <NARRATIVE>...</NARRATIVE> 包裹正文；如有状态变化，再追加 <STATE_DELTA>...</STATE_DELTA> JSON。`, strings.TrimSpace(message))
}

func writeField(sb *strings.Builder, name, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "（空）"
	}
	fmt.Fprintf(sb, "- %s：%s\n", name, value)
}

func writeBlock(sb *strings.Builder, title, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "（空）"
	}
	fmt.Fprintf(sb, "\n## %s\n\n%s\n", title, value)
}
