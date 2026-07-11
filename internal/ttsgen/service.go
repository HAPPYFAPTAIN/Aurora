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

const maxTTSChunkSize = 900

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

	chunks := splitTTSChunks(request.Text, maxTTSChunkSize)
	log.Printf("[ttsgen] synthesize begin provider=%s profile_id=%s model=%q voice=%q format=%q speed=%q text_len=%d chunks=%d", profile.Provider, profile.ProfileID, profile.OpenAIModel, request.Voice, request.Format, request.Speed, len(request.Text), len(chunks))

	if len(chunks) == 1 {
		result, err := adapter.Synthesize(ctx, profile, request)
		if err != nil {
			log.Printf("[ttsgen] synthesize failed provider=%s profile_id=%s model=%q err=%v", profile.Provider, profile.ProfileID, profile.OpenAIModel, err)
			return Result{}, err
		}
		log.Printf("[ttsgen] synthesize done provider=%s profile_id=%s model=%q audio_bytes=%d", profile.Provider, profile.ProfileID, profile.OpenAIModel, len(result.Audio))
		return result, nil
	}

	// 多段合成
	var combinedAudio []byte
	for i, chunk := range chunks {
		chunkReq := request
		chunkReq.Text = chunk
		log.Printf("[ttsgen] synthesize chunk %d/%d provider=%s text_len=%d", i+1, len(chunks), profile.Provider, len(chunk))
		result, err := adapter.Synthesize(ctx, profile, chunkReq)
		if err != nil {
			log.Printf("[ttsgen] synthesize chunk %d failed provider=%s err=%v", i+1, profile.Provider, err)
			return Result{}, err
		}
		combinedAudio = append(combinedAudio, result.Audio...)
		log.Printf("[ttsgen] synthesize chunk %d done audio_bytes=%d total=%d", i+1, len(result.Audio), len(combinedAudio))
	}

	mimeType := mimeTypeForTTSFormat(request.Format)
	log.Printf("[ttsgen] synthesize all done provider=%s model=%q chunks=%d total_audio_bytes=%d", profile.Provider, profile.OpenAIModel, len(chunks), len(combinedAudio))

	return Result{
		ProfileID: profile.ProfileID,
		Provider:  profile.Provider,
		Model:     profile.OpenAIModel,
		Voice:     request.Voice,
		Format:    request.Format,
		Audio:     combinedAudio,
		MIMEType:  mimeType,
	}, nil
}

// splitTTSChunks 将文本按自然段落分割，每段不超过 maxChars 字符
func splitTTSChunks(text string, maxChars int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if len([]rune(text)) <= maxChars {
		return []string{text}
	}

	// 按段落分割
	paragraphs := strings.Split(text, "\n")

	var chunks []string
	var current strings.Builder
	currentLen := 0

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		paraRunes := []rune(para)
		paraLen := len(paraRunes)

		// 如果当前段落本身超过限制，按句子分割
		if paraLen > maxChars {
			// 先把当前缓冲区 flush
			if currentLen > 0 {
				chunks = append(chunks, current.String())
				current.Reset()
				currentLen = 0
			}
			// 按句号/问号/感叹号分割
			sentences := splitBySentence(para)
			for _, sent := range sentences {
				sent = strings.TrimSpace(sent)
				if sent == "" {
					continue
				}
				sentRunes := []rune(sent)
				sentLen := len(sentRunes)
				if sentLen > maxChars {
					// 句子也太长，硬切
					for start := 0; start < sentLen; start += maxChars {
						end := start + maxChars
						if end > sentLen {
							end = sentLen
						}
						chunks = append(chunks, string(sentRunes[start:end]))
					}
				} else if currentLen+sentLen+1 > maxChars {
					if currentLen > 0 {
						chunks = append(chunks, current.String())
						current.Reset()
						currentLen = 0
					}
					current.WriteString(sent)
					currentLen = sentLen
				} else {
					if currentLen > 0 {
						current.WriteString("\n")
						currentLen++
					}
					current.WriteString(sent)
					currentLen += sentLen
				}
			}
			continue
		}

		// 段落不超过限制
		if currentLen+paraLen+1 > maxChars {
			if currentLen > 0 {
				chunks = append(chunks, current.String())
				current.Reset()
				currentLen = 0
			}
			current.WriteString(para)
			currentLen = paraLen
		} else {
			if currentLen > 0 {
				current.WriteString("\n")
				currentLen++
			}
			current.WriteString(para)
			currentLen += paraLen
		}
	}

	if currentLen > 0 {
		chunks = append(chunks, current.String())
	}

	if len(chunks) == 0 {
		return []string{text}
	}
	return chunks
}

// splitBySentence 按句子分割（句号、问号、感叹号、分号）
func splitBySentence(text string) []string {
	var sentences []string
	var current strings.Builder
	for _, r := range text {
		current.WriteRune(r)
		if r == '。' || r == '！' || r == '？' || r == '；' || r == '.' || r == '!' || r == '?' || r == ';' {
			sentences = append(sentences, current.String())
			current.Reset()
		}
	}
	if current.Len() > 0 {
		sentences = append(sentences, current.String())
	}
	return sentences
}
