package ttsgen

import (
	"context"

	"denova/config"
)

type SynthesizeRequest struct {
	ProfileID string `json:"profile_id,omitempty"`
	Text      string `json:"text"`
	Voice     string `json:"voice,omitempty"`
	Format    string `json:"format,omitempty"`
	Speed     string `json:"speed,omitempty"`
}

type Result struct {
	ProfileID string
	Provider  string
	Model     string
	Voice     string
	Format    string
	Audio     []byte
	MIMEType  string
}

type Adapter interface {
	Synthesize(ctx context.Context, profile config.ResolvedTTSAPIProfile, request SynthesizeRequest) (Result, error)
}
