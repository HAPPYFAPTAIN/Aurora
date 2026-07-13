import { useCallback, useRef, useState } from 'react'
import { synthesizeTTS, streamTTSSynthesize } from '@/lib/api-client'

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
  // 流式播放相关：音频队列与完成标记，用 ref 避免 onAudioChunk 闭包陈旧
  const streamQueueRef = useRef<string[]>([])
  const streamDoneRef = useRef<boolean>(false)
  const streamPlayingRef = useRef<boolean>(false)

  const stop = useCallback(() => {
    if (audioRef.current) {
      audioRef.current.pause()
      audioRef.current.currentTime = 0
    }
    audioRef.current = null
    currentMessageIdRef.current = null
    streamQueueRef.current = []
    streamDoneRef.current = false
    streamPlayingRef.current = false
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

  /**
   * speakStream 使用 SSE 流式合成并播放语音。
   * - 收到第一个音频块时立即开始播放
   * - 后续音频块排队，前一个播放完后自动播放下一个
   * - loading 状态持续到所有块接收完；speaking 状态持续到所有块播放完
   */
  const speakStream = useCallback(async (messageId: string, text: string) => {
    // 停止之前的播放（包括流式队列）
    if (audioRef.current) {
      audioRef.current.pause()
      audioRef.current = null
    }
    streamQueueRef.current = []
    streamDoneRef.current = false
    streamPlayingRef.current = false

    currentMessageIdRef.current = messageId
    setState({ speaking: false, loading: true, error: null, speakingMessageId: messageId })

    // 播放队列中下一个音频块；队列空且流式已结束时归位状态
    const playNext = () => {
      const next = streamQueueRef.current.shift()
      if (!next) {
        streamPlayingRef.current = false
        if (streamDoneRef.current) {
          // 所有块已接收且播放完毕
          audioRef.current = null
          currentMessageIdRef.current = null
          setState({ speaking: false, loading: false, error: null, speakingMessageId: null })
        }
        return
      }
      const audio = new Audio(next)
      audioRef.current = audio
      audio.onended = () => {
        audioRef.current = null
        playNext()
      }
      audio.onerror = () => {
        audioRef.current = null
        currentMessageIdRef.current = null
        streamQueueRef.current = []
        streamPlayingRef.current = false
        setState({ speaking: false, loading: false, error: '音频播放失败', speakingMessageId: null })
      }
      audio.play().catch(() => {
        audioRef.current = null
        currentMessageIdRef.current = null
        streamQueueRef.current = []
        streamPlayingRef.current = false
        setState({ speaking: false, loading: false, error: '音频播放失败', speakingMessageId: null })
      })
    }

    try {
      await streamTTSSynthesize(text, (audioBase64, mimeType) => {
        const dataUrl = `data:${mimeType};base64,${audioBase64}`
        streamQueueRef.current.push(dataUrl)
        // 第一个块：立即开始播放，loading -> speaking
        if (!streamPlayingRef.current) {
          streamPlayingRef.current = true
          setState(prev => ({ ...prev, loading: false, speaking: true }))
          playNext()
        }
      })
      // 流式传输结束，标记完成；若队列已空且未在播放则直接归位
      streamDoneRef.current = true
      if (!streamPlayingRef.current && streamQueueRef.current.length === 0) {
        audioRef.current = null
        currentMessageIdRef.current = null
        setState({ speaking: false, loading: false, error: null, speakingMessageId: null })
      }
    } catch (err) {
      let msg = err instanceof Error ? err.message : String(err)
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
      streamQueueRef.current = []
      streamDoneRef.current = false
      streamPlayingRef.current = false
      setState({ speaking: false, loading: false, error: msg, speakingMessageId: null })
      alert(`朗读失败：${msg}`)
    }
  }, [stop])

  return { ...state, speak, speakStream, stop }
}
