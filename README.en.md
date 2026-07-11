<p align="center">
  <img src="./web/favicon.svg" alt="Denova Aurora icon" width="76" height="76">
</p>

<p align="center">
  <strong>Denova Aurora — An AI creative platform for novel writing and AI-generated RPGs, with built-in AI Agents, Skills, TTS voice synthesis, streaming audio, image generation, automations, and version control.</strong>
</p>

<p align="center">
  English | <a href="README.md">中文</a>
</p>

<p align="center">
  <a href="https://github.com/HAPPYFAPTAIN/Denova-Modified-Version-Aurora/releases"><img alt="Release" src="https://img.shields.io/github/v/release/HAPPYFAPTAIN/Denova-Modified-Version-Aurora?style=flat-square"></a>
  <a href="./LICENSE"><img alt="License" src="https://img.shields.io/github/license/HAPPYFAPTAIN/Denova-Modified-Version-Aurora?style=flat-square"></a>
  <img alt="Go" src="https://img.shields.io/badge/Go-1.26%2B-00ADD8?style=flat-square&logo=go&logoColor=white">
  <img alt="Node.js" src="https://img.shields.io/badge/Node.js-20%2B-5FA04E?style=flat-square&logo=nodedotjs&logoColor=white">
</p>

<p align="center">
  Current version: <strong>v0.1.19</strong> (2026-07-11) · Beta
</p>

<p align="center">
  Forked from <a href="https://github.com/alfredxw/denova">alfredxw/denova</a> (original author <a href="https://github.com/alfredxw">@alfredxw</a>), licensed under Apache-2.0
</p>

---

## Why Aurora

Aurora is designed for long-form creative projects and interactive entertainment. It combines a writing IDE, interactive storytelling, structured lore library, Agent tool calls, TTS voice synthesis, image generation, automations, and local version control in a single workspace.

## Core Features

- **Writing Mode**: Novel-focused Markdown editor with multi-tab, global search, chapter stats, outlines, progress tracking, and novel import.
- **Creative Agent**: Reads selections, files, and lore library; generates or modifies chapters via tools; adapts to different writing tasks via Skills / SubAgents.
- **Game Mode**: Interactive text adventures with player input, story branches, scene memory, story director with goals, pressure, costs, event packs, and TRPG dice checks.
- **TTS Voice Synthesis**: Supports OpenAI-compatible TTS and Step Fun TTS (stepaudio-2.5-tts). Auto-chunks long text, SSE streaming playback, auto-fetch voice list, voice style instructions, editor toolbar read-aloud button.
- **Lore Library & Presets**: Structured settings for characters, worldbuilding, locations, factions, rules, items; pluggable narrative styles, event packs, TRPG checks, state systems, and image schemes.
- **Image Generation**: Chapter illustrations, interactive images, and book covers via OpenAI-compatible image models.
- **Context Management**: Progressive context organization with Memory Compact, cache optimization, and bounded tool results.
- **Async Memory System**: Background goroutine + channel pipeline with SSE progress streaming, preventing token overflow.
- **Version Control**: Local Git-based versioning with diff, restore, auto-save, and post-Agent saves.
- **Automation**: Scheduled tasks, review, auto-continuation, and custom prompt workflows.
- **Production Quality**: Bilingual UI (CN/EN), light/dark themes, remote access, PWA mobile support, cross-platform (Windows / macOS / Linux).
- **Material Card Index**: Full-text search service + AI card extraction + 9 preset templates for batch import from text sources.
- **De-AI Editing**: Built-in Skill for eliminating AI writing traces in Chinese novels.

## Differences from upstream denova

| Feature | Description |
|---------|-------------|
| **TTS Voice Synthesis** | OpenAI-compatible + Step Fun TTS, auto-chunking, SSE streaming playback, 32 Step Fun + 10 OpenAI voices, voice style instructions, editor read-aloud |
| **Lifecycle Hooks** | Priority-ordered callbacks at four Agent lifecycle stages for context injection, tool result truncation, memory triggers |
| **Async Memory Worker** | Background goroutine pipeline with SSE progress, preventing token overflow |
| **Material Card Index** | Full-text search + AI extraction + 9 templates for batch import |
| **De-AI Editing Skill** | AI writing trace elimination for Chinese novels |
| **Chinese Humanizer Skill** | AI-style text detection and rewriting, AIGC reduction |

## Quick Start

### Download Release

Download from [GitHub Releases](https://github.com/HAPPYFAPTAIN/Denova-Modified-Version-Aurora/releases), extract and run:

```bash
./denova
```

macOS security workaround:

```bash
xattr -dr com.apple.quarantine denova
```

### Build from Source

Requires Go 1.26+, Node.js 20+, and pnpm.

```bash
git clone https://github.com/HAPPYFAPTAIN/Denova-Modified-Version-Aurora.git
cd Denova-Modified-Version-Aurora/aurora-src
corepack enable
./bootstrap.sh
```

## License

[Apache-2.0](./LICENSE)

Forked from [alfredxw/denova](https://github.com/alfredxw/denova). Lifecycle hooks and async memory design inspired by [agentscope-ai/QwenPaw](https://github.com/agentscope-ai/QwenPaw) (Apache-2.0).
