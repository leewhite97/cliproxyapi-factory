# Architecture Review & Design Records

## 1. Scope & Objective

`cliproxyapi-factory` is a C-shared provider plugin for **CLIProxyAPI (CPA)**. It integrates Factory upstream as a standard inference provider:
- Client-neutral design with standard model mapping;
- Protocol parity across OpenAI, Claude, and OpenAI Responses formats;
- Built-in management interface for quota viewing, credential configuration, and compatibility options.

---

## 2. Technical Decisions

### 2.1 Client Neutrality & Preamble Handling
- Removed client-specific string matching and overrides.
- Configurable compatibility mode handles sensitive word filtering where necessary.

### 2.2 Model Capabilities Registry (`ModelSpec`)
- Centralized model parameters and thinking effort support in a shared specification.
- Dynamic model discovery falls back to conservative definitions without inventing unsupported capabilities.

### 2.3 Streaming & Request Lifecycle
- Upstream `response.incomplete` events are treated as terminal completion signals with proper finish reasons and token usage metrics.
- Downstream disconnections are categorized cleanly as request cancellations.

### 2.4 Management Web Interface
- Embedded resources (`/v0/resource/plugins/factory/quota`) provide web-based quota monitoring, credential management, and compatibility configuration.
- Follows standard same-origin session handling without persisting credentials in client storage.

---

## 3. Verification

- Full Go test suite passes with race detection (`go test -race ./...`).
- Session authentication adapter verified with Node test suite (`node scripts/test_panel_auth.cjs`).
- Multi-protocol translation verified with native streaming and token usage reporting.
