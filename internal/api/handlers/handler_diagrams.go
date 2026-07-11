package handlers

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"denova/internal/diagramgen"
)

// diagramGenerateRequest 是 POST /api/diagrams/generate 的请求体。
type diagramGenerateRequest struct {
	// Prompt 是用户对图表的自然语言描述。
	Prompt string `json:"prompt"`
	// Context 是可选的附加上下文（如故事摘要、角色列表等）。
	Context string `json:"context,omitempty"`
}

// HandleDiagramGenerate 处理图表生成请求，调用 LLM 生成 draw.io XML。
func (h *Handlers) HandleDiagramGenerate(ctx context.Context, c *app.RequestContext) {
	var req diagramGenerateRequest
	if err := c.BindJSON(&req); err != nil {
		writeErrorKey(c, consts.StatusBadRequest, "api.common.invalidRequestWithDetail", "detail", err.Error())
		return
	}

	result, err := h.app.GenerateDiagram(ctx, diagramgen.GenerateRequest{
		Prompt:  req.Prompt,
		Context: req.Context,
	})
	if err != nil {
		if err == diagramgen.ErrPromptRequired {
			writeErrorKey(c, consts.StatusBadRequest, "api.common.invalidRequestWithDetail", "detail", err.Error())
			return
		}
		if err == diagramgen.ErrModelNotConfig {
			writeErrorKey(c, consts.StatusBadRequest, "api.common.invalidRequestWithDetail", "detail", err.Error())
			return
		}
		writeError(c, consts.StatusBadRequest, err.Error())
		return
	}

	writeJSON(c, consts.StatusOK, result)
}
