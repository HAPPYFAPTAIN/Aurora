package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"

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

// HandleTTSVoices GET /api/tts/voices — 返回当前 provider 的可选音色列表。
func (h *Handlers) HandleTTSVoices(ctx context.Context, c *app.RequestContext) {
	profile, err := h.app.ResolveTTSProfile("")
	if err != nil {
		if err == novaApp.ErrNoWorkspace {
			writeErrorKey(c, consts.StatusBadRequest, "api.settings.workspaceMissing")
			return
		}
		writeError(c, consts.StatusBadRequest, err.Error())
		return
	}
	voices := ttsgen.VoicesForProvider(profile.Provider)
	writeJSON(c, consts.StatusOK, map[string]any{
		"provider": profile.Provider,
		"voices":   voices,
	})
}

// HandleTTSStreamSynthesize POST /api/tts/stream — SSE 流式合成，每合成完一段就推送一个事件。
//
// SSE 事件格式：
//
//	data: {"audio_base64":"...","mime_type":"...","chunk_index":0,"total_chunks":5}
//
//	（重复，每个 chunk 一条）
//
//	data: {"done":true}
//
// 若中途出错则推送 data: {"error":"..."} 并结束。
func (h *Handlers) HandleTTSStreamSynthesize(ctx context.Context, c *app.RequestContext) {
	var req ttsgen.SynthesizeRequest
	if err := c.BindJSON(&req); err != nil {
		writeErrorKey(c, consts.StatusBadRequest, "api.common.invalidRequestWithDetail", "detail", err.Error())
		return
	}

	c.Response.Header.Set("Content-Type", "text/event-stream")
	c.Response.Header.Set("Cache-Control", "no-cache")
	c.Response.Header.Set("Connection", "keep-alive")
	c.Response.ImmediateHeaderFlush = true

	pr, pw := io.Pipe()
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("[api] TTS 流式合成 panic recovered err=%v", recovered)
				_ = writeSSEData(pw, map[string]any{"error": fmt.Sprint(recovered)})
			}
			_ = pw.Close()
		}()

		err := h.app.SynthesizeTTSStream(ctx, req, func(chunk ttsgen.ChunkResult) error {
			audioBase64 := base64.StdEncoding.EncodeToString(chunk.Audio)
			return writeSSEData(pw, map[string]any{
				"audio_base64": audioBase64,
				"mime_type":    chunk.MIMEType,
				"chunk_index":  chunk.ChunkIndex,
				"total_chunks": chunk.TotalChunks,
			})
		})
		if err != nil {
			log.Printf("[api] TTS 流式合成 failed err=%v", err)
			_ = writeSSEData(pw, map[string]any{"error": err.Error()})
			return
		}
		_ = writeSSEData(pw, map[string]any{"done": true})
	}()
	c.Response.SetBodyStream(pr, -1)
}

// writeSSEData 向 writer 写入一条 SSE data 事件：data: <json>\n\n
func writeSSEData(w io.Writer, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", payload)
	return err
}
