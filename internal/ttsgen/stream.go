package ttsgen

import (
	"context"
	"fmt"
	"log"
	"strings"

	"denova/config"
)

// ChunkResult 是流式合成中单个文本片段的合成结果。
// 调用方负责将 Audio 进行 base64 编码后通过 SSE 推送给前端。
type ChunkResult struct {
	Audio       []byte
	MIMEType    string
	ChunkIndex  int
	TotalChunks int
}

// SynthesizeStream 逐段合成文本，每完成一段就通过 onChunk 回调推送结果。
// 每段使用 HTTP 非流式合成（adapter.Synthesize）以保证音质，但结果增量返回，
// 调用方可据此实现 SSE 流式推送，让前端边合成边播放。
//
// 与 Synthesize 不同的是：Synthesize 会把所有段拼接成一个完整音频返回，
// 而 SynthesizeStream 逐段回调，不持有完整音频，适合长文本流式场景。
// 配置解析、默认值填充、分段逻辑与 Synthesize 保持一致。
func (s *Service) SynthesizeStream(ctx context.Context, cfg *config.Config, request SynthesizeRequest, onChunk func(ChunkResult) error) error {
	if strings.TrimSpace(request.Text) == "" {
		return ErrTextRequired
	}
	profile, err := config.ResolveTTSAPIProfile(cfg, request.ProfileID)
	if err != nil {
		return err
	}
	if request.Voice == "" {
		request.Voice = profile.Voice
	}
	if request.Voice == "" {
		return fmt.Errorf("语音未配置：请在设置中填写 TTS Profile 的语音名，或调用时指定 voice 参数")
	}
	if request.Format == "" {
		request.Format = profile.Format
	}
	if request.Speed == "" {
		request.Speed = profile.Speed
	}

	adapter := s.adapters[profile.Provider]
	if adapter == nil {
		return fmt.Errorf("%w: %s", ErrUnsupportedProvider, profile.Provider)
	}

	chunks := splitTTSChunks(request.Text, maxTTSChunkSize)
	totalChunks := len(chunks)
	log.Printf("[ttsgen] stream synthesize begin provider=%s profile_id=%s model=%q voice=%q format=%q speed=%q text_len=%d chunks=%d",
		profile.Provider, profile.ProfileID, profile.OpenAIModel, request.Voice, request.Format, request.Speed, len(request.Text), totalChunks)

	for i, chunk := range chunks {
		chunkReq := request
		chunkReq.Text = chunk
		log.Printf("[ttsgen] stream synthesize chunk %d/%d provider=%s text_len=%d", i+1, totalChunks, profile.Provider, len(chunk))
		result, err := adapter.Synthesize(ctx, profile, chunkReq)
		if err != nil {
			log.Printf("[ttsgen] stream synthesize chunk %d failed provider=%s err=%v", i+1, profile.Provider, err)
			return err
		}
		if err := onChunk(ChunkResult{
			Audio:       result.Audio,
			MIMEType:    result.MIMEType,
			ChunkIndex:  i,
			TotalChunks: totalChunks,
		}); err != nil {
			return err
		}
		log.Printf("[ttsgen] stream synthesize chunk %d done audio_bytes=%d", i+1, len(result.Audio))
	}

	log.Printf("[ttsgen] stream synthesize all done provider=%s model=%q chunks=%d", profile.Provider, profile.OpenAIModel, totalChunks)
	return nil
}
