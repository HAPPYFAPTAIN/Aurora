import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import mermaid from 'mermaid'
import { LoaderCircle, Sparkles, XIcon } from 'lucide-react'
import { generateDiagram } from '@/lib/api-client/diagrams'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { InlineErrorNotice } from '@/components/common/inline-error-notice'
import { TooltipIconButton } from '@/components/common/tooltip-icon-button'

interface DiagramPanelProps {
  /** 关闭面板回调。 */
  onClose?: () => void
}

// 初始化 Mermaid（仅一次）
let mermaidInitialized = false
function ensureMermaidInit(theme: 'dark' | 'light') {
  if (mermaidInitialized) return
  mermaid.initialize({
    startOnLoad: false,
    theme,
    securityLevel: 'loose',
    fontFamily: 'inherit',
  })
  mermaidInitialized = true
}

/**
 * DiagramPanel 是一个简洁的 AI 图表自动生成页面：
 * 顶部输入提示词 → 点击生成 → 下方 Mermaid 渲染结果。
 */
export function DiagramPanel({ onClose }: DiagramPanelProps) {
  const { t } = useTranslation()
  const [prompt, setPrompt] = useState('')
  const [generating, setGenerating] = useState(false)
  const [error, setError] = useState('')
  const [mermaidCode, setMermaidCode] = useState('')
  const [svgHtml, setSvgHtml] = useState('')
  const [rendering, setRendering] = useState(false)
  const renderCounter = useRef(0)

  // 检测当前主题
  const isDark = typeof document !== 'undefined' && document.documentElement.classList.contains('dark')
  ensureMermaidInit(isDark ? 'dark' : 'light')

  // 当 mermaidCode 变化时渲染
  useEffect(() => {
    if (!mermaidCode) {
      setSvgHtml('')
      return
    }
    let cancelled = false
    setRendering(true)
    setError('')
    const id = `mermaid-${++renderCounter.current}`
    mermaid.render(id, mermaidCode)
      .then((result) => {
        if (!cancelled) {
          setSvgHtml(result.svg)
          setRendering(false)
        }
      })
      .catch((err) => {
        if (!cancelled) {
          console.error('[DiagramPanel] mermaid render failed', err)
          setError(err instanceof Error ? err.message : t('diagram.renderFailed'))
          setRendering(false)
        }
      })
    return () => { cancelled = true }
  }, [mermaidCode, t])

  const handleGenerate = useCallback(async () => {
    const trimmed = prompt.trim()
    if (!trimmed || generating) return
    setGenerating(true)
    setError('')
    setSvgHtml('')
    try {
      const res = await generateDiagram({ prompt: trimmed })
      if (res.xml) {
        setMermaidCode(res.xml)
      } else {
        setError(t('diagram.generateFailed'))
      }
    } catch (e) {
      console.error('[DiagramPanel] generateDiagram failed', e)
      setError(e instanceof Error ? e.message : t('diagram.generateFailed'))
    } finally {
      setGenerating(false)
    }
  }, [generating, prompt, t])

  return (
    <div className="flex h-full min-h-0 flex-col text-xs text-[var(--nova-text-muted)]">
      {/* 标题栏 */}
      <div className="flex h-9 shrink-0 items-center gap-2 border-b px-3">
        <Sparkles className="h-3.5 w-3.5 text-[var(--nova-text-muted)]" />
        <span className="font-semibold text-[var(--nova-text)]">{t('diagram.title')}</span>
        {onClose && (
          <TooltipIconButton label={t('diagram.close')} onClick={onClose} className="ml-auto">
            <XIcon className="h-3.5 w-3.5" />
          </TooltipIconButton>
        )}
      </div>

      {/* 提示词输入区 */}
      <div className="shrink-0 space-y-2 border-b p-3">
        <Textarea
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          placeholder={t('diagram.promptPlaceholder')}
          minRows={2}
          maxRows={4}
          className="resize-none"
          disabled={generating}
          onKeyDown={(e) => { if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) handleGenerate() }}
        />
        <div className="flex items-center gap-2">
          <Button
            type="button"
            size="sm"
            onClick={handleGenerate}
            disabled={!prompt.trim() || generating}
            className="gap-1.5"
          >
            {generating ? <LoaderCircle className="h-3.5 w-3.5 animate-spin" /> : <Sparkles className="h-3.5 w-3.5" />}
            <span>{generating ? t('diagram.generating') : t('diagram.generate')}</span>
          </Button>
          {error && <InlineErrorNotice message={error} />}
        </div>
      </div>

      {/* 图表渲染区 */}
      <div className="relative min-h-0 flex-1 overflow-auto p-4">
        {rendering && (
          <div className="absolute inset-0 z-[5] flex items-center justify-center bg-[var(--nova-surface)] text-xs text-[var(--nova-text-muted)]">
            {t('diagram.rendering')}
          </div>
        )}
        {svgHtml ? (
          <div
            className="flex h-full w-full items-center justify-center"
            dangerouslySetInnerHTML={{ __html: svgHtml }}
          />
        ) : !rendering && !generating && (
          <div className="flex h-full items-center justify-center text-[var(--nova-text-faint)]">
            {t('diagram.empty')}
          </div>
        )}
      </div>
    </div>
  )
}
