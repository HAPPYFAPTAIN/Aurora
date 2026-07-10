package config

import (
	"errors"
	"fmt"
	"strings"
)

const (
	DefaultTTSAPIProfileID = "default"
	DefaultTTSAPIProvider  = "openai"
	DefaultTTSAPIBaseURL   = "https://api.openai.com/v1"
	DefaultTTSAPIModel     = "gpt-4o-mini-tts"
	DefaultTTSVoice        = "alloy"
	DefaultTTSFormat       = "mp3"
	DefaultTTSSpeed        = 1.0
)

var (
	ErrTTSAPIProfileNotFound = errors.New("TTS 模型配置不存在")
	ErrTTSAPIKeyMissing      = errors.New("TTS 模型 API Key 未配置")
	ErrTTSModelMissing       = errors.New("TTS 模型未配置")
)

type TTSAPIProfileSettings struct {
	ID            string `toml:"id,omitempty" json:"id,omitempty"`
	Name          string `toml:"name,omitempty" json:"name,omitempty"`
	Provider      string `toml:"provider,omitempty" json:"provider,omitempty"`
	OpenAIAPIKey  string `toml:"openai_api_key,omitempty" json:"openai_api_key,omitempty"`
	OpenAIBaseURL string `toml:"openai_base_url,omitempty" json:"openai_base_url,omitempty"`
	OpenAIModel   string `toml:"openai_model,omitempty" json:"openai_model,omitempty"`
	DefaultVoice  string `toml:"default_voice,omitempty" json:"default_voice,omitempty"`
	DefaultFormat string `toml:"default_format,omitempty" json:"default_format,omitempty"`
	DefaultSpeed  string `toml:"default_speed,omitempty" json:"default_speed,omitempty"`
}

type ResolvedTTSAPIProfile struct {
	ProfileID     string
	Name          string
	Provider      string
	OpenAIAPIKey  string
	OpenAIBaseURL string
	OpenAIModel   string
	Voice         string
	Format        string
	Speed         string
}

func ResolveTTSAPIProfile(cfg *Config, requestedID string) (ResolvedTTSAPIProfile, error) {
	if cfg == nil {
		return ResolvedTTSAPIProfile{}, ErrTTSAPIProfileNotFound
	}
	profiles := map[string]TTSAPIProfileSettings{
		DefaultTTSAPIProfileID: legacyTTSAPIProfile(cfg),
	}
	for _, profile := range cfg.TTSAPIProfiles {
		id := ttsAPIProfileID(profile)
		if id == "" {
			continue
		}
		base := profiles[id]
		profile.ID = id
		profiles[id] = mergeTTSAPIProfile(base, profile)
	}

	profileID := normalizeTTSAPIProfileID(requestedID)
	if profileID == "" {
		profileID = normalizeTTSAPIProfileID(cfg.DefaultTTSAPIProfileID)
	}
	if profileID == "" {
		profileID = DefaultTTSAPIProfileID
	}
	profile, ok := profiles[profileID]
	if !ok {
		return ResolvedTTSAPIProfile{}, fmt.Errorf("%w: %s", ErrTTSAPIProfileNotFound, profileID)
	}
	if profile.Provider == "" {
		profile.Provider = DefaultTTSAPIProvider
	}
	if profile.OpenAIAPIKey == "" {
		profile.OpenAIAPIKey = cfg.TTSAPIKey
	}
	if profile.OpenAIBaseURL == "" {
		profile.OpenAIBaseURL = cfg.TTSAPIBaseURL
	}
	if profile.OpenAIBaseURL == "" {
		profile.OpenAIBaseURL = DefaultTTSAPIBaseURL
	}
	if profile.OpenAIModel == "" {
		profile.OpenAIModel = cfg.TTSAPIModel
	}
	if profile.OpenAIModel == "" {
		profile.OpenAIModel = DefaultTTSAPIModel
	}
	if profile.DefaultVoice == "" {
		profile.DefaultVoice = DefaultTTSVoice
	}
	if profile.DefaultFormat == "" {
		profile.DefaultFormat = DefaultTTSFormat
	}
	if profile.DefaultSpeed == "" {
		profile.DefaultSpeed = ""
	}
	if strings.EqualFold(profile.Provider, DefaultTTSAPIProvider) && strings.TrimSpace(profile.OpenAIAPIKey) == "" {
		return ResolvedTTSAPIProfile{}, ErrTTSAPIKeyMissing
	}
	if strings.EqualFold(profile.Provider, DefaultTTSAPIProvider) && strings.TrimSpace(profile.OpenAIModel) == "" {
		return ResolvedTTSAPIProfile{}, ErrTTSModelMissing
	}
	return ResolvedTTSAPIProfile{
		ProfileID:     profileID,
		Name:          strings.TrimSpace(profile.Name),
		Provider:      normalizeTTSAPIProvider(profile.Provider),
		OpenAIAPIKey:  strings.TrimSpace(profile.OpenAIAPIKey),
		OpenAIBaseURL: strings.TrimSpace(profile.OpenAIBaseURL),
		OpenAIModel:   strings.TrimSpace(profile.OpenAIModel),
		Voice:         normalizeTTSVoice(profile.DefaultVoice),
		Format:        normalizeTTSFormat(profile.DefaultFormat),
		Speed:         normalizeTTSSpeed(profile.DefaultSpeed),
	}, nil
}

func mergeTTSAPIProfiles(parent, child []TTSAPIProfileSettings) []TTSAPIProfileSettings {
	if len(child) == 0 {
		return parent
	}
	out := make([]TTSAPIProfileSettings, 0, len(parent)+len(child))
	index := make(map[string]int, len(parent)+len(child))
	for _, profile := range parent {
		id := ttsAPIProfileID(profile)
		if id == "" {
			continue
		}
		profile.ID = id
		index[id] = len(out)
		out = append(out, profile)
	}
	for _, profile := range child {
		id := ttsAPIProfileID(profile)
		if id == "" {
			continue
		}
		profile.ID = id
		if i, ok := index[id]; ok {
			out[i] = mergeTTSAPIProfile(out[i], profile)
		} else {
			index[id] = len(out)
			out = append(out, profile)
		}
	}
	return out
}

func sanitizeTTSAPIProfiles(profiles []TTSAPIProfileSettings) []TTSAPIProfileSettings {
	if len(profiles) == 0 {
		return profiles
	}
	out := make([]TTSAPIProfileSettings, 0, len(profiles))
	for _, profile := range profiles {
		profile.OpenAIModel = strings.TrimSpace(profile.OpenAIModel)
		profile.ID = ttsAPIProfileID(profile)
		if profile.ID == "" {
			continue
		}
		if profile.OpenAIModel == "" && profile.ID != DefaultTTSAPIProfileID {
			profile.OpenAIModel = profile.ID
		}
		profile.Name = strings.TrimSpace(profile.Name)
		profile.Provider = normalizeTTSAPIProvider(profile.Provider)
		profile.OpenAIBaseURL = strings.TrimSpace(profile.OpenAIBaseURL)
		profile.DefaultVoice = normalizeTTSVoice(profile.DefaultVoice)
		profile.DefaultFormat = normalizeTTSFormat(profile.DefaultFormat)
		profile.DefaultSpeed = normalizeTTSSpeed(profile.DefaultSpeed)
		out = append(out, profile)
	}
	return out
}

func mergeTTSAPIProfile(parent, child TTSAPIProfileSettings) TTSAPIProfileSettings {
	out := parent
	if id := ttsAPIProfileID(child); id != "" {
		out.ID = id
	}
	if child.Name != "" {
		out.Name = strings.TrimSpace(child.Name)
	}
	if child.Provider != "" {
		out.Provider = normalizeTTSAPIProvider(child.Provider)
	}
	if child.OpenAIAPIKey != "" {
		out.OpenAIAPIKey = child.OpenAIAPIKey
	}
	if child.OpenAIBaseURL != "" {
		out.OpenAIBaseURL = strings.TrimSpace(child.OpenAIBaseURL)
	}
	if child.OpenAIModel != "" {
		out.OpenAIModel = strings.TrimSpace(child.OpenAIModel)
	}
	if child.DefaultVoice != "" {
		out.DefaultVoice = normalizeTTSVoice(child.DefaultVoice)
	}
	if child.DefaultFormat != "" {
		out.DefaultFormat = normalizeTTSFormat(child.DefaultFormat)
	}
	if child.DefaultSpeed != "" {
		out.DefaultSpeed = normalizeTTSSpeed(child.DefaultSpeed)
	}
	return out
}

func legacyTTSAPIProfile(cfg *Config) TTSAPIProfileSettings {
	return TTSAPIProfileSettings{
		ID:            DefaultTTSAPIProfileID,
		Name:          "默认 TTS 模型",
		Provider:      DefaultTTSAPIProvider,
		OpenAIAPIKey:  cfg.TTSAPIKey,
		OpenAIBaseURL: firstNonEmpty(cfg.TTSAPIBaseURL, DefaultTTSAPIBaseURL),
		OpenAIModel:   firstNonEmpty(cfg.TTSAPIModel, DefaultTTSAPIModel),
		DefaultVoice:  DefaultTTSVoice,
		DefaultFormat: DefaultTTSFormat,
	}
}

func normalizeTTSAPIProfileID(id string) string {
	return strings.TrimSpace(id)
}

func ttsAPIProfileID(profile TTSAPIProfileSettings) string {
	if id := normalizeTTSAPIProfileID(profile.ID); id != "" {
		return id
	}
	return strings.TrimSpace(profile.OpenAIModel)
}

func normalizeTTSAPIProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", DefaultTTSAPIProvider:
		return DefaultTTSAPIProvider
	default:
		return ""
	}
}

func normalizeTTSVoice(voice string) string {
	trimmed := strings.TrimSpace(voice)
	if trimmed == "" {
		return DefaultTTSVoice
	}
	return trimmed
}

func normalizeTTSFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "mp3":
		return "mp3"
	case "opus":
		return "opus"
	case "aac":
		return "aac"
	case "flac":
		return "flac"
	case "wav":
		return "wav"
	case "pcm":
		return "pcm"
	default:
		return "mp3"
	}
}

func normalizeTTSSpeed(speed string) string {
	trimmed := strings.TrimSpace(speed)
	return trimmed
}
