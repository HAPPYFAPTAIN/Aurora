package ttsgen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"denova/config"
)

type StepFunAdapter struct {
	httpClient *http.Client
}

func NewStepFunAdapter(httpClient *http.Client) *StepFunAdapter {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &StepFunAdapter{httpClient: httpClient}
}

type stepfunTTSRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	VoiceID        string  `json:"voice_id"`
	ResponseFormat string  `json:"response_format,omitempty"`
	SpeedRatio     float64 `json:"speed_ratio,omitempty"`
	VolumeRatio    float64 `json:"volume_ratio,omitempty"`
	SampleRate     int     `json:"sample_rate,omitempty"`
	Instruction    string  `json:"instruction,omitempty"`
}

func (a *StepFunAdapter) Synthesize(ctx context.Context, profile config.ResolvedTTSAPIProfile, request SynthesizeRequest) (Result, error) {
	speedRatio := 1.0
	if request.Speed != "" {
		_, err := fmt.Sscanf(request.Speed, "%f", &speedRatio)
		if err != nil {
			speedRatio = 1.0
		}
		if speedRatio < 0.25 {
			speedRatio = 0.25
		}
		if speedRatio > 4.0 {
			speedRatio = 4.0
		}
	}

	ttsReq := stepfunTTSRequest{
		Model:          profile.OpenAIModel,
		Input:          request.Text,
		VoiceID:        request.Voice,
		ResponseFormat: request.Format,
		SpeedRatio:     speedRatio,
		VolumeRatio:    1.0,
	}
	if profile.Instruction != "" {
		ttsReq.Instruction = profile.Instruction
	}

	body, err := json.Marshal(ttsReq)
	if err != nil {
		return Result{}, fmt.Errorf("序列化 Step Fun TTS 请求失败: %w", err)
	}

	baseURL := profile.OpenAIBaseURL
	if baseURL == "" {
		baseURL = config.StepFunDefaultBaseURL
	}
	url := baseURL + "/audio/speech"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return Result{}, fmt.Errorf("创建 Step Fun TTS 请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+profile.OpenAIAPIKey)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return Result{}, fmt.Errorf("Step Fun TTS 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return Result{}, fmt.Errorf("Step Fun TTS 请求失败: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{}, fmt.Errorf("读取 Step Fun TTS 音频数据失败: %w", err)
	}
	if len(audioData) == 0 {
		return Result{}, fmt.Errorf("Step Fun TTS 模型未返回音频数据")
	}

	mimeType := mimeTypeForTTSFormat(request.Format)
	return Result{
		ProfileID: profile.ProfileID,
		Provider:  profile.Provider,
		Model:     profile.OpenAIModel,
		Voice:     request.Voice,
		Format:    request.Format,
		Audio:     audioData,
		MIMEType:  mimeType,
	}, nil
}
