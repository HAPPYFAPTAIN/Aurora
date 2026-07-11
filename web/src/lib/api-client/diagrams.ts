import { jsonHeaders, requestJSON } from './client'

export interface DiagramGenerateRequest {
  prompt: string
  context?: string
}

export interface DiagramGenerateResponse {
  xml: string
}

/** 调用后端 AI 生成图表接口，返回 draw.io XML。 */
export async function generateDiagram(req: DiagramGenerateRequest): Promise<DiagramGenerateResponse> {
  return requestJSON('/api/diagrams/generate', {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify({ prompt: req.prompt, context: req.context || '' }),
  })
}
