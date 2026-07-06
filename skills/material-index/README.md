# 资料卡片索引插件 (material-index)

Aurora 原生插件，提供文本资料导入、AI 卡片提炼、资料库搜索和卡片整理功能。

## 这是什么

本插件是一个 **Aurora Skill**，安装后 Aurora 的写作 Agent (`ide`) 和配置 Agent (`config_manager`) 即可在对话中直接使用以下能力：

- **导入文本**：将 md/txt 格式的资料导入到工作区
- **AI 提炼卡片**：Agent 自身作为 AI 大模型，按模板将资料提炼为结构化知识卡片
- **写入资料库**：通过 `write_lore_items` 将卡片写入 Aurora 资料库，写作时自动召回
- **搜索卡片**：通过 `list_lore_items` + `read_lore_items` 搜索已有卡片
- **整理卡片**：对照新资料与现有卡片，进行合并、拆分、补充、去重

无需启动任何额外服务，安装 Skill 即生效。

## 安装

插件已位于 `skills/material-index/` 目录，Aurora 启动时自动加载。

## 使用方式

在 Aurora 写作模式或配置模式中，直接对 Agent 说：

- "导入这段资料并提炼卡片" + 粘贴文本
- "从 `material-index/imports/xxx.md` 提炼人物卡"
- "搜索资料库中关于沈家禄的卡片"
- "把这份新资料和现有资料库对照整理一下"

Agent 会按照 Skill 定义的工作流程执行。

## 卡片模板

模板位于 `skills/material-index/templates/` 目录：

| 文件 | 类型 | 说明 |
|------|------|------|
| character.md | 人物卡 | 角色身份、人设、背景、能力 |
| event.md | 事件卡 | 重要事件、剧情节点 |
| location.md | 地点卡 | 长期反复出现的地点 |
| world.md | 世界观卡 | 世界类型、时代、秩序 |
| faction.md | 势力卡 | 组织、阵营、利益关系 |
| rule.md | 规则卡 | 能力体系、世界规则、禁忌 |
| item.md | 物品卡 | 关键物品、道具、线索 |
| concept.md | 概念卡 | 核心概念、主题、隐喻 |
| analysis.md | 分析卡 | 素材分析、解读、提炼 |

## 文件结构

```
skills/material-index/
├── SKILL.md              # Skill 定义（工作流程和规则）
├── README.md             # 本文件
└── templates/            # 卡片模板
    ├── character.md
    ├── event.md
    ├── location.md
    ├── world.md
    ├── faction.md
    ├── rule.md
    ├── item.md
    ├── concept.md
    └── analysis.md

material-index/                # 工作区数据目录（运行时自动创建）
├── imports/               # 导入的原始资料
└── cards/                 # AI 生成的卡片文件（备份用，主要存入资料库）
```

## 启用/禁用

- **禁用**：在 Aurora 设置页 → Agents → Skills 中关闭 `material-index`
- **启用**：重新打开即可
- **移除**：删除 `skills/material-index/` 目录

## 与资料库的关系

本插件的核心存储是 **Aurora 资料库**（`.nova/<book>/.nova/lore/items.json`）。卡片通过 `write_lore_items` 写入资料库后：

- 写作 Agent 在创作时自动召回相关卡片
- 互动模式 Agent 在故事中自动参考相关设定
- 可在资料库界面查看、编辑、删除卡片

`material-index/imports/` 和 `material-index/cards/` 目录是辅助文件存储，用于保留原始资料和卡片备份。

## 可选：搜索增强服务

如果需要更强的全文搜索能力（n-gram 索引、跨数据源搜索），可以启用可选的搜索增强服务。详见 `material-index/` 目录下的说明。

搜索增强服务是可选的，不启用也不影响插件的核心功能。
