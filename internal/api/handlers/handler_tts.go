package handlers

import (
	"context"
	"encoding/base64"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	novaApp "denova/internal/app"
	"denova/internal/ttsgen"
)

func (h *Handlers) HandleTTSSynthesize(ctx context.Context, c *app.RequestContext) {
	var req ttsgen.SynthesizeRequest
	if err := c.BindJSON(&req); err != nil {
		writeErrorKey(c, consts.StatusBadRequest, "api.common.invalidRequestWithDetail", "detail", err.Error())
		return
	}
	result, err := h.app.SynthesizeRawTTS(ctx, req)
	if err != nil {
		if err == novaApp.ErrNoWorkspace {
			writeErrorKey(c, consts.StatusBadRequest, "api.settings.workspaceMissing")
			return
		}
		writeError(c, consts.StatusBadRequest, err.Error())
		return
	}
	audioBase64 := base64.StdEncoding.EncodeToString(result.Audio)
	writeJSON(c, consts.StatusOK, map[string]any{
		"profile_id":   result.ProfileID,
		"provider":     result.Provider,
		"model":        result.Model,
		"voice":        result.Voice,
		"format":       result.Format,
		"mime_type":    result.MIMEType,
		"audio_base64": audioBase64,
	})
}
