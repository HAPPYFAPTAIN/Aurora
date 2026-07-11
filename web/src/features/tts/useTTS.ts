import { useCallback, useRef, useState } from 'react'
import { synthesizeTTS } from '@/lib/api-client'

export interface TTSState {
  speaking: boolean
  loading: boolean
  error: string | null
  speakingMessageId: string | null
}

export function useTTS() {
  const [state, setState] = useState<TTSState>({
    speaking: false,
    loading: false,
    error: null,
    speakingMessageId: null,
  })
  const audioRef = useRef<HTMLAudioElement | null>(null)
  const currentMessageIdRef = useRef<string | null>(null)

  const stop = useCallback(() => {
    if (audioRef.current) {
      audioRef.current.pause()
      audioRef.current.currentTime = 0
    }
    audioRef.current = null
    currentMessageIdRef.current = null
    setState({ speaking: false, loading: false, error: null, speakingMessageId: null })
  }, [])

  const speak = useCallback(async (messageId: string, text: string) => {
    // 如果正在播放同一条消息，停止
    if (currentMessageIdRef.current === messageId && audioRef.current) {
      stop()
      return
    }
    // 停止之前的播放
    if (audioRef.current) {
      audioRef.current.pause()
      audioRef.current = null
    }
    currentMessageIdRef.current = messageId
    setState({ speaking: false, loading: true, error: null, speakingMessageId: messageId })
    try {
      const result = await synthesizeTTS(text)
      const audioData = `data:${result.mime_type};base64,${result.audio_base64}`
      const audio = new Audio(audioData)
      audioRef.current = audio
      setState(prev => ({ ...prev, loading: false, speaking: true }))
      audio.onended = () => {
        audioRef.current = null
        currentMessageIdRef.current = null
        setState({ speaking: false, loading: false, error: null, speakingMessageId: null })
      }
      audio.onerror = () => {
        audioRef.current = null
        currentMessageIdRef.current = null
        setState({ speaking: false, loading: false, error: '音频播放失败', speakingMessageId: null })
      }
      await audio.play()
    } catch (err) {
      let msg = err instanceof Error ? err.message : String(err)
      // 提取 API 返回的错误信息
      try {
        const jsonMsg = JSON.parse(msg)
        if (jsonMsg.error?.message) {
          msg = jsonMsg.error.message
        }
      } catch {
        // not JSON, use raw message
      }
      audioRef.current = null
      currentMessageIdRef.current = null
      setState({ speaking: false, loading: false, error: msg, speakingMessageId: null })
      // 用 alert 确保用户看到错误
      alert(`朗读失败：${msg}`)
    }
  }, [stop])

  return { ...state, speak, stop }
}
