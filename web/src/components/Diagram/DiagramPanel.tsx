import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import mermaid from 'mermaid'
import { Users, Clock3, Map, GitBranch, Swords, LoaderCircle, Sparkles, XIcon } from 'lucide-react'
import { generateDiagram, type DiagramType } from '@/lib/api-client/diagrams'
import { InlineErrorNotice } from '@/components/common/inline-error-notice'
import { TooltipIconButton } from '@/components/common/tooltip-icon-button'

interface DiagramPanelProps {
  onClose?: () => void
}

interface PresetItem {
  type: DiagramType
  icon: typeof Users
  labelKey: string
  descKey: string
}

const PRESETS: PresetItem[] = [
  { type: 'character', icon: Users, labelKey: 'diagram.type.character', descKey: 'diagram.type.character.desc' },
  { type: 'timeline', icon: Clock3, labelKey: 'diagram.type.timeline', descKey: 'diagram.type.timeline.desc' },
  { type: 'worldmap', icon: Map, labelKey: 'diagram.type.worldmap', descKey: 'diagram.type.worldmap.desc' },
  { type: 'structure', icon: GitBranch, labelKey: 'diagram.type.structure', descKey: 'diagram.type.structure.desc' },
  { type: 'faction', icon: Swords, labelKey: 'diagram.type.faction', descKey: 'diagram.type.faction.desc' },
]

let mermaidInitialized = false
function ensureMermaidInit(theme: 'dark' | 'light') {
  if (mermaidInitialized) return
  mermaid.initialize({ startOnLoad: false, theme: theme === 'dark' ? 'dark' : 'default', securityLevel: 'loose', fontFamily: 'inherit' })
  mermaidInitialized = true
}

export function DiagramPanel({ onClose }: DiagramPanelProps) {
  const { t } = useTranslation()
  const [generating, setGenerating] = useState(false)
  const [activeType, setActiveType] = useState<DiagramType | null>(null)
  const [error, setError] = useState('')
  const [mermaidCode, setMermaidCode] = useState('')
  const [svgHtml, setSvgHtml] = useState('')
  const [rendering, setRendering] = useState(false)
  const renderCounter = useRef(0)

  const isDark = typeof document !== 'undefined' && document.documentElement.classList.contains('dark')
  ensureMermaidInit(isDark ? 'dark' : 'light')

  useEffect(() => {
    if (!mermaidCode) { setSvgHtml(''); return }
    let cancelled = false
    setRendering(true)
    setError('')
    const id = `mermaid-${++renderCounter.current}`
    mermaid.render(id, mermaidCode)
      .then((result) => { if (!cancelled) { setSvgHtml(result.svg); setRendering(false) } })
      .catch((err) => {
        if (!cancelled) {
          console.error('[DiagramPanel] mermaid render failed', err)
          setError(err instanceof Error ? err.message : t('diagram.renderFailed'))
          setRendering(false)
        }
      })
    return () => { cancelled = true }
  }, [mermaidCode, t])

  const handleGenerate = useCallback(async (type: DiagramType) => {
    if (generating) return
    setActiveType(type)
    setGenerating(true)
    setError('')
    setSvgHtml('')
    setMermaidCode('')
    try {
      const res = await generateDiagram({ type })
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
  }, [generating, t])

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

      {/* 预设按钮区 */}
      <div className="shrink-0 border-b p-3">
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
          {PRESETS.map((preset) => {
            const Icon = preset.icon
            const isActive = activeType === preset.type
            return (
              <button
                key={preset.type}
                type="button"
                onClick={() => handleGenerate(preset.type)}
                disabled={generating}
                className={[
                  'flex items-center gap-2.5 rounded-lg border p-2.5 text-left transition-colors',
                  'hover:bg-[var(--nova-hover)] disabled:opacity-50',
                  isActive && generating
                    ? 'border-[var(--nova-accent)] bg-[var(--nova-hover)]'
                    : 'border-[var(--nova-border)]',
                ].filter(Boolean).join(' ')}
              >
                <Icon className="h-4 w-4 shrink-0 text-[var(--nova-text-muted)]" />
                <div className="min-w-0 flex-1">
                  <div className="truncate font-medium text-[var(--nova-text)]">{t(preset.labelKey)}</div>
                  <div className="truncate text-[var(--nova-text-faint)]">{t(preset.descKey)}</div>
                </div>
                {isActive && generating && <LoaderCircle className="h-3.5 w-3.5 shrink-0 animate-spin" />}
              </button>
            )
          })}
        </div>
        {error && <div className="mt-2"><InlineErrorNotice message={error} /></div>}
      </div>

      {/* 图表渲染区 */}
      <div className="relative min-h-0 flex-1 overflow-auto p-4">
        {rendering && (
          <div className="absolute inset-0 z-[5] flex items-center justify-center bg-[var(--nova-surface)] text-xs text-[var(--nova-text-muted)]">
            {t('diagram.rendering')}
          </div>
        )}
        {svgHtml ? (
          <div className="flex h-full w-full items-center justify-center" dangerouslySetInnerHTML={{ __html: svgHtml }} />
        ) : !rendering && !generating && (
          <div className="flex h-full items-center justify-center text-[var(--nova-text-faint)]">
            {t('diagram.empty')}
          </div>
        )}
      </div>
    </div>
  )
}
