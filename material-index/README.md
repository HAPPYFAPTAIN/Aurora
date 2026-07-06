# 资料卡片搜索增强服务

直接对接 Aurora 资料库 API 的搜索增强前端。不读文件，所有数据通过 Aurora 的 `/api/lore/items` API 获取，写入也通过 API 回传，与 Aurora 完全同步。

## 与 Aurora 的关系

```
Aurora (8080)                    搜索增强服务 (8927)
┌─────────────────┐              ┌──────────────────┐
│  资料库 API     │ <── 读取 ── │  n-gram 索引     │
│  /api/lore/items│              │  搜索 + 评分     │
│                 │ ── 写回 ──> │  详情披露        │
│  workspace API  │ <── 代理 ── │  工作区搜索      │
│  /api/workspace │              │  导入 + AI 生成  │
└─────────────────┘              └──────────────────┘
```

## 启动

**前置条件**：Aurora 必须正在运行（搜索服务依赖 Aurora 的 API）。

```bash
# 编译
cd search-server
go build -o search-server.exe

# 启动
双击 启动搜索服务.bat
```

浏览器打开 **http://localhost:8927**

## 功能

| 功能 | 说明 |
|------|------|
| 全文搜索 | n-gram 中文索引，多关键词，评分排序 |
| 详情披露 | 点击卡片头展开完整内容、关键词、简介 |
| 关键词高亮 | 搜索结果中匹配的关键词高亮显示 |
| 工作区搜索 | 同时搜索章节正文和设定文件 |
| 类型筛选 | 按人物/地点/世界观等类型过滤 |
| 文本导入 | 拖拽 md/txt 文件到导入区 |
| AI 卡片生成 | 按模板提炼文本，直接写入 Aurora 资料库 |
| 卡片删除 | 从 Aurora 资料库删除卡片 |

## 数据流

所有操作都通过 Aurora API：

- **读取**：`GET http://localhost:8080/api/lore/items` → 构建索引 → 搜索
- **创建**：`POST http://localhost:8080/api/lore/items` → AI 生成的卡片直接写入资料库
- **更新**：`PATCH http://localhost:8080/api/lore/items/:id`
- **删除**：`DELETE http://localhost:8080/api/lore/items/:id`
- **工作区搜索**：`GET http://localhost:8080/api/workspace/search`

搜索服务不直接读写 `items.json`，所有变更都通过 Aurora API，确保数据一致性。

## API Key 配置

AI 生成功能需要 API Key。配置来源（按优先级）：

1. 环境变量 `OPENAI_API_KEY`
2. 本文件 `config.toml` 中的 `openai_api_key`
3. Aurora `config.toml` 中 `[[model_profiles]]` 的 `openai_api_key`

如果 Aurora 的 config.toml 中 `openai_api_key` 为空（通过 UI 配置的情况），需要在本文件或环境变量中单独设置。

## 文件结构

```
material-index/
├── config.toml               搜索服务配置
├── 启动搜索服务.bat
├── 编译搜索服务.bat
├── imports/                  导入的文本文件
├── cards/                    （备份用，主要存储在资料库）
└── search-server/             Go 程序
    ├── main.go
    ├── config.go
    ├── index.go
    ├── search.go
    ├── cardgen.go
    ├── go.mod
    └── dist/                  前端
        ├── index.html
        ├── style.css
        └── app.js
```

## 模块管理

| 操作 | 方法 |
|------|------|
| 启动 | 先启动 Aurora，再运行 启动搜索服务.bat |
| 关闭 | 关闭进程 |
| 移除 | 删除 `search-server/` 目录 |
