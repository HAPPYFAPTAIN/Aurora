package app

import (
	"context"
	"fmt"

	"denova/config"
	"denova/internal/diagramgen"
)

// DiagramAppService 负责图表生成任务的 App 层封装，
// 从 App 运行时快照中获取配置后委托给 diagramgen.Service。
type DiagramAppService struct {
	app *App
}

// DiagramGenerateResult 是图表生成的对外结果。
type DiagramGenerateResult struct {
	XML string `json:"xml"`
}

// GenerateDiagram 是 App 门面方法，委托给 DiagramAppService。
func (a *App) GenerateDiagram(ctx context.Context, request diagramgen.GenerateRequest) (DiagramGenerateResult, error) {
	return a.diagrams().Generate(ctx, request)
}

// Generate 解析当前运行时配置后调用 diagramgen.Service 生成图表。
// 使用 IDE Agent 的模型 profile，因为图表生成属于创作辅助场景。
func (s *DiagramAppService) Generate(ctx context.Context, request diagramgen.GenerateRequest) (DiagramGenerateResult, error) {
	cfg, err := s.configSnapshot()
	if err != nil {
		return DiagramGenerateResult{}, err
	}
	result, err := diagramgen.NewService(nil).Generate(ctx, &cfg, config.AgentKindIDE, request)
	if err != nil {
		return DiagramGenerateResult{}, err
	}
	return DiagramGenerateResult{XML: result.XML}, nil
}

// configSnapshot 从 App 中安全地读取当前配置快照。
func (s *DiagramAppService) configSnapshot() (config.Config, error) {
	app := s.app
	app.mu.RLock()
	defer app.mu.RUnlock()
	if app.cfg == nil {
		return config.Config{}, fmt.Errorf("运行配置未初始化")
	}
	return *app.cfg, nil
}
