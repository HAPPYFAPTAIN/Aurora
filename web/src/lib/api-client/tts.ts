import { fetchAPI, readErrorMessage } from './client'

export interface TTSSynthesizeResult {
  profile_id: string
  provider: string
  model: string
  voice: string
  format: string
  mime_type: string
  audio_base64: string
}

export interface TTSVoice {
  id: string
  name: string
}

export interface TTSVoicesResponse {
  provider: string
  voices: TTSVoice[]
}

export async function synthesizeTTS(
  text: string,
  options?: { profile_id?: string; voice?: string; format?: string; speed?: string }
): Promise<TTSSynthesizeResult> {
  const res = await fetchAPI('/api/tts/synthesize', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text, ...options }),
  })
  if (!res.ok) {
    const msg = await readErrorMessage(res)
    throw new Error(msg || `TTS 请求失败: HTTP ${res.status}`)
  }
  return res.json()
}

/**
 * fetchTTSVoices 拉取指定 profile（或默认 profile）可用的音色列表。
 * profile_id 为空时后端使用默认 TTS profile。
 */
export async function fetchTTSVoices(profileID?: string): Promise<TTSVoicesResponse> {
  const query = profileID ? `?profile_id=${encodeURIComponent(profileID)}` : ''
  const res = await fetchAPI(`/api/tts/voices${query}`, {
    method: 'GET',
  })
  if (!res.ok) {
    const msg = await readErrorMessage(res)
    throw new Error(msg || `TTS 音色列表请求失败: HTTP ${res.status}`)
  }
  return res.json()
}

/**
 * streamTTSSynthesize 以 SSE 流式方式合成语音。
 * 后端通过 `data: {...}` 事件逐块返回 base64 音频，每收到一块调用 onAudioChunk 回调。
 * 当后端发送 `{ done: true }` 时结束。
 */
export async function streamTTSSynthesize(
  text: string,
  onAudioChunk: (audioBase64: string, mimeType: string) => void,
  options?: { profile_id?: string; voice?: string; format?: string; speed?: string },
): Promise<void> {
  const res = await fetchAPI('/api/tts/stream', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text, ...options }),
  })
  if (!res.ok) {
    const msg = await readErrorMessage(res)
    throw new Error(msg || `TTS 流式请求失败: HTTP ${res.status}`)
  }

  const reader = res.body!.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    const lines = buffer.split('\n\n')
    buffer = lines.pop() || ''
    for (const line of lines) {
      if (line.startsWith('data: ')) {
        const data = JSON.parse(line.slice(6))
        if (data.done) return
        if (data.audio_base64) {
          onAudioChunk(data.audio_base64, data.mime_type || 'audio/mpeg')
        }
      }
    }
  }
}
