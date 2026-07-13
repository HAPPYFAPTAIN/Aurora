package app

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"denova/internal/book"
	"denova/internal/materialindex"
)

// MaterialIndexAppService 负责资料卡片索引与搜索增强。
// 它将原先独立的 search-server 集成到主进程中，直接复用 App 的资料库和配置。
type MaterialIndexAppService struct {
	app  *App
	idx  *materialindex.Index
}

func (a *App) materialIndex() *MaterialIndexAppService {
	a.ensureServices()
	return a.materialIndexApp
}

// MaterialIndexSearch 执行资料卡片搜索。
func (a *App) MaterialIndexSearch(query, typeFilter string, limit int) materialindex.SearchResponse {
	return a.materialIndex().Search(query, typeFilter, limit)
}

// MaterialIndexStats 返回索引统计。
func (a *App) MaterialIndexStats() materialindex.StatsResponse {
	return a.materialIndex().Stats()
}

// MaterialIndexGetCard 根据 ID 获取卡片详情。
func (a *App) MaterialIndexGetCard(id string) (materialindex.Card, bool) {
	return a.materialIndex().GetCard(id)
}

// MaterialIndexRebuild 重建索引。
func (a *App) MaterialIndexRebuild() (int, error) {
	return a.materialIndex().Rebuild()
}

// MaterialIndexGenerate 调用 AI 从文本提炼卡片并写入资料库。
func (a *App) MaterialIndexGenerate(req materialindex.GenRequest) materialindex.GenResponse {
	return a.materialIndex().Generate(req)
}

// MaterialIndexListTemplates 返回可用模板列表。
func (a *App) MaterialIndexListTemplates() []map[string]string {
	return materialindex.ListTemplates()
}

// MaterialIndexImport 导入文本文件到 material-index/imports/ 目录。
func (a *App) MaterialIndexImport(filename string, data []byte) (string, int, error) {
	return a.materialIndex().Import(filename, data)
}

// MaterialIndexListImports 列出已导入的文件。
func (a *App) MaterialIndexListImports() []map[string]interface{} {
	return a.materialIndex().ListImports()
}

// MaterialIndexDeleteImport 删除已导入的文件。
func (a *App) MaterialIndexDeleteImport(filename string) error {
	return a.materialIndex().DeleteImport(filename)
}

// MaterialIndexWorkspaceSearch 通过 BookService 搜索工作区文件。
func (a *App) MaterialIndexWorkspaceSearch(query string) ([]book.SearchResult, error) {
	return a.materialIndex().WorkspaceSearch(query)
}

// === Service methods ===

func (s *MaterialIndexAppService) ensureIndex() {
	if s.idx == nil {
		s.idx = materialindex.NewIndex()
	}
}

func (s *MaterialIndexAppService) importsDir() string {
	ws := s.app.Workspace()
	if ws == "" {
		return ""
	}
	dir := filepath.Join(ws, "material-index", "imports")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

func (s *MaterialIndexAppService) templateDir() string {
	ws := s.app.Workspace()
	if ws == "" {
		return ""
	}
	return filepath.Join(ws, "skills", "material-index", "templates")
}

// Build 构建索引。
func (s *MaterialIndexAppService) Build() (int, error) {
	s.ensureIndex()

	items, err := s.app.LoreItems()
	if err != nil {
		log.Printf("[material-index] 获取资料库失败: %v", err)
		return 0, err
	}

	dir := s.importsDir()
	count := s.idx.BuildFromLoreItems(items, dir)
	log.Printf("[material-index] 索引完成: %d 张卡片", count)
	return count, nil
}

// Rebuild 重建索引。
func (s *MaterialIndexAppService) Rebuild() (int, error) {
	return s.Build()
}

// Search 搜索卡片。
func (s *MaterialIndexAppService) Search(query, typeFilter string, limit int) materialindex.SearchResponse {
	s.ensureIndex()
	results := s.idx.Search(query, typeFilter, limit)
	return materialindex.SearchResponse{
		Query:      query,
		Total:      len(results),
		Results:    results,
		TypeFilter: typeFilter,
	}
}

// Stats 返回统计信息。
func (s *MaterialIndexAppService) Stats() materialindex.StatsResponse {
	s.ensureIndex()
	return s.idx.Stats()
}

// GetCard 根据 ID 获取卡片。
func (s *MaterialIndexAppService) GetCard(id string) (materialindex.Card, bool) {
	s.ensureIndex()
	return s.idx.GetCard(id)
}

// Generate 调用 AI 生成卡片并写入资料库。
func (s *MaterialIndexAppService) Generate(req materialindex.GenRequest) materialindex.GenResponse {
	cfg := s.aiConfig()
	if cfg.APIKey == "" {
		return materialindex.GenResponse{Success: false, Error: "API Key 未配置"}
	}

	resp := materialindex.GenerateCards(cfg, req, s.templateDir())
	if !resp.Success {
		return resp
	}

	// 将生成的卡片写入资料库
	for i := range resp.Cards {
		card := &resp.Cards[i]
		enabled := true
		input := book.LoreItemInput{
			Enabled:          &enabled,
			Type:             card.Type,
			Name:             card.Name,
			Importance:       card.Importance,
			Tags:             card.Tags,
			BriefDescription: card.Brief,
			Keywords:         card.Keywords,
			LoadMode:         book.LoreLoadModeAuto,
			Content:          card.Content,
		}
		_, err := s.app.CreateLoreItem(input)
		if err != nil {
			log.Printf("[material-index] 写入资料库失败 (%s): %v", card.Name, err)
		}
	}

	return resp
}

// Import 导入文本文件。
func (s *MaterialIndexAppService) Import(filename string, data []byte) (string, int, error) {
	dir := s.importsDir()
	if dir == "" {
		return "", 0, ErrNoWorkspace
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".md" && ext != ".txt" {
		filename += ".md"
	}
	savePath := filepath.Join(dir, filename)
	if _, err := os.Stat(savePath); err == nil {
		name := strings.TrimSuffix(filename, filepath.Ext(filename))
		savePath = filepath.Join(dir, name+"_"+time.Now().Format("20060102_150405")+filepath.Ext(filename))
	}
	if err := os.WriteFile(savePath, data, 0644); err != nil {
		return "", 0, err
	}
	return filepath.Base(savePath), len(data), nil
}

// ListImports 列出导入的文件。
func (s *MaterialIndexAppService) ListImports() []map[string]interface{} {
	dir := s.importsDir()
	if dir == "" {
		return nil
	}
	var files []map[string]interface{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		info, _ := e.Info()
		files = append(files, map[string]interface{}{
			"name": e.Name(), "size": info.Size(), "time": info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}
	return files
}

// DeleteImport 删除导入的文件。
func (s *MaterialIndexAppService) DeleteImport(filename string) error {
	dir := s.importsDir()
	if dir == "" {
		return ErrNoWorkspace
	}
	target := filepath.Join(dir, filepath.Base(filename))
	return os.Remove(target)
}

// WorkspaceSearch 搜索工作区文件。
func (s *MaterialIndexAppService) WorkspaceSearch(query string) ([]book.SearchResult, error) {
	bs := s.app.BookService()
	if bs == nil {
		return nil, ErrNoWorkspace
	}
	return bs.Search(query, book.DefaultSearchLimit)
}

// aiConfig 从 App 配置中提取 AI 调用所需的配置。
func (s *MaterialIndexAppService) aiConfig() materialindex.AIConfig {
	s.app.mu.RLock()
	defer s.app.mu.RUnlock()

	if s.app.cfg == nil {
		return materialindex.AIConfig{}
	}
	cfg := s.app.cfg

	// 优先使用顶层配置，其次从 default model profile 获取
	apiKey := cfg.OpenAIAPIKey
	baseURL := cfg.OpenAIBaseURL
	model := cfg.OpenAIModel

	if apiKey == "" || baseURL == "" || model == "" {
		for _, p := range cfg.ModelProfiles {
			if p.ID == "default" || (p.ID == "" && apiKey == "") {
				if apiKey == "" {
					apiKey = p.OpenAIAPIKey
				}
				if baseURL == "" {
					baseURL = p.OpenAIBaseURL
				}
				if model == "" {
					model = p.OpenAIModel
				}
				break
			}
		}
	}

	return materialindex.AIConfig{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
	}
}

// EnsureMaterialIndexBuilt 确保索引已构建（惰性初始化）。
func (s *MaterialIndexAppService) EnsureBuilt() {
	s.ensureIndex()
	if s.idx.Stats().Total == 0 {
		_, _ = s.Build()
	}
}

// MaterialIndexDir 返回 material-index 工作目录路径。
func (a *App) MaterialIndexDir() string {
	ws := a.Workspace()
	if ws == "" {
		return ""
	}
	return filepath.Join(ws, "material-index")
}

// EnsureMaterialIndexBuilt 确保索引已构建。
func (a *App) EnsureMaterialIndexBuilt() {
	a.materialIndex().EnsureBuilt()
}
