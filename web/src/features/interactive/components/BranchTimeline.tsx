import { useMemo, useState } from 'react'
import { ChevronDown, ChevronUp, GitBranch, GitCommitHorizontal, Plus } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import type { BranchSummary, PlotNode, Snapshot } from '../types'

interface BranchTimelineProps {
  snapshot: Snapshot | null
  branches: BranchSummary[]
  currentBranchId: string
  onSwitchBranch: (branchId: string) => void
  onCreateBranch: (turnId: string, title: string) => void
}

export function BranchTimeline({ snapshot, branches, currentBranchId, onSwitchBranch, onCreateBranch }: BranchTimelineProps) {
  const [expanded, setExpanded] = useState(false)
  const [selectedNode, setSelectedNode] = useState<PlotNode | null>(null)
  const [branchTitle, setBranchTitle] = useState('')
  const graphNodes = snapshot?.graph?.nodes || []
  const graphBranches = snapshot?.graph?.branches?.length ? snapshot.graph.branches : branches
  const branchRows = useMemo(() => {
    const rows = new Map<string, PlotNode[]>()
    for (const branch of graphBranches) rows.set(branch.id, [])
    for (const node of graphNodes) {
      if (!rows.has(node.branch_id)) rows.set(node.branch_id, [])
      rows.get(node.branch_id)?.push(node)
    }
    return Array.from(rows.entries()).map(([branchId, nodes]) => ({
      branchId,
      branch: graphBranches.find((branch) => branch.id === branchId),
      nodes,
    }))
  }, [graphBranches, graphNodes])

  const openCreateDialog = (node: PlotNode) => {
    setSelectedNode(node)
    setBranchTitle(`基于「${node.title}」的新剧情线`)
  }

  const submitCreateBranch = () => {
    if (!selectedNode) return
    onCreateBranch(selectedNode.id, branchTitle.trim() || '新剧情线')
    setSelectedNode(null)
    setBranchTitle('')
  }

  return (
    <div className={`${expanded ? 'h-[196px]' : 'h-[52px]'} border-t border-[#2f3540] bg-[#14171c] px-4 py-3 transition-[height]`}>
      <div className="flex items-center justify-between gap-3 text-xs text-[#858b96]">
        <button type="button" className="flex items-center gap-1.5 font-medium text-[#8f98a8] hover:text-[#dbe3ef]" onClick={() => setExpanded(!expanded)}>
          <GitBranch className="h-3.5 w-3.5 text-[#7fb7e8]" />
          剧情节点图
          {expanded ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronUp className="h-3.5 w-3.5" />}
        </button>
        <div className="flex min-w-0 flex-1 items-center justify-end gap-2">
          <span className="truncate text-[#737d8d]">{graphNodes.length || snapshot?.turns?.length || 0} 个剧情节点</span>
          <div className="flex max-w-[55%] gap-2 overflow-hidden">
            {graphBranches.map((branch) => (
              <Button key={branch.id} variant={branch.id === currentBranchId ? 'default' : 'outline'} size="xs" className={branch.id === currentBranchId ? 'bg-[#2d6fb8] hover:bg-[#347dca]' : 'border-[#343b47] bg-[#20242b] text-[#aab2c0] hover:bg-[#252831]'} onClick={() => onSwitchBranch(branch.id)}>
                {branch.title || (branch.id === 'main' ? '主线' : branch.id)}
              </Button>
            ))}
          </div>
        </div>
      </div>
      {expanded && (
        <ScrollArea className="mt-4 h-[128px] w-full">
          <div className="min-w-max space-y-2 pr-4">
            {branchRows.map(({ branchId, branch, nodes }) => (
              <div key={branchId} className="grid min-h-8 grid-cols-[112px_1fr] items-center gap-3">
                <button
                  type="button"
                  className={`truncate text-left text-[11px] font-medium ${branchId === currentBranchId ? 'text-[#dbe7ff]' : 'text-[#8792a3] hover:text-[#dbe3ef]'}`}
                  onClick={() => onSwitchBranch(branchId)}
                  title={branch?.title || branchId}
                >
                  {branch?.title || (branchId === 'main' ? '主线' : branchId)}
                </button>
                <div className="flex items-center">
                  {nodes.map((node, index) => (
                    <div key={node.id} className="flex items-center">
                      {index > 0 && <span className="h-px w-12 bg-[#3a465a]" />}
                      <button
                        type="button"
                        className={`group relative flex h-7 min-w-[72px] items-center gap-1.5 rounded border px-2 text-left transition ${node.current ? 'border-[#6aa8ff] bg-[#233856] text-[#edf5ff]' : 'border-[#343b47] bg-[#20242b] text-[#aab2c0] hover:border-[#4a5d79] hover:bg-[#252c38]'}`}
                        onClick={() => openCreateDialog(node)}
                        title={`${node.title}\n${node.summary}`}
                      >
                        <GitCommitHorizontal className={`h-3.5 w-3.5 ${node.head ? 'text-[#7fb7e8]' : 'text-[#697487]'}`} />
                        <span className="max-w-[120px] truncate text-[11px]">{node.title}</span>
                        {node.current && <Badge className="ml-0.5 h-4 rounded-sm bg-[#2d6fb8] px-1 text-[9px]">当前</Badge>}
                      </button>
                    </div>
                  ))}
                  {nodes.length === 0 && <span className="text-xs text-[#858b96]">还没有剧情节点。</span>}
                </div>
              </div>
            ))}
            {branchRows.length === 0 && <span className="text-xs text-[#858b96]">还没有剧情节点，输入第一句话开始。</span>}
          </div>
        </ScrollArea>
      )}
      <Dialog open={!!selectedNode} onOpenChange={(open) => { if (!open) setSelectedNode(null) }}>
        <DialogContent className="border-[#303238] bg-[#202329] text-[#d7dbe2]">
          <DialogHeader>
            <DialogTitle>从此节点创建剧情线</DialogTitle>
            <DialogDescription className="text-[#9aa4b5]">
              {selectedNode ? `将从「${selectedNode.title}」分叉，创建后故事舞台会切换到新剧情线。` : ''}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Input className="border-[#3a3d45] bg-[#17191d] text-sm" value={branchTitle} onChange={(event) => setBranchTitle(event.target.value)} placeholder="剧情线名称" />
            {selectedNode?.summary && <div className="rounded-md border border-[#303743] bg-[#17191d] p-2 text-xs leading-5 text-[#aab2c0]">{selectedNode.summary}</div>}
          </div>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setSelectedNode(null)}>取消</Button>
            <Button className="gap-1.5 bg-[#2d6fb8] hover:bg-[#347dca]" onClick={submitCreateBranch}>
              <Plus className="h-4 w-4" />
              创建并切换
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
