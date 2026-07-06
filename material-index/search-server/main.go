package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

//go:embed dist
var distFS embed.FS

var (
	cfg *ModuleConfig
	idx *Index
)

func main() {
	configPath := flag.String("config", "", "模块配置文件路径")
	flag.Parse()

	cp := *configPath
	if cp == "" {
		cp = filepath.Join(filepath.Dir(getExecutableDir()), "config.toml")
		if _, err := os.Stat(cp); err != nil {
			cp = "config.toml"
		}
	}

	cfg = LoadConfig(cp)
	logConfig(cfg)

	if !cfg.Enabled {
		log.Println("模块已禁用。设置 enabled = true 启用。")
		return
	}

	os.MkdirAll(filepath.Join(cfg.Workspace, "material-index", "imports"), 0755)

	// 构建索引
	idx = NewIndex()
	count, err := idx.Build(cfg)
	if err != nil {
		log.Printf("警告: 无法连接 Aurora API (%s)，请确保 Aurora 正在运行。索引为空。", cfg.AuroraAPI)
		count = 0
	}
	log.Printf("索引完成: %d 张卡片", count)

	mux := http.NewServeMux()

	// 搜索 API
	mux.HandleFunc("/api/search", idx.handleSearch(cfg))
	mux.HandleFunc("/api/stats", idx.handleStats)
	mux.HandleFunc("/api/card/", idx.handleCardDetail)

	// 模板 API
	mux.HandleFunc("/api/templates", idx.handleListTemplates)

	// AI 生成 API
	mux.HandleFunc("/api/generate", idx.handleGenerate(cfg))

	// 文件管理 API
	mux.HandleFunc("/api/import", idx.handleImport(cfg))
	mux.HandleFunc("/api/imports", idx.handleListImports(cfg))
	mux.HandleFunc("/api/delete-import", idx.handleDeleteImport(cfg))

	// 资料库操作 API（通过 Aurora API）
	mux.HandleFunc("/api/lore/delete", idx.handleDeleteCard(cfg))
	mux.HandleFunc("/api/lore/update", idx.handleUpdateCard(cfg))

	// workspace 搜索代理
	mux.HandleFunc("/api/workspace-search", idx.handleWorkspaceSearch(cfg))

	// 索引管理
	mux.HandleFunc("/api/rebuild", idx.handleRebuild(cfg))

	// 配置
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]interface{}{
			"workspace":    cfg.Workspace,
			"port":         cfg.Port,
			"AuroraAPI":   cfg.AuroraAPI,
			"model":        cfg.OpenAIModel,
			"apiConfigured": cfg.OpenAIKey != "",
			"enabled":      cfg.Enabled,
		})
	})

	// 静态文件
	distSub, _ := fs.Sub(distFS, "dist")
	mux.Handle("/", http.FileServer(http.FS(distSub)))

	addr := ":" + cfg.Port
	log.Printf("搜索增强服务启动: http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func getExecutableDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error":%q}`, strings.ReplaceAll(msg, `"`, `\"`))
}
