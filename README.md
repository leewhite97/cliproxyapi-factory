<div align="center">

# CLIProxyAPI Factory Provider

<p><strong>A client-neutral Factory provider plugin for CLI Proxy API (CPA).</strong></p>

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![CLIProxyAPI](https://img.shields.io/badge/CLIProxyAPI-v8.0.4-blue?style=flat-square)](https://github.com/router-for-me/CLIProxyAPI)
[![Management Panel](https://img.shields.io/badge/Panel-v1.26.1-purple?style=flat-square)](https://github.com/router-for-me/Cli-Proxy-API-Management-Center)
[![Tests](https://img.shields.io/badge/Tests-Passing-brightgreen?style=flat-square)]()
[![License](https://img.shields.io/badge/License-MIT-yellow?style=flat-square)](./LICENSE)

[English](./README.md) | [中文说明](#中文指引)

</div>

---

## Features

- **Multi-Protocol Translation**: Translates requests and responses across **OpenAI** (`/v1/chat/completions`), **Anthropic Claude** (`/v1/messages`), and **OpenAI Responses** (`/v1/responses`).
- **Streaming Support**: Supports server-sent events (SSE) streaming with usage reporting across all endpoints.
- **Embedded Web Console**: Integrated web interface in the official CPA management panel for quota tracking, credential management, and compatibility options.
- **Client Neutral**: Generic provider adapter that operates without assuming specific client tools or frameworks.

---

## Quick Start

### 1. Build from Source

Requirements: Go 1.26+, a compatible C compiler (CGO enabled), and `make`.

```bash
git clone https://github.com/leewhite97/cliproxyapi-factory.git
cd cliproxyapi-factory

# Run tests and compile the C-shared library
make verify
```

The output library is created at `build/factory.so` (or `.dylib` / `.dll` depending on the target OS).

To build in a clean container without a local Go toolchain:
```bash
sh go-builder.sh make verify
```

---

### 2. Configure CLIProxyAPI

Place the compiled library into your CPA `plugins/` directory and enable it in `config.yaml`:

```yaml
plugins:
  enabled: true
  dir: "./plugins"
  configs:
    factory:
      enabled: true
      catalog_cache_path: "./logs/factory-cache/models.json"
      compatibility:
        mode: "factory"              # "factory" (default) or "passthrough"
        sensitive_words:
          - "Hermes"
          - "Nous"
          - "pi"
        obfuscate_user_text: false   # false = instructions only; true = include user text
```

Restart CLIProxyAPI to load the plugin.

To save settings from the web console, CPA must be able to write its configuration
file. For Docker, mount the configuration file read-write; the plugin library
can stay read-only. A read-only configuration remains valid for inference, but
Management API saves will fail. The deployment template must not overwrite the
runtime configuration on subsequent starts.

`passthrough` disables Factory identity adaptation and cannot enable word
filtering; the console clears its word list and user-text flag when saving this
mode. Upstream acceptance is not guaranteed. Use `factory` with an empty word
list to disable filtering while keeping Factory adaptation.

---

### 3. Manage Credentials & Quotas

1. Open your CPA Management Panel.
2. Navigate to the **Factory Provider** item in the sidebar.
3. In the **凭证管理 (Credentials)** tab, enter your Factory API key and label.
4. View rolling quotas and usage metrics in the **额度监控 (Quotas)** tab.

---

## Configuration Reference

| Option | Type | Default | Description |
|---|:---:|:---:|---|
| `enabled` | `bool` | `true` | Enable or disable the plugin. |
| `catalog_cache_path` | `string` | `""` | Path to save model discovery snapshot. |
| `compatibility.mode` | `string` | `"factory"` | Factory adaptation, or `"passthrough"` without identity adaptation. |
| `compatibility.sensitive_words` | `string[]` | `[]` | Optional word list. An empty list disables extra word filtering. |
| `compatibility.obfuscate_user_text`| `bool` | `false` | Apply compatibility filtering to user messages in addition to system instructions. |

---

## Testing

```bash
# Run unit tests with race detection
go test -race ./... -count=1

# Verify panel session integration
node scripts/test_panel_auth.cjs
node scripts/test_panel_config.cjs

# Verify build output
make verify
```

---

<div id="中文指引">

## 中文指引

### 项目简介
`cliproxyapi-factory` 是为 **CLIProxyAPI (CPA)** 开发的 Factory 提供商插件，实现通用协议转换、流式传输、额度展示与凭证管理。

### 核心功能
1. **多协议兼容**：支持 OpenAI、Anthropic Claude 及 OpenAI Responses 协议互相转换与流式响应。
2. **控制台集成**：内嵌在 CPA 官方面板中，提供额度查看、凭证添加/启停与兼容配置。
3. **客户端中立**：保持纯净通用的提供商适配，不针对任何特定客户端工具绑定私有逻辑。

### 编译与安装
1. 编译产物：执行 `make verify` 生成 `build/factory.so`。
2. 配置生效：将产物复制到 CPA 的 `plugins/` 目录并在 `config.yaml` 中启用。
3. 管理使用：在 CPA 官方面板中通过 Factory Provider 菜单直接管理凭证与查看额度。

</div>

---

## License

MIT License. See [LICENSE](./LICENSE). Not affiliated with Factory AI or Router-For.ME.
