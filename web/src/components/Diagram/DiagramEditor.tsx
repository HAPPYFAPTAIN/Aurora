import { forwardRef, useCallback, useEffect, useImperativeHandle, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { DrawIoEmbed, type DrawIoEmbedRef, type EventAutoSave, type EventExport, type EventLoad } from 'react-drawio'
import { LoaderCircle, Sparkles } from 'lucide-react'
import { generateDiagram } from '@/lib/api-client/diagrams'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { InlineErrorNotice } from '@/components/common/inline-error-notice'

/** DiagramEditor 暴露给父组件的命令式接口。 */
export interface DiagramEditorHandle {
  /** 获取当前编辑器中的图表 XML。 */
  getXml: () => Promise<string>
  /** 加载新的 XML 到编辑器。 */
  loadXml: (xml: string) => void
  /** 新建空白图表。 */
  newDiagram: () => void
}

interface DiagramEditorProps {
  /** 初始图表 XML，变化时会重新加载到编辑器。 */
  initialXml?: string
  /** 当 draw.io 原生保存触发时回调（需启用保存按钮）。 */
  onSave?: (xml: string) => void
}

/**
 * DiagramEditor 基于 react-drawio 嵌入 draw.io 编辑器，
 * 支持初始 XML 加载、AI 生成图表以及命令式 XML 读取。
 */
export const DiagramEditor = forwardRef<DiagramEditorHandle, DiagramEditorProps>(function DiagramEditor(
  { initialXml, onSave },
  ref,
) {
  const { t } = useTranslation()
  const editorRef = useRef<DrawIoEmbedRef>(null)
  const [editorReady, setEditorReady] = useState(false)
  const [diagramXml, setDiagramXml] = useState<string>(initialXml ?? '')
  const [aiDialogOpen, setAiDialogOpen] = useState(false)
  const [prompt, setPrompt] = useState('')
  const [context, setContext] = useState('')
  const [generating, setGenerating] = useState(false)
  const [error, setError] = useState('')

  // 通过 autosave 事件持续追踪编辑器中的最新 XML，供 getXml 同步返回。
  const latestXmlRef = useRef<string>('')
  // 当 autosave 尚未捕获到 XML 时的导出回退解析器。
  const exportResolverRef = useRef<((xml: string) => void) | null>(null)

  // initialXml 变化时同步到本地状态，触发 DrawIoEmbed 重新加载。
  useEffect(() => {
    setDiagramXml(initialXml ?? '')
  }, [initialXml])

  // 组件卸载时清理未完成的导出回退，避免 Promise 悬挂。
  useEffect(() => {
    return () => {
      if (exportResolverRef.current) {
        exportResolverRef.current('')
        exportResolverRef.current = null
      }
    }
  }, [])

  const handleLoad = useCallback((data: EventLoad) => {
    setEditorReady(true)
    latestXmlRef.current = data.xml
  }, [])

  const handleAutoSave = useCallback((data: EventAutoSave) => {
    latestXmlRef.current = data.xml
  }, [])

  const handleExport = useCallback(
    (data: EventExport) => {
      const xml = data.xml || latestXmlRef.current
      // 由 draw.io 原生保存触发的导出，回调 onSave。
      if (data.message.parentEvent === 'save') {
        onSave?.(xml)
        return
      }
      // 否则解析挂起的 getXml Promise。
      if (exportResolverRef.current) {
        exportResolverRef.current(xml)
        exportResolverRef.current = null
      }
    },
    [onSave],
  )

  const getXml = useCallback((): Promise<string> => {
    // 优先返回 autosave 追踪到的最新 XML。
    if (latestXmlRef.current) {
      return Promise.resolve(latestXmlRef.current)
    }
    // 回退：请求导出并等待 onExport 事件。
    return new Promise<string>((resolve) => {
      exportResolverRef.current = resolve
      editorRef.current?.exportDiagram({ format: 'xmlsvg' })
    })
  }, [])

  const loadXml = useCallback((xml: string) => {
    setDiagramXml(xml)
    latestXmlRef.current = xml
  }, [])

  const newDiagram = useCallback(() => {
    setDiagramXml('')
    latestXmlRef.current = ''
    setError('')
  }, [])

  useImperativeHandle(ref, () => ({ getXml, loadXml, newDiagram }), [getXml, loadXml, newDiagram])

  const openAiDialog = useCallback(() => {
    setError('')
    setAiDialogOpen(true)
  }, [])

  const closeAiDialog = useCallback(() => {
    setAiDialogOpen(false)
    if (!generating) setError('')
  }, [generating])

  const handleGenerate = useCallback(async () => {
    const trimmedPrompt = prompt.trim()
    if (!trimmedPrompt || generating) return
    setGenerating(true)
    setError('')
    try {
      const res = await generateDiagram({ prompt: trimmedPrompt, context: context.trim() || undefined })
      if (res.xml) {
        setDiagramXml(res.xml)
        latestXmlRef.current = res.xml
        setAiDialogOpen(false)
        setPrompt('')
        setContext('')
      } else {
        setError(t('diagram.generateFailed'))
      }
    } catch (e) {
      console.error('[DiagramEditor] generateDiagram failed', e)
      setError(e instanceof Error ? e.message : t('diagram.generateFailed'))
    } finally {
      setGenerating(false)
    }
  }, [context, generating, prompt, t])

  return (
    <div className="relative flex h-full min-h-0 w-full flex-col">
      {/* 工具栏浮层 */}
      <div className="pointer-events-none absolute right-2 top-2 z-10 flex items-center gap-1.5">
        <Button
          type="button"
          size="sm"
          variant="secondary"
          className="pointer-events-auto gap-1.5 shadow-sm"
          onClick={openAiDialog}
          disabled={generating}
        >
          {generating ? <LoaderCircle className="h-3.5 w-3.5 animate-spin" /> : <Sparkles className="h-3.5 w-3.5" />}
          <span>{generating ? t('diagram.generating') : t('diagram.aiGenerate')}</span>
        </Button>
      </div>

      {/* 编辑器区域 */}
      <div className="relative min-h-0 flex-1 overflow-auto rounded-lg border border-[var(--nova-border)] bg-[var(--nova-surface)]">
        {!editorReady && (
          <div className="absolute inset-0 z-[5] flex items-center justify-center bg-[var(--nova-surface)] text-xs text-[var(--nova-text-muted)]">
            {t('diagram.editorLoading')}
          </div>
        )}
        <DrawIoEmbed
          ref={editorRef}
          xml={diagramXml}
          autosave
          urlParameters={{ ui: 'kennedy', noSaveBtn: true, spin: true, modified: false }}
          onLoad={handleLoad}
          onAutoSave={handleAutoSave}
          onExport={handleExport}
        />
      </div>

      {error && <InlineErrorNotice className="mt-2" message={error} />}

      {/* AI 生成对话框 */}
      <Dialog open={aiDialogOpen} onOpenChange={(open) => { if (open) openAiDialog(); else closeAiDialog() }}>
        <DialogContent className="max-w-[min(calc(100vw-2rem),36rem)]">
          <DialogHeader>
            <DialogTitle>{t('diagram.aiGenerateTitle')}</DialogTitle>
            <DialogDescription>{t('diagram.aiGenerateDescription')}</DialogDescription>
          </DialogHeader>

          <div className="flex flex-col gap-3">
            <div className="flex flex-col gap-1.5">
              <label className="text-xs font-medium text-[var(--nova-text-muted)]" htmlFor="diagram-prompt">
                {t('diagram.promptLabel')}
              </label>
              <Textarea
                id="diagram-prompt"
                value={prompt}
                onChange={(e) => setPrompt(e.target.value)}
                placeholder={t('diagram.promptPlaceholder')}
                minRows={3}
                maxRows={8}
                className="resize-none"
                disabled={generating}
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <label className="text-xs font-medium text-[var(--nova-text-muted)]" htmlFor="diagram-context">
                {t('diagram.contextLabel')}
              </label>
              <Input
                id="diagram-context"
                value={context}
                onChange={(e) => setContext(e.target.value)}
                placeholder={t('diagram.contextPlaceholder')}
                disabled={generating}
              />
            </div>

            {error && <InlineErrorNotice message={error} />}
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={closeAiDialog} disabled={generating}>
              {t('diagram.close')}
            </Button>
            <Button
              type="button"
              onClick={handleGenerate}
              disabled={!prompt.trim() || generating}
              className="gap-1.5"
            >
              {generating ? <LoaderCircle className="h-3.5 w-3.5 animate-spin" /> : <Sparkles className="h-3.5 w-3.5" />}
              <span>{generating ? t('diagram.generating') : t('diagram.generate')}</span>
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
})
