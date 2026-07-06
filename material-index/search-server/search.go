package main

import (
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type SearchResult struct {
	Card    Card     `json:"card"`
	Score   float64  `json:"score"`
	Matches []string `json:"matches,omitempty"`
}

type SearchResponse struct {
	Query    string         `json:"query"`
	Total    int            `json:"total"`
	Results  []SearchResult `json:"results"`
	TypeFilter string       `json:"typeFilter,omitempty"`
}

// handleSearch 搜索卡片
func (idx *Index) handleSearch(cfg *ModuleConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		typeFilter := strings.TrimSpace(r.URL.Query().Get("type"))
		limit := 50
		if l := r.URL.Query().Get("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 500 {
				limit = n
			}
		}

		if query == "" && typeFilter == "" {
			writeJSON(w, 200, SearchResponse{Query: "", Results: []SearchResult{}})
			return
		}

		var results []SearchResult
		if query == "" {
			results = idx.filterOnly(typeFilter, limit)
		} else {
			results = idx.search(query, typeFilter, limit)
		}

		writeJSON(w, 200, SearchResponse{
			Query:     query,
			Total:     len(results),
			Results:   results,
			TypeFilter: typeFilter,
		})
	}
}

// handleCardDetail 获取单张卡片详情
func (idx *Index) handleCardDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/card/"))
	if id == "" {
		writeJSONError(w, 400, "缺少卡片ID")
		return
	}
	cardIdx, ok := idx.byID[id]
	if !ok {
		writeJSONError(w, 404, "卡片不存在: "+id)
		return
	}
	writeJSON(w, 200, idx.cards[cardIdx])
}

// handleStats 统计信息
func (idx *Index) handleStats(w http.ResponseWriter, r *http.Request) {
	typeStats := make(map[string]int)
	for t, idxs := range idx.byType {
		label := typeLabels[t]
		if label == "" {
			label = t
		}
		typeStats[label] += len(idxs)
	}

	writeJSON(w, 200, map[string]interface{}{
		"total":  len(idx.cards),
		"byType": typeStats,
	})
}

// handleWorkspaceSearch 通过 Aurora API 搜索工作区文件（章节、设定等）
func (idx *Index) handleWorkspaceSearch(cfg *ModuleConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		if query == "" {
			writeJSONError(w, 400, "缺少查询参数 q")
			return
		}

		url := strings.TrimSuffix(cfg.AuroraAPI, "/") + "/api/workspace/search?q=" + query
		resp, err := http.Get(url)
		if err != nil {
			writeJSONError(w, 502, "无法连接 Aurora API: "+err.Error())
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
	}
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

func (idx *Index) search(query, typeFilter string, limit int) []SearchResult {
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
	titleText := card.Title
	briefText := card.Brief
	count := strings.Count(text, term)
	titleCount := strings.Count(titleText, term)
	briefCount := strings.Count(briefText, term)
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

// passThrough 透传 JSON 响应
func passThrough(resp *http.Response, w http.ResponseWriter) {
	body, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

// jsonBody 读取 JSON body
func jsonBody(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}
