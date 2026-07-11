package handlers

import (
	"context"
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"denova/internal/materialindex"
)

// HandleMaterialIndexSearch GET /api/material-index/search — 搜索资料卡片。
func (h *Handlers) HandleMaterialIndexSearch(ctx context.Context, c *app.RequestContext) {
	if !h.requireWorkspace(c) {
		return
	}
	h.app.EnsureMaterialIndexBuilt()

	query := strings.TrimSpace(string(c.Query("q")))
	typeFilter := strings.TrimSpace(string(c.Query("type")))
	limit := 50
	if l := string(c.Query("limit")); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}

	resp := h.app.MaterialIndexSearch(query, typeFilter, limit)
	writeJSON(c, consts.StatusOK, resp)
}

// HandleMaterialIndexStats GET /api/material-index/stats — 索引统计信息。
func (h *Handlers) HandleMaterialIndexStats(ctx context.Context, c *app.RequestContext) {
	if !h.requireWorkspace(c) {
		return
	}
	h.app.EnsureMaterialIndexBuilt()
	writeJSON(c, consts.StatusOK, h.app.MaterialIndexStats())
}

// HandleMaterialIndexCard GET /api/material-index/card/:id — 获取单张卡片详情。
func (h *Handlers) HandleMaterialIndexCard(ctx context.Context, c *app.RequestContext) {
	if !h.requireWorkspace(c) {
		return
	}
	h.app.EnsureMaterialIndexBuilt()
	id := c.Param("id")
	if id == "" {
		writeError(c, consts.StatusBadRequest, "缺少卡片ID")
		return
	}
	card, ok := h.app.MaterialIndexGetCard(id)
	if !ok {
		writeError(c, consts.StatusNotFound, "卡片不存在: "+id)
		return
	}
	writeJSON(c, consts.StatusOK, card)
}

// HandleMaterialIndexTemplates GET /api/material-index/templates — 列出卡片模板。
func (h *Handlers) HandleMaterialIndexTemplates(ctx context.Context, c *app.RequestContext) {
	templates := h.app.MaterialIndexListTemplates()
	writeJSON(c, consts.StatusOK, map[string]interface{}{"templates": templates})
}

// HandleMaterialIndexGenerate POST /api/material-index/generate — AI 生成卡片并写入资料库。
func (h *Handlers) HandleMaterialIndexGenerate(ctx context.Context, c *app.RequestContext) {
	if !h.requireWorkspace(c) {
		return
	}
	var req materialindex.GenRequest
	if err := c.BindJSON(&req); err != nil {
		writeError(c, consts.StatusBadRequest, "请求解析失败: "+err.Error())
		return
	}
	resp := h.app.MaterialIndexGenerate(req)
	writeJSON(c, consts.StatusOK, resp)
}

// HandleMaterialIndexRebuild POST /api/material-index/rebuild — 重建索引。
func (h *Handlers) HandleMaterialIndexRebuild(ctx context.Context, c *app.RequestContext) {
	if !h.requireWorkspace(c) {
		return
	}
	count, err := h.app.MaterialIndexRebuild()
	if err != nil {
		writeError(c, consts.StatusInternalServerError, "重建索引失败: "+err.Error())
		return
	}
	writeJSON(c, consts.StatusOK, map[string]interface{}{"success": true, "total": count})
}

// HandleMaterialIndexImport POST /api/material-index/import — 导入文本文件。
func (h *Handlers) HandleMaterialIndexImport(ctx context.Context, c *app.RequestContext) {
	if !h.requireWorkspace(c) {
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		writeError(c, consts.StatusBadRequest, "未找到上传文件")
		return
	}
	data, err := file.Open()
	if err != nil {
		writeError(c, consts.StatusInternalServerError, "读取文件失败: "+err.Error())
		return
	}
	defer data.Close()

	buf := make([]byte, file.Size)
	_, err = data.Read(buf)
	if err != nil {
		writeError(c, consts.StatusInternalServerError, "读取文件内容失败: "+err.Error())
		return
	}

	filename := file.Filename
	savedName, size, err := h.app.MaterialIndexImport(filename, buf)
	if err != nil {
		writeError(c, consts.StatusInternalServerError, "保存文件失败: "+err.Error())
		return
	}
	writeJSON(c, consts.StatusOK, map[string]interface{}{
		"success":  true,
		"filename": savedName,
		"size":     size,
	})
}

// HandleMaterialIndexImports GET /api/material-index/imports — 列出已导入文件。
func (h *Handlers) HandleMaterialIndexImports(ctx context.Context, c *app.RequestContext) {
	if !h.requireWorkspace(c) {
		return
	}
	files := h.app.MaterialIndexListImports()
	writeJSON(c, consts.StatusOK, map[string]interface{}{"files": files})
}

// HandleMaterialIndexImportDelete DELETE /api/material-index/imports — 删除导入文件。
func (h *Handlers) HandleMaterialIndexImportDelete(ctx context.Context, c *app.RequestContext) {
	if !h.requireWorkspace(c) {
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if err := c.BindJSON(&req); err != nil {
		writeError(c, consts.StatusBadRequest, "请求解析失败")
		return
	}
	if err := h.app.MaterialIndexDeleteImport(req.Path); err != nil {
		writeError(c, consts.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}
	writeJSON(c, consts.StatusOK, map[string]interface{}{"success": true})
}

// HandleMaterialIndexWorkspaceSearch GET /api/material-index/workspace-search — 搜索工作区文件。
func (h *Handlers) HandleMaterialIndexWorkspaceSearch(ctx context.Context, c *app.RequestContext) {
	if !h.requireWorkspace(c) {
		return
	}
	query := strings.TrimSpace(string(c.Query("q")))
	if query == "" {
		writeError(c, consts.StatusBadRequest, "缺少查询参数 q")
		return
	}
	results, err := h.app.MaterialIndexWorkspaceSearch(query)
	if err != nil {
		writeError(c, consts.StatusInternalServerError, "搜索失败: "+err.Error())
		return
	}
	writeJSON(c, consts.StatusOK, map[string]interface{}{"results": results})
}
