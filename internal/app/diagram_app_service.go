package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	"denova/config"
	"denova/internal/diagramgen"
)

// DiagramAppService 负责图表生成任务的 App 层封装，
// 根据图表类型自动从工作区收集上下文，然后委托给 diagramgen.Service。
type DiagramAppService struct {
	app *App
}

// DiagramGenerateResult 是图表生成的对外结果。
type DiagramGenerateResult struct {
	XML string `json:"xml"`
}

// GenerateDiagram 是 App 门面方法，委托给 DiagramAppService。
func (a *App) GenerateDiagram(ctx context.Context, request diagramgen.GenerateRequest) (DiagramGenerateResult, error) {
	return a.diagrams().Generate(ctx, request)
}

// Generate 解析当前运行时配置后调用 diagramgen.Service 生成图表。
// 会根据 request.Type 自动从工作区收集相关上下文。
func (s *DiagramAppService) Generate(ctx context.Context, request diagramgen.GenerateRequest) (DiagramGenerateResult, error) {
	// 根据图表类型自动收集工作区上下文
	wsContext := s.collectContext(request.Type)
	if wsContext != "" {
		request.Context = wsContext
		log.Printf("[diagram-app] collected context for type=%q len=%d", request.Type, len(wsContext))
	}

	cfg, err := s.configSnapshot()
	if err != nil {
		return DiagramGenerateResult{}, err
	}
	result, err := diagramgen.NewService(nil).Generate(ctx, &cfg, config.AgentKindIDE, request)
	if err != nil {
		return DiagramGenerateResult{}, err
	}
	return DiagramGenerateResult{XML: result.XML}, nil
}

// collectContext 根据图表类型从工作区收集相关上下文数据。
func (s *DiagramAppService) collectContext(diagramType diagramgen.DiagramType) string {
	if !s.app.HasWorkspace() {
		return ""
	}

	switch diagramType {
	case diagramgen.TypeCharacter:
		return s.collectCharacterContext()
	case diagramgen.TypeTimeline:
		return s.collectTimelineContext()
	case diagramgen.TypeWorldMap:
		return s.collectWorldMapContext()
	case diagramgen.TypeStructure:
		return s.collectStructureContext()
	case diagramgen.TypeFaction:
		return s.collectFactionContext()
	default:
		return ""
	}
}

// collectCharacterContext 收集角色相关上下文：资料库中类型为 character 的条目 + 角色状态。
func (s *DiagramAppService) collectCharacterContext() string {
	var sb strings.Builder
	sb.WriteString("## 角色列表\n\n")

	lores, err := s.app.LoreItems()
	if err == nil {
		for _, l := range lores {
			if l.Type != "character" && l.Type != "Character" {
				continue
			}
			sb.WriteString(fmt.Sprintf("- **%s**", l.Name))
			if l.BriefDescription != "" {
				sb.WriteString(fmt.Sprintf("： %s", l.BriefDescription))
			}
			if len(l.Tags) > 0 {
				sb.WriteString(fmt.Sprintf(" [标签: %s]", strings.Join(l.Tags, ", ")))
			}
			sb.WriteString("\n")
			if l.Content != "" {
				// 只取前 500 字，避免上下文过长
				content := l.Content
				if len([]rune(content)) > 500 {
					content = string([]rune(content)[:500]) + "..."
				}
				sb.WriteString(fmt.Sprintf("  详情: %s\n", content))
			}
		}
	}

	// 追加角色状态文件
	_, compactCtx := s.app.Status()
	if compactCtx != "" {
		// 从 compact context 中提取角色状态部分
		if idx := strings.Index(compactCtx, "## 角色状态"); idx >= 0 {
			end := strings.Index(compactCtx[idx:], "\n## ")
			if end < 0 {
				end = len(compactCtx) - idx
			}
			sb.WriteString("\n## 角色状态\n\n")
			sb.WriteString(compactCtx[idx : idx+end])
		}
	}

	return sb.String()
}

// collectTimelineContext 收集时间线上下文：章节列表（标题、卷名）。
func (s *DiagramAppService) collectTimelineContext() string {
	summary, err := s.app.BookService().Summary()
	if err != nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## 章节列表\n\n")

	currentVolume := ""
	for _, ch := range summary.Chapters {
		vol := ch.Volume
		if vol == "" {
			vol = "默认"
		}
		if vol != currentVolume {
			sb.WriteString(fmt.Sprintf("\n### %s\n\n", vol))
			currentVolume = vol
		}
		sb.WriteString(fmt.Sprintf("- 第%d章: %s (字数: %d, 状态: %s)\n",
			ch.Index, ch.DisplayTitle, ch.Words, ch.Status))
	}

	return sb.String()
}

// collectWorldMapContext 收集世界地图上下文：资料库中类型为 world/location 的条目。
func (s *DiagramAppService) collectWorldMapContext() string {
	var sb strings.Builder
	sb.WriteString("## 世界设定与地点\n\n")

	lores, err := s.app.LoreItems()
	if err == nil {
		for _, l := range lores {
			ltype := strings.ToLower(l.Type)
			if ltype != "world" && ltype != "location" && ltype != "世界" && ltype != "地点" {
				continue
			}
			sb.WriteString(fmt.Sprintf("- **%s** (类型: %s)", l.Name, l.Type))
			if l.BriefDescription != "" {
				sb.WriteString(fmt.Sprintf("： %s", l.BriefDescription))
			}
			sb.WriteString("\n")
			if l.Content != "" {
				content := l.Content
				if len([]rune(content)) > 500 {
					content = string([]rune(content)[:500]) + "..."
				}
				sb.WriteString(fmt.Sprintf("  详情: %s\n", content))
			}
		}
	}

	return sb.String()
}

// collectStructureContext 收集故事结构上下文：大纲、章节计划、章节列表。
func (s *DiagramAppService) collectStructureContext() string {
	var sb strings.Builder

	summary, err := s.app.BookService().Summary()
	if err == nil {
		if summary.Outline != nil && summary.Outline.Excerpt != "" {
			sb.WriteString("## 大纲\n\n")
			outline := summary.Outline.Excerpt
			if len([]rune(outline)) > 2000 {
				outline = string([]rune(outline)[:2000]) + "..."
			}
			sb.WriteString(outline)
			sb.WriteString("\n\n")
		}

		sb.WriteString("## 章节结构\n\n")
		currentVolume := ""
		for _, ch := range summary.Chapters {
			vol := ch.Volume
			if vol == "" {
				vol = "默认"
			}
			if vol != currentVolume {
				sb.WriteString(fmt.Sprintf("\n### %s\n\n", vol))
				currentVolume = vol
			}
			sb.WriteString(fmt.Sprintf("- 第%d章: %s\n", ch.Index, ch.DisplayTitle))
		}

		if len(summary.ChapterPlans) > 0 {
			sb.WriteString("\n## 章节计划\n\n")
			for _, p := range summary.ChapterPlans {
				if p.Excerpt != "" {
					plan := p.Excerpt
					if len([]rune(plan)) > 300 {
						plan = string([]rune(plan)[:300]) + "..."
					}
					sb.WriteString(fmt.Sprintf("- %s: %s\n", p.Title, plan))
				}
			}
		}
	}

	return sb.String()
}

// collectFactionContext 收集势力关系上下文：资料库中类型为 faction 的条目。
func (s *DiagramAppService) collectFactionContext() string {
	var sb strings.Builder
	sb.WriteString("## 势力/组织\n\n")

	lores, err := s.app.LoreItems()
	if err == nil {
		for _, l := range lores {
			ltype := strings.ToLower(l.Type)
			if ltype != "faction" && ltype != "势力" && ltype != "组织" {
				continue
			}
			sb.WriteString(fmt.Sprintf("- **%s**", l.Name))
			if l.BriefDescription != "" {
				sb.WriteString(fmt.Sprintf("： %s", l.BriefDescription))
			}
			sb.WriteString("\n")
			if l.Content != "" {
				content := l.Content
				if len([]rune(content)) > 500 {
					content = string([]rune(content)[:500]) + "..."
				}
				sb.WriteString(fmt.Sprintf("  详情: %s\n", content))
			}
		}
	}

	return sb.String()
}

// configSnapshot 从 App 中安全地读取当前配置快照。
func (s *DiagramAppService) configSnapshot() (config.Config, error) {
	app := s.app
	app.mu.RLock()
	defer app.mu.RUnlock()
	if app.cfg == nil {
		return config.Config{}, fmt.Errorf("运行配置未初始化")
	}
	return *app.cfg, nil
}
