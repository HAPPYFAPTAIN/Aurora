// Package diagramgen 负责通过 LLM 生成 draw.io 格式的 XML 图表。
//
// 调用链路：handler -> DiagramAppService.Generate -> Service.Generate -> OpenAI Chat Completions API。
// 模型配置通过 config.ResolveAgentModel 解析，复用 IDE Agent 的 profile，
// 避免引入额外的配置项；如果未来需要独立 profile，只需在 AgentModelSettings 中新增字段。
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
	ErrPromptRequired = errors.New("图表提示词不能为空")
	ErrModelNotConfig = errors.New("模型未配置，请先在设置中配置 API Key 和模型")
	ErrEmptyResponse  = errors.New("模型返回内容为空")
)

// GenerateRequest 是图表生成请求。
type GenerateRequest struct {
	// Prompt 是用户对图表的自然语言描述，例如"画一个人物关系图，主角张三，反派李四"。
	Prompt string `json:"prompt"`
	// Context 是可选的附加上下文，例如故事摘要、角色列表等，帮助模型生成更准确的图表。
	Context string `json:"context,omitempty"`
}

// GenerateResult 是图表生成结果。
type GenerateResult struct {
	// XML 是 draw.io 格式的 XML 字符串，以 <mxGraphModel> 开头、</mxGraphModel> 结尾。
	XML string `json:"xml"`
}

// diagramSystemPrompt 指导 AI 生成 Mermaid 格式图表的系统提示词。
const diagramSystemPrompt = `You are a diagram generation assistant. Generate a valid Mermaid diagram based on the user's request.

Rules:
1. Output ONLY the Mermaid code, no explanation, no markdown code fences
2. Use appropriate diagram types: flowchart, sequenceDiagram, classDiagram, stateDiagram, gantt, gitGraph, etc.
3. Add clear labels to all nodes and connections
4. Use styles/colors when helpful: linkStyle, classDef, style
5. Keep the layout clean and readable
6. For character relationship diagrams: use flowchart with person nodes and labeled arrows
7. For flowcharts: use flowchart TD or LR with decision diamonds and yes/no labels
8. For timelines: use gantt or flowchart with horizontal layout
9. For story structure: use flowchart with tree/hierarchy layout

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

// Generate 根据 prompt 调用 LLM 生成 draw.io XML。
// cfg 用于解析模型配置（API Key、Base URL、Model 等），agentKind 指定使用哪个 Agent 的模型 profile。
func (s *Service) Generate(ctx context.Context, cfg *config.Config, agentKind string, request GenerateRequest) (GenerateResult, error) {
	if strings.TrimSpace(request.Prompt) == "" {
		return GenerateResult{}, ErrPromptRequired
	}

	resolved := config.ResolveAgentModel(cfg, agentKind)
	if resolved.OpenAIAPIKey == "" || resolved.OpenAIModel == "" {
		return GenerateResult{}, ErrModelNotConfig
	}

	userContent := strings.TrimSpace(request.Prompt)
	if ctxStr := strings.TrimSpace(request.Context); ctxStr != "" {
		userContent = fmt.Sprintf("Context:\n%s\n\nRequest:\n%s", ctxStr, userContent)
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

	log.Printf("[diagramgen] generate begin model=%q base_url=%q prompt_len=%d context_len=%d",
		resolved.OpenAIModel, baseURL, len(request.Prompt), len(request.Context))

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
	log.Printf("[diagramgen] generate done model=%q mermaid_len=%d", resolved.OpenAIModel, len(mermaid))

	return GenerateResult{XML: mermaid}, nil
}

// extractMermaid 从模型输出中提取 Mermaid 代码。
// 模型有时会包裹 markdown 代码块（```mermaid ... ```），这里做容错处理。
func extractMermaid(content string) string {
	trimmed := strings.TrimSpace(content)

	// 去除 markdown 代码块包裹
	if strings.HasPrefix(trimmed, "```") {
		// 去掉开头的 ```mermaid 或 ```
		lines := strings.SplitN(trimmed, "\n", 2)
		if len(lines) > 1 {
			trimmed = strings.TrimSpace(lines[1])
		}
		// 去掉结尾的 ```
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
