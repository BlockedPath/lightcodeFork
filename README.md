<h1 align="center">Lightcode</h1>
<p align="center">A lightweight terminal coding agent written in Go</p>

<p align="center">
  <img alt="Go" src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go&logoColor=white" />
  <img alt="Bubble Tea" src="https://img.shields.io/badge/TUI-Bubble%20Tea-FFB300?style=flat-square" />
  <img alt="OpenAI Compatible" src="https://img.shields.io/badge/API-OpenAI%20Compatible-10A37F?style=flat-square" />
  <a href="LICENSE"><img alt="License" src="https://img.shields.io/badge/License-MIT-black?style=flat-square" /></a>
</p>

![Lightcode demo](assets/lightcode.gif)

---

## What is Lightcode?

Lightcode is a terminal-based coding agent for developers. It connects to any OpenAI-compatible model provider.

## Features

- **OpenAI-compatible** — works with any provider that speaks the OpenAI Chat Completions API (OpenAI, Anthropic via proxy, Ollama, LM Studio, etc.)
- **Low memory usage** — uses less ram around 20-30 mb
- **Skills** — uses specialized [agent-skills](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview).
- **Multi-model support** — configure multiple providers and switch between models with `/models` inside the TUI


## Requirements

- [Go](https://go.dev/dl/) **1.25+**
- At least one OpenAI-compatible endpoint configured


## Install

```bash
go install github.com/Kartik-2239/lightcode/cmd/lightcode@latest
```


## Quick Start

Run the **TUI** and **API server** together (defaults to `:8080`):

```bash
lightcode
```

Or run directly from source:

```bash
go run ./cmd/lightcode/main.go
```

On first run, Lightcode creates `~/.lightcode/` with a default `config.json`. Add your model provider details (see [Configuration](#configuration)) and you're ready to go.


## Configuration

All settings live under **`~/.lightcode/config.json`**. The file is created automatically on first run.

### Full example

```json
{
  "theme": "light",
  "skills_path": "~/.lightcode/skills",
  "port": "8080",
  "providers": [
    {
      "models": ["gpt-5.5"],
      "base_url": "https://api.openai.com/v1",
      "api_key": "sk-..."
    },
    {
      "models": ["some-ai-800b"],
      "base_url": "https://your-gateway.example/v1",
      "api_key": "your-api-key"
    }
  ]
}
```

### Config reference

| Key | Default | Description |
|-----|---------|-------------|
| `theme` | `"light"` | UI theme — `"light"` or `"dark"` |
| `skills_path`| `~/.lightcode/skills` | Path to your skills directory (or change it to another skill path) |
| `port` | `"8080"` | Port for the local HTTP API server |
| `providers` | `[]` | List of model providers (see below) |

### Providers

Each entry in the `providers` array requires:

| Key | Description |
|-----|-------------|
| `models` | List of model IDs available at this endpoint |
| `base_url` | Base URL of the OpenAI-compatible API |
| `api_key` | API key for authentication |

Once configured, run `/models` inside the TUI to select your active model.


## Skills

Skills give the agent domain-specific context and significantly improve response quality for specialized tasks.

**To add a skill:**

1. Create a subdirectory under `~/.lightcode/skills/`
2. Add a `SKILL.md` file inside it describing the context or instructions

```
~/.lightcode/skills/
├── golang/
│   └── SKILL.md
└── docker/
    └── SKILL.md
```

You can also point `skills_path` in `config.json` to any other directory on your system.


