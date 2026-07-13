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

type OpenAIAdapter struct {
	httpClient *http.Client
}

func NewOpenAIAdapter(httpClient *http.Client) *OpenAIAdapter {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &OpenAIAdapter{httpClient: httpClient}
}

type openaiTTSRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format,omitempty"`
	Speed          float64 `json:"speed,omitempty"`
}

func (a *OpenAIAdapter) Synthesize(ctx context.Context, profile config.ResolvedTTSAPIProfile, request SynthesizeRequest) (Result, error) {
	speed := 1.0
	if request.Speed != "" {
		_, err := fmt.Sscanf(request.Speed, "%f", &speed)
		if err != nil {
			speed = 1.0
		}
		if speed < 0.25 {
			speed = 0.25
		}
		if speed > 4.0 {
			speed = 4.0
		}
	}

	ttsReq := openaiTTSRequest{
		Model:          profile.OpenAIModel,
		Input:          request.Text,
		Voice:          request.Voice,
		ResponseFormat: request.Format,
		Speed:          speed,
	}
	body, err := json.Marshal(ttsReq)
	if err != nil {
		return Result{}, fmt.Errorf("序列化 TTS 请求失败: %w", err)
	}

	baseURL := profile.OpenAIBaseURL
	if baseURL == "" {
		baseURL = config.DefaultTTSAPIBaseURL
	}
	url := baseURL + "/audio/speech"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return Result{}, fmt.Errorf("创建 TTS 请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+profile.OpenAIAPIKey)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return Result{}, fmt.Errorf("TTS 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return Result{}, fmt.Errorf("TTS 请求失败: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{}, fmt.Errorf("读取 TTS 音频数据失败: %w", err)
	}
	if len(audioData) == 0 {
		return Result{}, fmt.Errorf("TTS 模型未返回音频数据")
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

func mimeTypeForTTSFormat(format string) string {
	switch format {
	case "mp3":
		return "audio/mpeg"
	case "opus":
		return "audio/ogg"
	case "aac":
		return "audio/aac"
	case "flac":
		return "audio/flac"
	case "wav":
		return "audio/wav"
	case "pcm":
		return "audio/pcm"
	default:
		return "audio/mpeg"
	}
}
