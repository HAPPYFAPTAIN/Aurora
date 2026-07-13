package materialindex

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode"

	"denova/internal/book"
)

// CardTypeLabel 将资料库 type 编码映射为中文标签。
var CardTypeLabel = map[string]string{
	"character": "人物卡",
	"world":     "世界观卡",
	"location":  "地点卡",
	"faction":   "势力卡",
	"rule":      "规则卡",
	"item":      "物品卡",
	"other":     "其他卡",
}

// Card 是索引中的单张卡片，由 LoreItem 转换而来。
type Card struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	TypeLabel   string   `json:"typeLabel,omitempty"`
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Brief       string   `json:"brief,omitempty"`
	Content     string   `json:"content"`
	Keywords    []string `json:"keywords,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Importance  string   `json:"importance,omitempty"`
	LoadMode    string   `json:"loadMode,omitempty"`
}

// SearchResult 表示一条搜索结果。
type SearchResult struct {
	Card    Card     `json:"card"`
	Score   float64  `json:"score"`
	Matches []string `json:"matches,omitempty"`
}

// SearchResponse 是搜索 API 的完整响应。
type SearchResponse struct {
	Query      string         `json:"query"`
	Total      int            `json:"total"`
	Results    []SearchResult `json:"results"`
	TypeFilter string         `json:"typeFilter,omitempty"`
}

// StatsResponse 统计信息。
type StatsResponse struct {
	Total  int            `json:"total"`
	ByType map[string]int `json:"byType"`
}

// Index 维护资料库卡片的 n-gram 全文索引，支持并发安全搜索。
type Index struct {
	mu       sync.RWMutex
	cards    []Card
	byType   map[string][]int
	byID     map[string]int
	ngramIdx map[string]map[int]struct{}
}

// NewIndex 创建空索引。
func NewIndex() *Index {
	return &Index{
		byType:   make(map[string][]int),
		byID:     make(map[string]int),
		ngramIdx: make(map[string]map[int]struct{}),
	}
}

// BuildFromLoreItems 从 LoreItem 列表构建索引，同时索引 importsDir 中的文本文件。
func (idx *Index) BuildFromLoreItems(items []book.LoreItem, importsDir string) int {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.cards = nil
	idx.byType = make(map[string][]int)
	idx.byID = make(map[string]int)
	idx.ngramIdx = make(map[string]map[int]struct{})

	for _, item := range items {
		if !item.Enabled {
			continue
		}
		idx.addLoreItem(item)
	}

	idx.indexImportFiles(importsDir)
	idx.buildNgrams()
	return len(idx.cards)
}

// Rebuild 清空并重建索引。
func (idx *Index) Rebuild(items []book.LoreItem, importsDir string) int {
	return idx.BuildFromLoreItems(items, importsDir)
}

func (idx *Index) addLoreItem(item book.LoreItem) {
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
	if label, ok := CardTypeLabel[item.Type]; ok {
		card.TypeLabel = label
	}

	cardIdx := len(idx.cards)
	idx.cards = append(idx.cards, card)
	idx.byID[card.ID] = cardIdx
	if card.Type != "" {
		idx.byType[card.Type] = append(idx.byType[card.Type], cardIdx)
	}
}

func (idx *Index) indexImportFiles(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return
	}
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
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
}

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

// Search 执行搜索，返回评分排序后的结果。
func (idx *Index) Search(query, typeFilter string, limit int) []SearchResult {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	if query == "" && typeFilter == "" {
		return nil
	}

	if query == "" {
		return idx.filterOnly(typeFilter, limit)
	}

	terms := tokenize(query)
	if len(terms) == 0 {
		return nil
	}

	scores := make(map[int]float64)
	matches := make(map[int]map[string]bool)

	for _, term := range terms {
		hits := idx.matchTerm(term)
		for cardIdx, score := range hits {
			scores[cardIdx] += score
			if matches[cardIdx] == nil {
				matches[cardIdx] = make(map[string]bool)
			}
			matches[cardIdx][term] = true
		}
	}

	var results []SearchResult
	for cardIdx, score := range scores {
		card := idx.cards[cardIdx]
		if typeFilter != "" && card.Type != typeFilter && card.TypeLabel != typeFilter {
			continue
		}
		var matchList []string
		for m := range matches[cardIdx] {
			matchList = append(matchList, m)
		}
		results = append(results, SearchResult{
			Card:    card,
			Score:   score,
			Matches: matchList,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > limit {
		results = results[:limit]
	}
	return results
}

func (idx *Index) filterOnly(typeFilter string, limit int) []SearchResult {
	var results []SearchResult
	for _, card := range idx.cards {
		if typeFilter != "" && card.Type != typeFilter && card.TypeLabel != typeFilter {
			continue
		}
		results = append(results, SearchResult{Card: card, Score: 1})
		if len(results) >= limit {
			break
		}
	}
	return results
}

func (idx *Index) matchTerm(term string) map[int]float64 {
	result := make(map[int]float64)
	runes := []rune(term)

	switch {
	case len(runes) == 1:
		for i, card := range idx.cards {
			text := card.Title + card.Brief + card.Content + strings.Join(card.Keywords, " ")
			if strings.ContainsRune(text, runes[0]) {
				result[i] = idx.scoreMatch(card, term)
			}
		}
	case len(runes) == 2:
		for i, card := range idx.cards {
			text := card.Title + card.Brief + card.Content + strings.Join(card.Keywords, " ")
			if strings.Contains(text, term) {
				result[i] = idx.scoreMatch(card, term)
			}
		}
	default:
		ngrams := extractNgrams(term, 3)
		if len(ngrams) == 0 {
			break
		}
		candidateCount := make(map[int]int)
		for _, ng := range ngrams {
			if cardSet, ok := idx.ngramIdx[ng]; ok {
				for cardIdx := range cardSet {
					candidateCount[cardIdx]++
				}
			}
		}
		for cardIdx, count := range candidateCount {
			if count >= len(ngrams) {
				card := idx.cards[cardIdx]
				text := card.Title + card.Brief + card.Content + strings.Join(card.Keywords, " ")
				if strings.Contains(text, term) {
					result[cardIdx] = idx.scoreMatch(card, term)
				}
			}
		}
	}
	return result
}

func (idx *Index) scoreMatch(card Card, term string) float64 {
	text := card.Title + " " + card.Brief + " " + card.Content + " " + strings.Join(card.Keywords, " ")
	count := strings.Count(text, term)
	titleCount := strings.Count(card.Title, term)
	briefCount := strings.Count(card.Brief, term)
	return float64(count) + float64(titleCount)*4.0 + float64(briefCount)*2.0
}

func tokenize(query string) []string {
	var terms []string
	var current []rune
	for _, r := range query {
		if unicode.IsSpace(r) {
			if len(current) > 0 {
				terms = append(terms, string(current))
				current = nil
			}
			continue
		}
		current = append(current, r)
	}
	if len(current) > 0 {
		terms = append(terms, string(current))
	}
	return terms
}

// Stats 返回索引统计信息。
func (idx *Index) Stats() StatsResponse {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	typeStats := make(map[string]int)
	for t, idxs := range idx.byType {
		label := CardTypeLabel[t]
		if label == "" {
			label = t
		}
		typeStats[label] += len(idxs)
	}
	return StatsResponse{
		Total:  len(idx.cards),
		ByType: typeStats,
	}
}

// GetCard 根据 ID 获取单张卡片。
func (idx *Index) GetCard(id string) (Card, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	cardIdx, ok := idx.byID[id]
	if !ok {
		return Card{}, false
	}
	return idx.cards[cardIdx], true
}
