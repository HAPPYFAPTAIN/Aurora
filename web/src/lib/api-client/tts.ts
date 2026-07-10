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
