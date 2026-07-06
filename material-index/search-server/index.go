package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// 卡片类型标签
var typeLabels = map[string]string{
	"character": "人物卡",
	"world":     "世界观卡",
	"location":  "地点卡",
	"faction":   "势力卡",
	"rule":      "规则卡",
	"item":      "物品卡",
	"other":     "其他卡",
}

// Aurora lore item 结构
type LoreItem struct {
	ID              string   `json:"id"`
	Enabled         bool     `json:"enabled"`
	Type            string   `json:"type"`
	Name            string   `json:"name"`
	Importance      string   `json:"importance"`
	Tags            []string `json:"tags"`
	BriefDescription string   `json:"brief_description"`
	Keywords        []string `json:"keywords"`
	LoadMode        string   `json:"load_mode"`
	Content         string   `json:"content"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

type Index struct {
	cards    []Card
	byType   map[string][]int
	byID     map[string]int
	ngramIdx map[string]map[int]struct{}
}

type Card struct {
	ID        string   `json:"id"`
	Type      string   `json:"type"`
	TypeLabel string   `json:"typeLabel,omitempty"`
	Name      string   `json:"name"`
	Title     string   `json:"title"`
	Brief     string   `json:"brief,omitempty"`
	Content   string   `json:"content"`
	Keywords  []string `json:"keywords,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	Importance string   `json:"importance,omitempty"`
	LoadMode  string   `json:"loadMode,omitempty"`
}

func NewIndex() *Index {
	return &Index{
		byType:   make(map[string][]int),
		byID:     make(map[string]int),
		ngramIdx: make(map[string]map[int]struct{}),
	}
}

// fetchLoreItems 从 Aurora API 获取资料库数据
func fetchLoreItems(AuroraAPI string) ([]LoreItem, error) {
	url := strings.TrimSuffix(AuroraAPI, "/") + "/api/lore/items"
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, log.Output(2, "Aurora API 返回 "+resp.Status+": "+string(body))
	}

	var result struct {
		Items []LoreItem `json:"items"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result.Items, nil
}

// Build 从 Aurora API 构建索引
func (idx *Index) Build(cfg *ModuleConfig) (int, error) {
	items, err := fetchLoreItems(cfg.AuroraAPI)
	if err != nil {
		return 0, err
	}

	for _, item := range items {
		if !item.Enabled {
			continue
		}
		idx.addLoreItem(item)
	}

	// 也索引导入目录中的文件
	importsDir := filepath.Join(cfg.Workspace, "material-index", "imports")
	idx.indexImportFiles(importsDir)

	idx.buildNgrams()
	return len(idx.cards), nil
}

func (idx *Index) addLoreItem(item LoreItem) {
	card := Card{
		ID:         item.ID,
		Type:       item.Type,
		Name:       item.Name,
		Title:      item.Name,
		Brief:      item.BriefDescription,
		Content:    item.Content,
		Keywords:   item.Keywords,
		Tags:       item.Tags,
		Importance: item.Importance,
		LoadMode:   item.LoadMode,
	}
	if label, ok := typeLabels[item.Type]; ok {
		card.TypeLabel = label
	}

	cardIdx := len(idx.cards)
	idx.cards = append(idx.cards, card)
	idx.byID[card.ID] = cardIdx
	if card.Type != "" {
		idx.byType[card.Type] = append(idx.byType[card.Type], cardIdx)
	}
}

// indexImportFiles 索引导入的文本文件
func (idx *Index) indexImportFiles(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return
	}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" && ext != ".txt" {
			return nil
		}
		if strings.HasPrefix(filepath.Base(path), ".") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(data)
		title := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))

		card := Card{
			ID:        "import-" + sanitizeID(title),
			Type:      "import",
			TypeLabel: "导入资料",
			Name:      title,
			Title:     title,
			Content:   content,
			Tags:      []string{"导入"},
		}
		cardIdx := len(idx.cards)
		idx.cards = append(idx.cards, card)
		idx.byID[card.ID] = cardIdx
		idx.byType["import"] = append(idx.byType["import"], cardIdx)
		return nil
	})
	if err != nil {
		log.Printf("索引导入文件失败: %v", err)
	}
}

// buildNgrams 构建字符 n-gram 索引
func (idx *Index) buildNgrams() {
	for i, card := range idx.cards {
		text := card.Title + " " + card.Brief + " " + card.Content + " " + strings.Join(card.Keywords, " ") + " " + strings.Join(card.Tags, " ")
		for _, ng := range extractNgrams(text, 3) {
			if idx.ngramIdx[ng] == nil {
				idx.ngramIdx[ng] = make(map[int]struct{})
			}
			idx.ngramIdx[ng][i] = struct{}{}
		}
	}
}

func extractNgrams(text string, n int) []string {
	var runes []rune
	for _, r := range text {
		if !unicode.IsSpace(r) {
			runes = append(runes, r)
		}
	}
	if len(runes) < n {
		if len(runes) > 0 {
			return []string{string(runes)}
		}
		return nil
	}
	seen := make(map[string]bool)
	var result []string
	for i := 0; i <= len(runes)-n; i++ {
		ng := string(runes[i : i+n])
		if !seen[ng] {
			seen[ng] = true
			result = append(result, ng)
		}
	}
	return result
}

func sanitizeID(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")
	return s
}

// matchRegexp 用于安全提取
var cardHeaderRe = regexp.MustCompile(`(?m)^###\s+`)
