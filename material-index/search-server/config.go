package main

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// ModuleConfig 模块配置
type ModuleConfig struct {
	Enabled     bool   `toml:"enabled"`
	Port        string `toml:"port"`
	Workspace   string `toml:"workspace"`
	AuroraDir   string `toml:"Aurora_dir"`
	AuroraAPI   string `toml:"Aurora_api_url"`
	OpenAIKey   string `toml:"openai_api_key"`
	OpenAIURL   string `toml:"openai_base_url"`
	OpenAIModel string `toml:"openai_model"`
	AutoIndex   bool   `toml:"auto_index"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *ModuleConfig {
	return &ModuleConfig{
		Enabled:     true,
		Port:        "8927",
		Workspace:   "",
		AuroraDir:   "",
		AuroraAPI:   "http://localhost:8080",
		OpenAIKey:   "",
		OpenAIURL:   "https://api.deepseek.com",
		OpenAIModel: "deepseek-chat",
		AutoIndex:   true,
	}
}

// LoadConfig 从配置文件加载配置，支持环境变量覆盖
func LoadConfig(configPath string) *ModuleConfig {
	cfg := DefaultConfig()

	// 尝试读取模块配置文件（简单 TOML 解析）
	parseSimpleTOML(configPath, func(key, value string) {
		setConfigField(cfg, key, value)
	})

	// 环境变量覆盖工作区
	if v := os.Getenv("Aurora_WORKSPACE"); v != "" && cfg.Workspace == "" {
		cfg.Workspace = v
	}

	// 自动检测工作区（必须在加载 Aurora 配置之前完成）
	if cfg.Workspace == "" {
		cfg.Workspace = findWorkspace()
	}
	if cfg.AuroraDir == "" {
		cfg.AuroraDir = filepath.Join(cfg.Workspace, ".nova")
	}

	// 从 Aurora 主配置读取模型配置
	AuroraConfigPath := filepath.Join(cfg.Workspace, "config.toml")
	if absConfigPath, _ := filepath.Abs(configPath); true {
		if absAurora, _ := filepath.Abs(AuroraConfigPath); absAurora != absConfigPath {
			if _, err := os.Stat(AuroraConfigPath); err == nil {
				loadAuroraModelConfig(cfg, AuroraConfigPath)
			}
		}
	}

	// 环境变量覆盖（最高优先级）
	if v := os.Getenv("MATERIAL_INDEX_ENABLED"); v == "false" || v == "0" {
		cfg.Enabled = false
	}
	if v := os.Getenv("MATERIAL_INDEX_PORT"); v != "" {
		cfg.Port = v
	}
	if v := os.Getenv("Aurora_API_URL"); v != "" {
		cfg.AuroraAPI = v
	}
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		cfg.OpenAIKey = v
	}
	if v := os.Getenv("OPENAI_BASE_URL"); v != "" {
		cfg.OpenAIURL = v
	}
	if v := os.Getenv("OPENAI_MODEL"); v != "" {
		cfg.OpenAIModel = v
	}

	return cfg
}

// setConfigField 根据键名设置配置字段
func setConfigField(cfg *ModuleConfig, key, value string) {
	switch key {
	case "enabled":
		cfg.Enabled = value == "true" || value == "1"
	case "port":
		cfg.Port = value
	case "workspace":
		cfg.Workspace = value
	case "Aurora_dir":
		cfg.AuroraDir = value
	case "Aurora_api_url":
		cfg.AuroraAPI = value
	case "openai_api_key":
		cfg.OpenAIKey = value
	case "openai_base_url":
		cfg.OpenAIURL = value
	case "openai_model":
		cfg.OpenAIModel = value
	case "auto_index":
		cfg.AutoIndex = value == "true" || value == "1"
	}
}

// loadAuroraModelConfig 从 Aurora config.toml 读取模型配置
func loadAuroraModelConfig(cfg *ModuleConfig, path string) {
	if cfg.OpenAIKey != "" {
		return
	}

	inModelProfiles := false
	inDefault := false

	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// 处理表头 [[xxx]] 和 [xxx]
		if strings.HasPrefix(line, "[[") {
			section := strings.TrimSuffix(strings.TrimPrefix(line, "[["), "]]")
			inModelProfiles = section == "model_profiles"
			inDefault = false
			continue
		}
		if strings.HasPrefix(line, "[") {
			section := strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			// 单独的表，如 [agent_models.xxx]
			inModelProfiles = false
			inDefault = false
			_ = section
			continue
		}
		// 只在 [[model_profiles]] 内读取
		if !inModelProfiles {
			continue
		}
		// 解析 key = value
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		value = strings.Trim(value, "\"'")
		if idx := strings.Index(value, " #"); idx >= 0 {
			value = strings.TrimSpace(value[:idx])
			value = strings.Trim(value, "\"'")
		}
		if key == "" {
			continue
		}
		// 跟踪 default profile
		if key == "id" {
			inDefault = value == "default"
			continue
		}
		if !inDefault {
			continue
		}
		switch key {
		case "openai_api_key":
			if cfg.OpenAIKey == "" {
				cfg.OpenAIKey = value
			}
		case "openai_base_url":
			if cfg.OpenAIURL == "" || cfg.OpenAIURL == "https://api.deepseek.com" {
				cfg.OpenAIURL = value
			}
		case "openai_model":
			if cfg.OpenAIModel == "" || cfg.OpenAIModel == "deepseek-chat" {
				cfg.OpenAIModel = value
			}
		}
	}

	// 读取 backend_port
	var backendPort string
	parseSimpleTOML(path, func(key, value string) {
		if key == "backend_port" {
			backendPort = value
		}
	})
	if backendPort != "" && cfg.AuroraAPI == "http://localhost:8080" {
		cfg.AuroraAPI = "http://localhost:" + backendPort
	}
}

// parseSimpleTOML 解析简单的 key = value TOML 格式
func parseSimpleTOML(path string, handler func(key, value string)) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		value = strings.Trim(value, "\"'")
		if idx := strings.Index(value, " #"); idx >= 0 {
			value = strings.TrimSpace(value[:idx])
			value = strings.Trim(value, "\"'")
		}
		if key != "" {
			handler(key, value)
		}
	}
}

// findWorkspace 自动检测 Aurora 工作区根目录
func findWorkspace() string {
	dir, _ := os.Getwd()
	for {
		base := filepath.Base(dir)
		if base == "material-index" || base == "search-server" {
			dir = filepath.Dir(dir)
			continue
		}
		break
	}
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(dir, ".nova")); err == nil {
			return dir
		}
		if _, err := os.Stat(filepath.Join(dir, "Aurora.exe")); err == nil {
			return dir
		}
		if _, err := os.Stat(filepath.Join(dir, "Aurora")); err == nil {
			return dir
		}
		if data, err := os.ReadFile(filepath.Join(dir, "config.toml")); err == nil {
			if strings.Contains(string(data), "[model_profiles]") || strings.Contains(string(data), "backend_port =") {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	dir, _ = os.Getwd()
	base := filepath.Base(dir)
	if base == "search-server" {
		return filepath.Dir(filepath.Dir(dir))
	}
	if base == "material-index" {
		return filepath.Dir(dir)
	}
	return dir
}

// logConfig 记录配置信息
func logConfig(cfg *ModuleConfig) {
	log.Printf("模块启用: %v", cfg.Enabled)
	log.Printf("监听端口: %s", cfg.Port)
	log.Printf("工作区: %s", cfg.Workspace)
	log.Printf("Aurora API: %s", cfg.AuroraAPI)
	log.Printf("模型: %s @ %s", cfg.OpenAIModel, cfg.OpenAIURL)
	if cfg.OpenAIKey != "" {
		keyPreview := cfg.OpenAIKey
		if len(keyPreview) > 8 {
			keyPreview = keyPreview[:8]
		}
		log.Printf("API Key: 已配置 (%s...)", keyPreview)
	} else {
		log.Printf("API Key: 未配置（AI 生成功能不可用）")
	}
}
