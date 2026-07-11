package ttsgen

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"denova/config"
)

var (
	ErrTextRequired        = errors.New("朗读文本不能为空")
	ErrUnsupportedProvider = errors.New("不支持的 TTS 模型 provider")
)

type Service struct {
	adapters map[string]Adapter
}

func NewService() *Service {
	return &Service{adapters: map[string]Adapter{
		config.DefaultTTSAPIProvider: NewOpenAIAdapter(nil),
		config.TTSProviderStepFun:    NewStepFunAdapter(nil),
	}}
}

func (s *Service) Synthesize(ctx context.Context, cfg *config.Config, request SynthesizeRequest) (Result, error) {
	if strings.TrimSpace(request.Text) == "" {
		return Result{}, ErrTextRequired
	}
	profile, err := config.ResolveTTSAPIProfile(cfg, request.ProfileID)
	if err != nil {
		return Result{}, err
	}
	if request.Voice == "" {
		request.Voice = profile.Voice
	}
	if request.Voice == "" {
		return Result{}, errors.New("语音未配置：请在设置中填写 TTS Profile 的语音名，或调用时指定 voice 参数")
	}
	if request.Format == "" {
		request.Format = profile.Format
	}
	if request.Speed == "" {
		request.Speed = profile.Speed
	}

	adapter := s.adapters[profile.Provider]
	if adapter == nil {
		return Result{}, fmt.Errorf("%w: %s", ErrUnsupportedProvider, profile.Provider)
	}
	log.Printf("[ttsgen] synthesize begin provider=%s profile_id=%s model=%q voice=%q format=%q speed=%q text_len=%d", profile.Provider, profile.ProfileID, profile.OpenAIModel, request.Voice, request.Format, request.Speed, len(request.Text))
	result, err := adapter.Synthesize(ctx, profile, request)
	if err != nil {
		log.Printf("[ttsgen] synthesize failed provider=%s profile_id=%s model=%q err=%v", profile.Provider, profile.ProfileID, profile.OpenAIModel, err)
		return Result{}, err
	}
	log.Printf("[ttsgen] synthesize done provider=%s profile_id=%s model=%q audio_bytes=%d", profile.Provider, profile.ProfileID, profile.OpenAIModel, len(result.Audio))
	return result, nil
}
