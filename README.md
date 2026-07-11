<p align="center">
  <img src="./web/favicon.svg" alt="Denova Aurora 图标" width="76" height="76">
</p>

<p align="center">
  <strong>Denova Aurora — 面向小说创作与 AI 角色扮演游戏的 AI 创作平台，内置 AI Agents、Skills、TTS 语音朗读、流式合成、图像生成、自动化与项目版本管理。</strong>
</p>

<p align="center">
  <a href="README.en.md">English</a> | 中文
</p>

<p align="center">
  <a href="https://github.com/HAPPYFAPTAIN/Denova-Modified-Version-Aurora/releases"><img alt="Release" src="https://img.shields.io/github/v/release/HAPPYFAPTAIN/Denova-Modified-Version-Aurora?style=flat-square"></a>
  <a href="./LICENSE"><img alt="License" src="https://img.shields.io/github/license/HAPPYFAPTAIN/Denova-Modified-Version-Aurora?style=flat-square"></a>
  <img alt="Go" src="https://img.shields.io/badge/Go-1.26%2B-00ADD8?style=flat-square&logo=go&logoColor=white">
  <img alt="Node.js" src="https://img.shields.io/badge/Node.js-20%2B-5FA04E?style=flat-square&logo=nodedotjs&logoColor=white">
</p>

<p align="center">
  当前版本：<strong>v0.1.19</strong>（2026-07-11） · Beta
</p>

<p align="center">
  Fork 自 <a href="https://github.com/alfredxw/denova">alfredxw/denova</a>（原作者 <a href="https://github.com/alfredxw">@alfredxw</a>），基于 Apache-2.0 协议开源
</p>

---

## 为什么选择 Aurora

Aurora 面向长期创作项目和互动娱乐，把写作 IDE、互动故事、结构化资料库、Agent 工具调用、TTS 语音朗读、图像生成、自动化和本地版本管理放在同一个项目工作区里，让创作过程可以反复迭代、回溯和沉淀。

你可以从原创灵感开始，也可以导入已有小说做同人、改编或续写；还可以导入 AI 酒馆角色卡，快速搭建互动文字冒险。模型上下文会按来源、用途和大小上限组织，避免把完整历史、日志或全部设定无界塞进下一轮对话。

## 核心能力

- **写作模式**：面向小说创作，支持 Markdown 编辑、多 Tab、全局搜索、章节统计、大纲、章节组细纲、进度追踪和现有小说导入。
- **创作 Agent**：可读取选区、文件和资料库，调用工具生成或修改章节，并通过 Skills / SubAgents 适配不同写作任务、文风和工作流。
- **游戏模式**：运行互动文字冒险，支持玩家输入、剧情分支、故事线切换、行动建议、场景记忆、长期故事记忆，以及由故事导演驱动的目标、压力、代价、事件卡包和规则检定。
- **TTS 语音朗读**：支持 OpenAI 兼容 TTS 和阶跃星辰 Step Fun TTS，长文本自动分段合成、SSE 流式边生成边播放、音色列表自动获取、编辑器朗读整篇文章。
- **资料库与预设**：沉淀角色、世界观、地点、势力、规则、物品等稳定设定；叙事风格负责文风、提示词槽位和场景风格，故事导演可插拔组合叙事风格、事件包、TRPG 检定、状态系统、Story Memory Structure、开局选择器和图像方案，且每个模块都可独立关闭。
- **图像创作**：支持章节插画、互动图像和书籍封面生成，复用 OpenAI 兼容图像模型配置，并在界面中预览和管理结果。
- **上下文管理**：渐进式组织模型可见上下文，支持 Memory Compact、缓存优化和有界工具结果，降低长篇创作的上下文噪音与 token 成本。
- **异步记忆系统**：后台 goroutine + channel 串行处理记忆任务，将单次大型 LLM 调用拆分为分块管道，每个分块完成后通过 SSE 实时推送进度，避免 token 溢出和输出截断。
- **版本与恢复**：基于本地 Git 保存版本、查看 Diff、恢复历史，并支持定时保存和 Agent 大量输出后的自动保存。
- **自动化**：支持定时任务、Review、自动续写和自定义 Prompt 工作流。
- **产品化体验**：中英文界面、浅色/深色主题、OpenAI 兼容模型配置、远程访问、PWA 手机使用，以及 Windows / macOS / Linux 全平台支持。
- **资料卡片索引**：内置资料卡片导入、AI 提炼和全文搜索增强系统，支持从文本素材自动生成结构化知识卡片并写入资料库。
- **去AI味改稿**：内置去AI味改稿写手 Skill，针对中文小说正文消除 AI 写作痕迹，修正排版、标点、病句，保留原意和叙事节奏。

## 与原版 denova 的差异

Aurora Fork 自 [alfredxw/denova](https://github.com/alfredxw/denova)，在保持与上游同步的基础上，额外引入了以下特性：

| 特性 | 说明 |
|------|------|
| **TTS 语音朗读** | 支持 OpenAI 兼容 TTS 和阶跃星辰 Step Fun TTS（stepaudio-2.5-tts），长文本自动分段合成、SSE 流式边生成边播放、音色列表自动获取（32 个 Step Fun 音色 + 10 个 OpenAI 音色）、语音风格指令（instruction）、编辑器工具栏朗读按钮 |
| **生命周期钩子系统** | 在 Agent 运行的四个阶段注入按优先级排序的自定义回调，支持上下文注入、工具结果截断、记忆触发 |
| **异步记忆 Worker** | 后台 goroutine + channel 串行处理记忆任务，分块管道 + SSE 实时进度推送，避免 token 溢出 |
| **资料卡片索引系统** | 全文搜索服务 + AI 卡片提炼 + 9 种预设模板，支持从文本素材批量导入 |
| **去AI味改稿 Skill** | 针对中文小说正文的 AI 写作痕迹消除，修正排版、标点、病句 |
| **中文人性化 Skill** | 检测和重写 AI 风格中文文本，支持学术 AIGC 降重和风格转换 |

## 快速开始

### 下载 Release

从 [GitHub Releases](https://github.com/HAPPYFAPTAIN/Denova-Modified-Version-Aurora/releases) 下载对应平台压缩包，解压后运行：

```bash
./denova
```

Windows 用户运行 `denova.exe`。macOS 如果提示安全限制，可以执行：

```bash
xattr -dr com.apple.quarantine denova
```

### 从源码运行

需要 Go 1.26+、Node.js 20+ 和 pnpm。

```bash
git clone https://github.com/HAPPYFAPTAIN/Denova-Modified-Version-Aurora.git
cd Denova-Modified-Version-Aurora/aurora-src
corepack enable
./bootstrap.sh
```

默认地址：

- 前端：`http://localhost:5173`
- 后端：`http://localhost:8080`

## 模型与配置

Aurora 使用 OpenAI 兼容接口。推荐先在设置页配置语言模型、图像模型、TTS 模型、Agent 参数、默认写作 Skill、编辑器、游戏模式、版本管理、语言、主题和字体。

配置优先级：

```text
内置默认值 < 全局 config.toml < 用户级配置 < 工作区级配置 < 环境变量
```

### TTS 语音朗读配置

在 **设置页 → TTS API** 中配置：

1. 选择 Provider（OpenAI 兼容 / 阶跃星辰 Step Fun）
2. 填入 API Key、Base URL、Model
3. 点击音色下拉按钮自动获取可用音色列表
4. 可选填写语音风格指令（如"语气温柔，语速偏慢"）

配置完成后，在聊天消息或编辑器工具栏点击喇叭图标即可朗读。

## 远程访问与手机使用

在 **设置页 → 远程访问** 开启「允许局域网访问」并设置用户名和密码后，其他设备可以打开设置页展示的访问地址。手机浏览器登录后可添加到主屏幕，以接近独立应用的方式使用。

如果要通过公网或域名访问，建议使用 Caddy / Nginx 等反向代理提供 HTTPS：

```text
aurora.example.com {
    reverse_proxy 127.0.0.1:8080
}
```

## 开发

启动前后端：

```bash
./bootstrap.sh
```

分开启动前端或后端：

```bash
./bootstrap.sh fe
./bootstrap.sh be
```

## 赞助项目

> 给项目冲点 token，帮助这个项目持续迭代，持续开源！

<p align="center">
  <img src="./web/donate.png" alt="捐赠" width="240">
</p>

## Star History

<a href="https://www.star-history.com/#HAPPYFAPTAIN/Denova-Modified-Version-Aurora&type=date&legend=top-left">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=HAPPYFAPTAIN/Denova-Modified-Version-Aurora&type=date&theme=dark&legend=top-left" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=HAPPYFAPTAIN/Denova-Modified-Version-Aurora&type=date&legend=top-left" />
   <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=HAPPYFAPTAIN/Denova-Modified-Version-Aurora&type=date&legend=top-left" />
 </picture>
</a>

## License

[Apache-2.0](./LICENSE)

本项目 Fork 自 [alfredxw/denova](https://github.com/alfredxw/denova)，感谢原作者 [@alfredxw](https://github.com/alfredxw) 的开源贡献。生命周期钩子与异步记忆系统的设计参考了 [agentscope-ai/QwenPaw](https://github.com/agentscope-ai/QwenPaw)（Apache-2.0）。
