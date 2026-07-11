# 资料卡片索引系统

资料卡片索引系统已集成到 Aurora 主程序中，不再需要独立运行搜索服务。

## 集成方式

资料卡片索引系统的全部功能（n-gram 全文搜索、AI 卡片生成、文本导入、工作区搜索）已作为 Aurora 内置 API 提供，通过以下端点访问：

| API 端点 | 方法 | 说明 |
|---------|------|------|
| `/api/material-index/search` | GET | 全文搜索资料卡片（n-gram 中文索引，评分排序） |
| `/api/material-index/stats` | GET | 索引统计信息 |
| `/api/material-index/card/:id` | GET | 获取单张卡片详情 |
| `/api/material-index/templates` | GET | 列出可用卡片模板 |
| `/api/material-index/generate` | POST | AI 从文本提炼卡片并写入资料库 |
| `/api/material-index/rebuild` | POST | 重建索引 |
| `/api/material-index/import` | POST | 导入文本文件（md/txt） |
| `/api/material-index/imports` | GET | 列出已导入文件 |
| `/api/material-index/imports` | DELETE | 删除导入文件 |
| `/api/material-index/workspace-search` | GET | 搜索工作区文件（章节、设定等） |

## 使用方式

1. 启动 Aurora 主程序
2. 索引在首次访问搜索 API 时惰性构建
3. 所有数据通过 Aurora 的资料库 API 管理，无需独立服务

## 与 Aurora 资料库的关系

```
Aurora 主程序 (8080)
┌─────────────────────────────────────────┐
│  资料库 (lore items)                     │
│  /api/lore/items — CRUD                  │
│                                          │
│  资料卡片索引 (内置)                      │
│  /api/material-index/search — 全文搜索   │
│  /api/material-index/generate — AI 生成  │
│  /api/material-index/import — 文本导入   │
│                                          │
│  工作区搜索 (内置)                       │
│  /api/material-index/workspace-search    │
└─────────────────────────────────────────┘
```

## 目录结构

```
material-index/
├── config.toml               配置参考（主程序内置，此文件仅作文档）
├── imports/                  导入的文本文件（运行时数据）
└── cards/                    卡片文件备份（运行时数据）
```

## AI 配置

AI 卡片生成功能使用 Aurora 主配置 (`config.toml`) 中的模型配置：
- `openai_api_key` — API Key
- `openai_base_url` — API 基础 URL
- `openai_model` — 模型名称

也支持从 `[[model_profiles]]` 中 ID 为 `default` 的配置自动读取。

## Skills 集成

配套的 `skills/material-index/` Skill 提供了通过 Agent 进行卡片提炼、搜索和整理的能力，包括：
- 导入文本资料
- 按模板提炼卡片
- 搜索资料卡片
- 整理资料库
