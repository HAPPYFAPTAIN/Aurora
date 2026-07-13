import { jsonHeaders, requestJSON } from './client'

export type DiagramType = 'character' | 'timeline' | 'worldmap' | 'structure' | 'faction'

export interface DiagramGenerateRequest {
  type: DiagramType
}

export interface DiagramGenerateResponse {
  xml: string
}

/** 调用后端 AI 生成图表接口，返回 Mermaid 代码。 */
export async function generateDiagram(req: DiagramGenerateRequest): Promise<DiagramGenerateResponse> {
  return requestJSON('/api/diagrams/generate', {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify({ type: req.type }),
  })
}
