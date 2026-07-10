package app

import (
	"context"
	"fmt"
	"log"

	"denova/config"
	"denova/internal/ttsgen"
)

type TTSAppService struct {
	app *App
}

type TTSSynthesizeResult struct {
	ProfileID   string `json:"profile_id"`
	Provider    string `json:"provider"`
	Model       string `json:"model"`
	Voice       string `json:"voice"`
	Format      string `json:"format"`
	MIMEType    string `json:"mime_type"`
	AudioBase64 string `json:"audio_base64"`
}

func (a *App) SynthesizeTTS(ctx context.Context, request ttsgen.SynthesizeRequest) (TTSSynthesizeResult, error) {
	return a.tts().Synthesize(ctx, request)
}

func (a *App) SynthesizeRawTTS(ctx context.Context, request ttsgen.SynthesizeRequest) (ttsgen.Result, error) {
	return a.tts().SynthesizeRaw(ctx, request)
}

func (s *TTSAppService) Synthesize(ctx context.Context, request ttsgen.SynthesizeRequest) (TTSSynthesizeResult, error) {
	cfg, err := s.runtimeSnapshot()
	if err != nil {
		return TTSSynthesizeResult{}, err
	}
	result, err := ttsgen.NewService().Synthesize(ctx, &cfg, request)
	if err != nil {
		return TTSSynthesizeResult{}, err
	}
	log.Printf("[tts] synthesized profile=%s model=%s voice=%s format=%s bytes=%d", result.ProfileID, result.Model, result.Voice, result.Format, len(result.Audio))
	return TTSSynthesizeResult{
		ProfileID:   result.ProfileID,
		Provider:    result.Provider,
		Model:       result.Model,
		Voice:       result.Voice,
		Format:      result.Format,
		MIMEType:    result.MIMEType,
		AudioBase64: "", // Will be set by handler
	}, nil
}

func (s *TTSAppService) SynthesizeRaw(ctx context.Context, request ttsgen.SynthesizeRequest) (ttsgen.Result, error) {
	cfg, err := s.runtimeSnapshot()
	if err != nil {
		return ttsgen.Result{}, err
	}
	return ttsgen.NewService().Synthesize(ctx, &cfg, request)
}

func (s *TTSAppService) runtimeSnapshot() (config.Config, error) {
	app := s.app
	app.mu.RLock()
	defer app.mu.RUnlock()
	if app.workspace == "" || app.bookService == nil {
		return config.Config{}, ErrNoWorkspace
	}
	if app.cfg == nil {
		return config.Config{}, fmt.Errorf("运行配置未初始化")
	}
	cfg := *app.cfg
	return cfg, nil
}
