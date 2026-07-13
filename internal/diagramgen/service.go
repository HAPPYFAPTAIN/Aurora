// Package diagramgen 负责通过 LLM 生成 Mermaid 格式的图表。
//
// 调用链路：handler -> DiagramAppService.Generate -> Service.Generate -> OpenAI Chat Completions API。
// 模型配置通过 config.ResolveAgentModel 解析，复用 IDE Agent 的 profile。
package diagramgen

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"denova/config"
)

// 错误定义。
var (
	ErrTypeRequired  = errors.New("图表类型不能为空")
	ErrModelNotConfig = errors.New("模型未配置，请先在设置中配置 API Key 和模型")
	ErrEmptyResponse  = errors.New("模型返回内容为空")
)

// DiagramType 是预设的图表类型。
type DiagramType string

const (
	TypeCharacter DiagramType = "character" // 人物关系图
	TypeTimeline  DiagramType = "timeline"  // 剧情时间线
	TypeWorldMap  DiagramType = "worldmap"  // 世界地图
	TypeStructure DiagramType = "structure" // 故事结构图
	TypeFaction   DiagramType = "faction"   // 势力关系图
)

// AllTypes 返回所有支持的图表类型。
func AllTypes() []DiagramType {
	return []DiagramType{TypeCharacter, TypeTimeline, TypeWorldMap, TypeStructure, TypeFaction}
}

// GenerateRequest 是图表生成请求。
type GenerateRequest struct {
	// Type 是预设的图表类型（character/timeline/worldmap/structure/faction）。
	Type DiagramType `json:"type"`
	// Context 是由 App 层自动收集的工作区上下文。
	Context string `json:"context,omitempty"`
}

// GenerateResult 是图表生成结果。
type GenerateResult struct {
	// XML 是 Mermaid 格式的代码字符串。
	XML string `json:"xml"`
}

// typePrompt 定义每种图表类型的系统提示词。
var typePrompts = map[DiagramType]string{
	TypeCharacter: `Generate a Mermaid flowchart (use "flowchart LR" for left-to-right layout) showing the relationships between characters.
- Each character is a node with their name
- Use labeled arrows to show relationships (e.g., "朋友", "敌人", "父子", "恋人", "师徒")
- Use different colors via classDef to distinguish allies vs enemies
- Include all major characters mentioned in the context
- Keep it clean and readable`,

	TypeTimeline: `Generate a Mermaid flowchart (use "flowchart TD" for top-down layout) showing the story timeline.
- Each major event/chapter is a node with the chapter title
- Use arrows to show chronological progression
- Group related events using subgraph if needed
- Include volume/arc divisions as subgraphs
- Keep node labels concise (chapter number + title)`,

	TypeWorldMap: `Generate a Mermaid flowchart (use "flowchart TD") showing the world/setting structure as a hierarchical map.
- Use subgraphs to represent regions, continents, or major locations
- Each location is a node with its name
- Use arrows to show connections between locations (e.g., "邻接", "包含", "位于")
- Use classDef to color-code different region types (city, wilderness, dungeon, etc.)
- Include all locations mentioned in the context`,

	TypeStructure: `Generate a Mermaid flowchart (use "flowchart TD") showing the story structure.
- Use subgraphs to represent story arcs or volumes
- Each major plot point is a node
- Show the hierarchy: beginning -> rising action -> climax -> resolution
- Use arrows to show narrative flow
- Keep it concise but comprehensive`,

	TypeFaction: `Generate a Mermaid flowchart (use "flowchart LR") showing the relationships between factions/forces.
- Each faction is a node with its name
- Use labeled arrows to show relationships (e.g., "同盟", "敌对", "附庸", "贸易")
- Use classDef to color-code different faction types
- Include all factions mentioned in the context
- Show power dynamics through the layout`,
}

// diagramSystemPrompt 是通用的系统提示词，约束输出为 Mermaid 格式。
const diagramSystemPrompt = `You are a diagram generation assistant. Generate a valid Mermaid diagram based on the user's request.

Rules:
1. Output ONLY the Mermaid code, no explanation, no markdown code fences
2. Use valid Mermaid syntax only
3. Use Chinese labels for all nodes and connections (matching the story's language)
4. Keep the layout clean and readable
5. Use classDef for color styling when appropriate

The output must start with a Mermaid diagram type keyword (e.g. "flowchart", "sequenceDiagram", "classDiagram", "stateDiagram", "gantt", etc.) and be valid Mermaid syntax.`

// Service 是图表生成服务，通过 OpenAI 兼容的 Chat Completions API 生成 Mermaid 代码。
type Service struct {
	httpClient *http.Client
}

// NewService 创建图表生成服务。传入 nil httpClient 时使用默认 http.Client。
func NewService(httpClient *http.Client) *Service {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Service{httpClient: httpClient}
}

// Generate 根据 type 和 context 调用 LLM 生成 Mermaid 代码。
func (s *Service) Generate(ctx context.Context, cfg *config.Config, agentKind string, request GenerateRequest) (GenerateResult, error) {
	typePrompt, ok := typePrompts[request.Type]
	if !ok {
		return GenerateResult{}, ErrTypeRequired
	}

	resolved := config.ResolveAgentModel(cfg, agentKind)
	if resolved.OpenAIAPIKey == "" || resolved.OpenAIModel == "" {
		return GenerateResult{}, ErrModelNotConfig
	}

	userContent := typePrompt
	if ctxStr := strings.TrimSpace(request.Context); ctxStr != "" {
		userContent = fmt.Sprintf("%s\n\n---\n\nStory Context:\n%s", typePrompt, ctxStr)
	}

	baseURL := strings.TrimRight(resolved.OpenAIBaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	url := baseURL + "/chat/completions"

	temp := 0.7
	if resolved.Temperature != nil {
		temp = *resolved.Temperature
	}

	reqBody := chatCompletionsRequest{
		Model: resolved.OpenAIModel,
		Messages: []chatMessage{
			{Role: "system", Content: diagramSystemPrompt},
			{Role: "user", Content: userContent},
		},
		Temperature: temp,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("序列化请求失败: %w", err)
	}

	log.Printf("[diagramgen] generate begin model=%q type=%q context_len=%d",
		resolved.OpenAIModel, request.Type, len(request.Context))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return GenerateResult{}, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+resolved.OpenAIAPIKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("调用模型失败: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return GenerateResult{}, fmt.Errorf("模型返回错误: HTTP %d, body: %s", resp.StatusCode, string(respBytes))
	}

	var chatResp chatCompletionsResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return GenerateResult{}, fmt.Errorf("解析响应失败: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return GenerateResult{}, ErrEmptyResponse
	}

	content := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	if content == "" {
		return GenerateResult{}, ErrEmptyResponse
	}

	mermaid := extractMermaid(content)
	log.Printf("[diagramgen] generate done model=%q type=%q mermaid_len=%d", resolved.OpenAIModel, request.Type, len(mermaid))

	return GenerateResult{XML: mermaid}, nil
}

// extractMermaid 从模型输出中提取 Mermaid 代码。
func extractMermaid(content string) string {
	trimmed := strings.TrimSpace(content)

	// 去除 markdown 代码块包裹
	if strings.HasPrefix(trimmed, "```") {
		lines := strings.SplitN(trimmed, "\n", 2)
		if len(lines) > 1 {
			trimmed = strings.TrimSpace(lines[1])
		}
		trimmed = strings.TrimSuffix(trimmed, "```")
		trimmed = strings.TrimSpace(trimmed)
	}

	return trimmed
}

// chatCompletionsRequest 是 OpenAI Chat Completions API 的请求体。
type chatCompletionsRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

// chatMessage 是对话消息。
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatCompletionsResponse 是 OpenAI Chat Completions API 的响应体。
type chatCompletionsResponse struct {
	Choices []struct {
		Message      chatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
}
