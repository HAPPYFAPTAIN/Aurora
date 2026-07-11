import { useCallback, useRef, useState, type ChangeEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { FilePlus, FolderOpen, LoaderCircle, Save, XIcon } from 'lucide-react'
import { toast } from 'sonner'
import { saveFile } from '@/lib/api-client/workspace'
import { DiagramEditor, type DiagramEditorHandle } from './DiagramEditor'
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
import { TooltipIconButton } from '@/components/common/tooltip-icon-button'
import { InlineErrorNotice } from '@/components/common/inline-error-notice'

interface DiagramPanelProps {
  /** 关闭面板回调。 */
  onClose?: () => void
}

/** 工作区内图表文件的存放目录。 */
const DIAGRAMS_DIR = 'diagrams'
const DIAGRAM_EXTENSION = '.drawio'

/** 将用户输入的文件名转换为工作区内 diagrams/ 目录下的相对路径。 */
function buildDiagramPath(name: string): string {
  const safe = name.trim().replace(/[\\/]+/g, '-').replace(/\.drawio$/i, '')
  return `${DIAGRAMS_DIR}/${safe || 'untitled'}${DIAGRAM_EXTENSION}`
}

/**
 * DiagramPanel 是图表编辑器的面板容器，
 * 提供标题栏、工具栏（新建 / 保存 / 从文件打开）和 DiagramEditor 内容区。
 * 图表文件保存在工作区的 diagrams/ 目录下。
 */
export function DiagramPanel({ onClose }: DiagramPanelProps) {
  const { t } = useTranslation()
  const editorRef = useRef<DiagramEditorHandle>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [currentPath, setCurrentPath] = useState<string>('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [saveDialogOpen, setSaveDialogOpen] = useState(false)
  const [fileName, setFileName] = useState('')

  const saveToPath = useCallback(
    async (path: string, xml?: string) => {
      setSaving(true)
      setError('')
      try {
        const content = xml ?? (await editorRef.current?.getXml()) ?? ''
        await saveFile(path, content)
        setCurrentPath(path)
        toast.success(t('diagram.savedTo', { path }))
      } catch (e) {
        console.error('[DiagramPanel] save diagram failed', e)
        const msg = e instanceof Error ? e.message : t('diagram.saveFailed')
        setError(msg)
        toast.error(msg)
      } finally {
        setSaving(false)
      }
    },
    [t],
  )

  const handleNew = useCallback(() => {
    editorRef.current?.newDiagram()
    setCurrentPath('')
    setError('')
  }, [])

  const handleSaveClick = useCallback(() => {
    if (saving) return
    // 已有当前文件路径时直接保存。
    if (currentPath) {
      void saveToPath(currentPath)
      return
    }
    // 否则弹出文件名输入对话框。
    setFileName('')
    setError('')
    setSaveDialogOpen(true)
  }, [currentPath, saving, saveToPath])

  const confirmSave = useCallback(() => {
    const name = fileName.trim()
    if (!name) return
    const path = buildDiagramPath(name)
    setSaveDialogOpen(false)
    void saveToPath(path)
  }, [fileName, saveToPath])

  const handleOpenFile = useCallback(() => {
    fileInputRef.current?.click()
  }, [])

  const handleFileSelected = useCallback(
    async (e: ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0]
      if (!file) return
      try {
        const text = await file.text()
        editorRef.current?.loadXml(text)
        setCurrentPath('')
        setError('')
      } catch (err) {
        console.error('[DiagramPanel] open file failed', err)
        const msg = err instanceof Error ? err.message : t('diagram.openFailed')
        setError(msg)
        toast.error(msg)
      } finally {
        // 重置 input，以便重复选择同一文件。
        if (fileInputRef.current) fileInputRef.current.value = ''
      }
    },
    [t],
  )

  return (
    <div className="nova-sidebar flex h-full min-h-0 flex-col text-xs text-[var(--nova-text-muted)]">
      {/* 标题栏 */}
      <div className="nova-topbar flex h-9 shrink-0 items-center gap-1 border-b px-3">
        <span className="font-semibold text-[var(--nova-text)]">{t('diagram.title')}</span>
        <div className="ml-auto flex items-center gap-1">
          <TooltipIconButton label={t('diagram.new')} onClick={handleNew}>
            <FilePlus className="h-3.5 w-3.5" />
          </TooltipIconButton>
          <TooltipIconButton label={t('diagram.save')} onClick={handleSaveClick} disabled={saving}>
            {saving ? <LoaderCircle className="h-3.5 w-3.5 animate-spin" /> : <Save className="h-3.5 w-3.5" />}
          </TooltipIconButton>
          <TooltipIconButton label={t('diagram.openFile')} onClick={handleOpenFile}>
            <FolderOpen className="h-3.5 w-3.5" />
          </TooltipIconButton>
          {onClose && (
            <TooltipIconButton label={t('diagram.close')} onClick={onClose}>
              <XIcon className="h-3.5 w-3.5" />
            </TooltipIconButton>
          )}
        </div>
      </div>

      {/* 内容区：图表编辑器 */}
      <div className="min-h-0 flex-1 p-2">
        <DiagramEditor ref={editorRef} />
      </div>

      {error && <div className="px-3 pb-2"><InlineErrorNotice message={error} /></div>}

      {/* 隐藏的文件选择 input */}
      <input
        ref={fileInputRef}
        type="file"
        accept=".drawio,.xml,.svg"
        className="hidden"
        onChange={handleFileSelected}
      />

      {/* 保存文件名输入对话框 */}
      <Dialog open={saveDialogOpen} onOpenChange={setSaveDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('diagram.save')}</DialogTitle>
            <DialogDescription>{t('diagram.fileNamePrompt')}</DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-1.5">
            <label className="text-xs font-medium text-[var(--nova-text-muted)]" htmlFor="diagram-filename">
              {t('diagram.fileNameLabel')}
            </label>
            <Input
              id="diagram-filename"
              value={fileName}
              onChange={(e) => setFileName(e.target.value)}
              placeholder={t('diagram.fileNamePlaceholder')}
              onKeyDown={(e) => { if (e.key === 'Enter') confirmSave() }}
              autoFocus
            />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setSaveDialogOpen(false)}>
              {t('diagram.close')}
            </Button>
            <Button type="button" onClick={confirmSave} disabled={!fileName.trim()}>
              {t('diagram.confirm')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
