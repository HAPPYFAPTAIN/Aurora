package materialindex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TemplateMeta 描述可用卡片模板。
var TemplateMeta = map[string]struct{ Name, Desc string }{
	"auto":      {"自动识别", "让 AI 自动判断最合适的卡片类型"},
	"character": {"人物卡", "角色身份、人设、背景、能力"},
	"event":     {"事件卡", "重要事件、剧情节点"},
	"location":  {"地点卡", "长期反复出现的地点"},
	"world":     {"世界观卡", "世界类型、时代、秩序"},
	"faction":   {"势力卡", "组织、阵营、利益关系"},
	"rule":      {"规则卡", "能力体系、世界规则、禁忌"},
	"item":      {"物品卡", "关键物品、道具、线索"},
	"concept":   {"概念卡", "核心概念、主题"},
	"analysis":  {"分析卡", "素材分析、解读"},
}

// GenRequest AI 卡片生成请求。
type GenRequest struct {
	Text         string `json:"text"`
	TemplateID   string `json:"templateId"`
	CustomPrompt string `json:"customPrompt"`
}

// GenCard AI 生成的单张卡片。
type GenCard struct {
	Type       string   `json:"type"`
	Name       string   `json:"name"`
	Brief      string   `json:"brief"`
	Content    string   `json:"content"`
	Tags       []string `json:"tags"`
	Keywords   []string `json:"keywords"`
	Importance string   `json:"importance"`
}

// GenResponse AI 卡片生成响应。
type GenResponse struct {
	Success     bool      `json:"success"`
	Cards       []GenCard `json:"cards,omitempty"`
	Error       string    `json:"error,omitempty"`
	RawResponse string    `json:"rawResponse,omitempty"`
}

// AIConfig 提供 AI 调用所需的配置。
type AIConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

// ListTemplates 返回模板列表。
func ListTemplates() []map[string]string {
	var templates []map[string]string
	for id, meta := range TemplateMeta {
		templates = append(templates, map[string]string{
			"id": id, "name": meta.Name, "desc": meta.Desc,
		})
	}
	return templates
}

// GenerateCards 调用 AI 从文本提炼卡片。
// templateDir 为卡片模板目录路径（如 skills/material-index/templates/）。
func GenerateCards(cfg AIConfig, req GenRequest, templateDir string) GenResponse {
	if cfg.APIKey == "" {
		return GenResponse{Success: false, Error: "API Key 未配置"}
	}
	if strings.TrimSpace(req.Text) == "" {
		return GenResponse{Success: false, Error: "文本内容不能为空"}
	}

	// 加载模板
	templateContent := ""
	if req.TemplateID != "" && req.TemplateID != "auto" {
		templatePath := filepath.Join(templateDir, req.TemplateID+".md")
		if data, err := os.ReadFile(templatePath); err == nil {
			templateContent = string(data)
		}
	}

	systemPrompt := buildSystemPrompt(req.TemplateID, templateContent, req.CustomPrompt)
	userPrompt := buildUserPrompt(req.Text, req.TemplateID)

	aiResp, err := callOpenAI(cfg, systemPrompt, userPrompt)
	if err != nil {
		return GenResponse{Success: false, Error: "AI 调用失败: " + err.Error()}
	}

	cards, rawResp, err := parseAIResponse(aiResp)
	if err != nil {
		return GenResponse{Success: false, Error: "AI 返回解析失败: " + err.Error(), RawResponse: aiResp}
	}
	_ = rawResp
	return GenResponse{Success: true, Cards: cards, RawResponse: aiResp}
}

func buildSystemPrompt(templateID, templateContent, customPrompt string) string {
	var sb strings.Builder
	sb.WriteString("你是一个资料卡片提炼专家。阅读用户提供的文本资料，按模板提炼出结构化知识卡片。\n\n")
	sb.WriteString("## 输出格式\n\n输出 JSON 数组：\n```json\n")
	sb.WriteString(`[
  {
    "type": "character",
    "name": "卡片标题",
    "brief": "类型 名称。3-5句简要说明。上下文出现相关内容时，一定要参考本项详情。",
    "content": "Markdown 格式卡片正文",
    "tags": ["标签"],
    "keywords": ["关键词"],
    "importance": "important"
  }
]`)
	sb.WriteString("\n```\n\n")
	sb.WriteString("## 类型\ncharacter/event/location/world/faction/rule/item/concept/analysis\n\n")
	if templateContent != "" {
		sb.WriteString("## 模板\n\n")
		sb.WriteString(templateContent)
		sb.WriteString("\n\n")
	}
	sb.WriteString("## 原则\n1. 忠于原文，不臆造\n2. 每张卡片聚焦一个主题\n3. brief 以\"类型 名称。\"开头\n4. keywords 包含名称、别名、关键特征\n5. importance: important 或 minor\n6. 多主题拆分为多张卡片\n")
	if customPrompt != "" {
		sb.WriteString("\n## 额外要求\n\n")
		sb.WriteString(customPrompt)
	}
	sb.WriteString("\n只输出 JSON 数组。")
	return sb.String()
}

func buildUserPrompt(text, templateID string) string {
	var sb strings.Builder
	if templateID == "auto" || templateID == "" {
		sb.WriteString("请阅读以下文本，自动判断类型并提炼卡片：\n\n")
	} else {
		sb.WriteString(fmt.Sprintf("请按【%s】模板提炼卡片：\n\n", TemplateMeta[templateID].Name))
	}
	sb.WriteString("---\n\n")
	sb.WriteString(text)
	sb.WriteString("\n\n---\n")
	return sb.String()
}

func callOpenAI(cfg AIConfig, systemPrompt, userPrompt string) (string, error) {
	reqBody := map[string]interface{}{
		"model": cfg.Model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0.3,
		"max_tokens":  8192,
	}
	jsonBody, _ := json.Marshal(reqBody)

	url := strings.TrimSuffix(cfg.BaseURL, "/") + "/v1/chat/completions"
	req, _ := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API %d: %s", resp.StatusCode, string(body))
	}

	var aiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	json.Unmarshal(body, &aiResp)
	if len(aiResp.Choices) == 0 {
		return "", fmt.Errorf("AI 未返回内容")
	}
	return aiResp.Choices[0].Message.Content, nil
}

func parseAIResponse(raw string) ([]GenCard, string, error) {
	jsonStr := raw
	if idx := strings.Index(raw, "```json"); idx >= 0 {
		jsonStr = raw[idx+7:]
		if end := strings.Index(jsonStr, "```"); end >= 0 {
			jsonStr = jsonStr[:end]
		}
	} else if idx := strings.Index(raw, "```"); idx >= 0 {
		jsonStr = raw[idx+3:]
		if end := strings.Index(jsonStr, "```"); end >= 0 {
			jsonStr = jsonStr[:end]
		}
	}
	jsonStr = strings.TrimSpace(jsonStr)
	start := strings.Index(jsonStr, "[")
	end := strings.LastIndex(jsonStr, "]")
	if start >= 0 && end > start {
		jsonStr = jsonStr[start : end+1]
	}

	var cards []GenCard
	if err := json.Unmarshal([]byte(jsonStr), &cards); err != nil {
		return nil, raw, err
	}
	return cards, raw, nil
}
