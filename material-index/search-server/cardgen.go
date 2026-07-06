package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GenRequest AI 卡片生成请求
type GenRequest struct {
	Text         string `json:"text"`
	TemplateID   string `json:"templateId"`
	CustomPrompt string `json:"customPrompt"`
}

// GenCard AI 生成的卡片
type GenCard struct {
	Type       string   `json:"type"`
	Name       string   `json:"name"`
	Brief      string   `json:"brief"`
	Content    string   `json:"content"`
	Tags       []string `json:"tags"`
	Keywords   []string `json:"keywords"`
	Importance string   `json:"importance"`
}

// GenResponse AI 卡片生成响应
type GenResponse struct {
	Success     bool      `json:"success"`
	Cards       []GenCard `json:"cards,omitempty"`
	Error       string    `json:"error,omitempty"`
	RawResponse string    `json:"rawResponse,omitempty"`
}

var templateMeta = map[string]struct{ Name, Desc string }{
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

// handleListTemplates 列出模板
func (idx *Index) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	var templates []map[string]string
	for id, meta := range templateMeta {
		templates = append(templates, map[string]string{
			"id": id, "name": meta.Name, "desc": meta.Desc,
		})
	}
	writeJSON(w, 200, map[string]interface{}{"templates": templates})
}

// handleGenerate AI 生成卡片并写入 Aurora 资料库
func (idx *Index) handleGenerate(cfg *ModuleConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSONError(w, 405, "仅支持 POST")
			return
		}
		if cfg.OpenAIKey == "" {
			writeJSONError(w, 400, "API Key 未配置")
			return
		}

		var req GenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, 400, "请求解析失败: "+err.Error())
			return
		}
		if strings.TrimSpace(req.Text) == "" {
			writeJSONError(w, 400, "文本内容不能为空")
			return
		}

		// 加载模板
		templateContent := ""
		if req.TemplateID != "" && req.TemplateID != "auto" {
			templatePath := filepath.Join(cfg.Workspace, "skills", "material-index", "templates", req.TemplateID+".md")
			if data, err := os.ReadFile(templatePath); err == nil {
				templateContent = string(data)
			}
		}

		// 调用 AI
		systemPrompt := buildSystemPrompt(req.TemplateID, templateContent, req.CustomPrompt)
		userPrompt := buildUserPrompt(req.Text, req.TemplateID)

		aiResp, err := callOpenAI(cfg, systemPrompt, userPrompt)
		if err != nil {
			writeJSONError(w, 500, "AI 调用失败: "+err.Error())
			return
		}

		cards, rawResp, err := parseAIResponse(aiResp)
		if err != nil {
			writeJSON(w, 200, GenResponse{
				Success: false, Error: "AI 返回解析失败: " + err.Error(), RawResponse: aiResp,
			})
			return
		}

		// 通过 Aurora API 写入资料库
		for i := range cards {
			err := createLoreItem(cfg, &cards[i])
			if err != nil {
				log.Printf("写入资料库失败 (%s): %v", cards[i].Name, err)
			}
		}

		writeJSON(w, 200, GenResponse{Success: true, Cards: cards, RawResponse: rawResp})
	}
}

// createLoreItem 通过 Aurora API 创建资料库条目
func createLoreItem(cfg *ModuleConfig, card *GenCard) error {
	now := time.Now().Format(time.RFC3339)
	item := map[string]interface{}{
		"id":                "",
		"enabled":           true,
		"type":              card.Type,
		"name":              card.Name,
		"importance":        card.Importance,
		"tags":              card.Tags,
		"brief_description": card.Brief,
		"keywords":          card.Keywords,
		"load_mode":         "auto",
		"content":           card.Content,
		"created_at":        now,
		"updated_at":        now,
	}

	body, _ := json.Marshal(item)
	url := strings.TrimSuffix(cfg.AuroraAPI, "/") + "/api/lore/items"
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("API 返回 %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// updateLoreItem 通过 Aurora API 更新资料库条目
func updateLoreItem(cfg *ModuleConfig, id string, card *GenCard) error {
	now := time.Now().Format(time.RFC3339)
	item := map[string]interface{}{
		"name":               card.Name,
		"type":               card.Type,
		"importance":         card.Importance,
		"tags":               card.Tags,
		"brief_description":  card.Brief,
		"keywords":           card.Keywords,
		"load_mode":          "auto",
		"content":            card.Content,
		"updated_at":         now,
	}

	body, _ := json.Marshal(item)
	url := strings.TrimSuffix(cfg.AuroraAPI, "/") + "/api/lore/items/" + id
	req, _ := http.NewRequest("PATCH", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API 返回 %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// deleteLoreItem 通过 Aurora API 删除资料库条目
func deleteLoreItem(cfg *ModuleConfig, id string) error {
	url := strings.TrimSuffix(cfg.AuroraAPI, "/") + "/api/lore/items/" + id
	req, _ := http.NewRequest("DELETE", url, nil)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API 返回 %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// handleImport 导入文本文件
func (idx *Index) handleImport(cfg *ModuleConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSONError(w, 405, "仅支持 POST")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)
		if err := r.ParseMultipartForm(10 * 1024 * 1024); err != nil {
			writeJSONError(w, 400, "文件解析失败: "+err.Error())
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			writeJSONError(w, 400, "未找到上传文件")
			return
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			writeJSONError(w, 500, "读取文件失败: "+err.Error())
			return
		}

		importsDir := filepath.Join(cfg.Workspace, "material-index", "imports")
		os.MkdirAll(importsDir, 0755)

		filename := header.Filename
		ext := strings.ToLower(filepath.Ext(filename))
		if ext != ".md" && ext != ".txt" {
			filename += ".md"
		}
		savePath := filepath.Join(importsDir, filename)
		if _, err := os.Stat(savePath); err == nil {
			name := strings.TrimSuffix(filename, filepath.Ext(filename))
			savePath = filepath.Join(importsDir, name+"_"+time.Now().Format("20060102_150405")+filepath.Ext(filename))
		}
		os.WriteFile(savePath, data, 0644)

		writeJSON(w, 200, map[string]interface{}{
			"success": true, "filename": filepath.Base(savePath), "size": len(data), "content": string(data),
		})
	}
}

// handleListImports 列出导入文件
func (idx *Index) handleListImports(cfg *ModuleConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		importsDir := filepath.Join(cfg.Workspace, "material-index", "imports")
		var files []map[string]interface{}
		entries, err := os.ReadDir(importsDir)
		if err == nil {
			for _, e := range entries {
				if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
					continue
				}
				info, _ := e.Info()
				files = append(files, map[string]interface{}{
					"name": e.Name(), "size": info.Size(), "time": info.ModTime().Format("2006-01-02 15:04:05"),
				})
			}
		}
		writeJSON(w, 200, map[string]interface{}{"files": files})
	}
}

// handleDeleteImport 删除导入文件
func (idx *Index) handleDeleteImport(cfg *ModuleConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Path string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, 400, "请求解析失败")
			return
		}
		importsDir := filepath.Join(cfg.Workspace, "material-index", "imports")
		target := filepath.Join(importsDir, filepath.Base(req.Path))
		if err := os.Remove(target); err != nil {
			writeJSONError(w, 500, "删除失败: "+err.Error())
			return
		}
		writeJSON(w, 200, map[string]interface{}{"success": true})
	}
}

// handleDeleteCard 通过 Aurora API 删除资料库条目
func (idx *Index) handleDeleteCard(cfg *ModuleConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, 400, "请求解析失败")
			return
		}
		if err := deleteLoreItem(cfg, req.ID); err != nil {
			writeJSONError(w, 500, "删除失败: "+err.Error())
			return
		}
		writeJSON(w, 200, map[string]interface{}{"success": true})
	}
}

// handleUpdateCard 通过 Aurora API 更新资料库条目
func (idx *Index) handleUpdateCard(cfg *ModuleConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSONError(w, 405, "仅支持 POST")
			return
		}
		var req struct {
			ID   string  `json:"id"`
			Card GenCard `json:"card"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, 400, "请求解析失败")
			return
		}
		if err := updateLoreItem(cfg, req.ID, &req.Card); err != nil {
			writeJSONError(w, 500, "更新失败: "+err.Error())
			return
		}
		writeJSON(w, 200, map[string]interface{}{"success": true})
	}
}

// handleRebuild 重建索引
func (idx *Index) handleRebuild(cfg *ModuleConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idx.cards = nil
		idx.byType = make(map[string][]int)
		idx.byID = make(map[string]int)
		idx.ngramIdx = make(map[string]map[int]struct{})

		count, err := idx.Build(cfg)
		if err != nil {
			writeJSONError(w, 500, "重建索引失败: "+err.Error())
			return
		}
		writeJSON(w, 200, map[string]interface{}{"success": true, "total": count})
	}
}

// ===== AI 调用 =====

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
		sb.WriteString(fmt.Sprintf("请按【%s】模板提炼卡片：\n\n", templateMeta[templateID].Name))
	}
	sb.WriteString("---\n\n")
	sb.WriteString(text)
	sb.WriteString("\n\n---\n")
	return sb.String()
}

func callOpenAI(cfg *ModuleConfig, systemPrompt, userPrompt string) (string, error) {
	reqBody := map[string]interface{}{
		"model": cfg.OpenAIModel,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0.3,
		"max_tokens":  8192,
	}
	jsonBody, _ := json.Marshal(reqBody)

	url := strings.TrimSuffix(cfg.OpenAIURL, "/") + "/v1/chat/completions"
	req, _ := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.OpenAIKey)

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
