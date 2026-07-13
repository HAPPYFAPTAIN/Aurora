package handlers

import (
	"context"
	"errors"
	"log"

	"denova/internal/diagramgen"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// diagramGenerateRequest 是 POST /api/diagrams/generate 的请求体。
type diagramGenerateRequest struct {
	// Type 是预设的图表类型：character/timeline/worldmap/structure/faction。
	Type string `json:"type" vd:"required,len($)>0"`
}

// HandleDiagramGenerate 处理图表生成请求。
// 根据预设类型自动从工作区收集上下文，调用 LLM 生成 Mermaid 代码。
func (h *Handlers) HandleDiagramGenerate(ctx context.Context, c *app.RequestContext) {
	var req diagramGenerateRequest
	if err := c.BindAndValidate(&req); err != nil {
		log.Printf("[handler-diagrams] bind error: %v", err)
		c.JSON(consts.StatusBadRequest, map[string]any{
			"error": "invalid request: type is required (character/timeline/worldmap/structure/faction)",
		})
		return
	}

	result, err := h.app.GenerateDiagram(ctx, diagramgen.GenerateRequest{
		Type: diagramgen.DiagramType(req.Type),
	})
	if err != nil {
		if errors.Is(err, diagramgen.ErrModelNotConfig) {
			c.JSON(consts.StatusUnauthorized, map[string]any{
				"error": "模型未配置，请先在设置中配置 API Key 和模型",
			})
			return
		}
		log.Printf("[handler-diagrams] generate error type=%q: %v", req.Type, err)
		c.JSON(consts.StatusInternalServerError, map[string]any{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, map[string]any{
		"xml": result.XML,
	})
}
