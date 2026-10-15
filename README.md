# WorkBuddy OpenClaw Bridge

> Seamlessly bridge Tencent **WorkBuddy / CodeBuddy** free tier quotas—especially **DeepSeek V4.1 Flash (1M Context Window)**—into standard OpenAI-compatible endpoints, and integrate directly with **OpenClaw** as your primary engine and sub-agent delegation backbone.

---

## 🌟 Key Highlights

- **🧠 1,000,000 Tokens (1M) Context Window**: Effortlessly ingest dozens of codebase files, extensive logs, or complete RFC specs in a single prompt.
- **🛠️ Native Tool Calling (Function Calling)**: 100% compliant with standard OpenAI `tool_calls` schemas. Fully capable of executing terminal commands, file edits, git workflows, and autonomous sub-agent routines.
- **💸 0 Credit Cost (Free Tier)**: Connects to WorkBuddy Global (`workbuddy.ai`) endpoints where DeepSeek V4.1 Flash is marked as `credits: x0.00` with zero billing consumption.
- **⚡️ Built-in Reasoning Chain**: Returns structured `reasoning_content` with deep multi-step thinking traces.
- **🚀 Non-Invasive Architecture**: Runs as lightweight Docker containers with multi-account connection pooling, token keepalives, and automatic hot-reloading.

---

## 🏗️ Architecture

```text
┌────────────────────────────────────────────────────────┐
│                   OpenClaw Gateway                     │
│  (Orchestrator, Hacker Agent, Sub-Agents, Workflows)   │
└──────────────────────────┬─────────────────────────────┘
                           │ (OpenAI-compatible /v1)
                           ▼
┌────────────────────────────────────────────────────────┐
│             CLIProxyAPI Router (Port :8317)            │
│         (Model Mapping & Protocol Normalization)       │
└──────────────────────────┬─────────────────────────────┘
                           │ (Forward to upstream gateway)
                           ▼
┌────────────────────────────────────────────────────────┐
│          workbuddy2api Docker (Port :7863)             │
│   (Multi-Account Pool, Token Keepalive & Hot-Reload)   │
└──────────────────────────┬─────────────────────────────┘
                           │ (Reverse-engineered HTTPS API)
                           ▼
┌────────────────────────────────────────────────────────┐
│              Tencent WorkBuddy Global API              │
│       https://www.workbuddy.ai (credits: x0.00)        │
└──────────────────────────┬─────────────────────────────┘
                           │
                           ▼
         DeepSeek V4.1 Flash (1M Context + Tools)
```

---

## 🚀 Quick Start

### 1. Launch Upstream Gateway (`workbuddy2api`)

Create a `docker-compose.yml` file:

```yaml
services:
  wb2api:
    image: workbuddy2api-wb2api
    container_name: workbuddy2api
    restart: unless-stopped
    ports:
      - "7863:7863"
    environment:
      - TZ=Asia/Hong_Kong
    volumes:
      - ./auths:/app/auths
      - ./data:/app/data
      - ./config.json:/app/config.json:ro

  # Optional Web UI Manager & Dashboard
  workbuddy-manager:
    image: ghcr.io/ithtelab/workbuddy-manager:latest
    container_name: workbuddy-manager
    restart: unless-stopped
    ports:
      - "127.0.0.1:7864:7864"
    environment:
      TZ: Asia/Hong_Kong
      WB2API_BASE: http://wb2api:7863
      WB_TRUST_PROXY: "1"
    volumes:
      - ./manager-data:/app/data
      - .:/opt/workbuddy2api
    depends_on:
      - wb2api
```

Start the service:
```bash
docker compose up -d
```

---

### 2. Authenticate WorkBuddy Account (OAuth)

1. Generate an authorization URL:
   ```bash
   curl -s -X POST "https://www.workbuddy.ai/v2/plugin/auth/state?platform=CLI"
   ```
2. Open the returned `authUrl` in your browser and complete Google OAuth login:
   ```text
   https://www.workbuddy.ai/login?platform=CLI&state=<STATE_UUID>
   ```
3. Poll for the access token:
   ```bash
   curl -s "https://www.workbuddy.ai/v2/plugin/auth/token?state=<STATE_UUID>"
   ```
4. Save the resulting credentials into `./auths/workbuddy-<uid>.json`. The upstream pool will hot-reload the account within 5 seconds without restarting the container.

---

### 3. Configure OpenClaw Model

Add the model entry to `~/.openclaw/openclaw.json` under `models.providers.cliproxy.models`:

```json
{
  "id": "deepseek-v4.1-flash",
  "name": "DeepSeek V4.1 Flash (WorkBuddy)",
  "api": "openai-completions",
  "reasoning": true,
  "contextWindow": 1000000,
  "contextTokens": 1000000,
  "maxTokens": 128000,
  "compat": {
    "supportsReasoningEffort": true
  },
  "input": ["text", "image"]
}
```

Add an alias under `agents.defaults.models`:
```json
"cliproxy/deepseek-v4.1-flash": {
  "alias": "ds-v4.1"
}
```

Restart OpenClaw Gateway to reload the catalog:
```bash
openclaw gateway restart
```

---

### 4. Delegate Heavy Sub-Agent Tasks to DeepSeek V4.1 Flash

Leverage the 1M token context window and zero-cost quota by re-routing automated sub-agents (`fixer`, `librarian`, `explorer`) in `~/.openclaw/openclaw.json`:

```json
{
  "agents": {
    "entries": {
      "explorer": {
        "model": "cliproxy/deepseek-v4.1-flash"
      },
      "librarian": {
        "model": "cliproxy/deepseek-v4.1-flash"
      },
      "fixer": {
        "model": "cliproxy/deepseek-v4.1-flash"
      }
    }
  }
}
```

| Sub-Agent | Role | Upgraded Capability |
|---|---|---|
| **`explorer` 🧭** | Codebase navigation & pattern discovery | Ingests entire repositories and AST maps without token anxiety. |
| **`librarian` 📚** | Documentation & ecosystem research | Reads voluminous API docs, GitHub issues, and RFCs in a single prompt. |
| **`fixer` 🔨** | Implementation & bug fixing | Leverages native Tool Calling for terminal execution, test verification, and automated patching. |

---

## 🧪 Verification

Test connectivity with a simple `curl` request:

```bash
curl -s -X POST http://127.0.0.1:8317/v1/chat/completions \
  -H "Authorization: Bearer <YOUR_API_KEY>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-v4.1-flash",
    "messages": [
      {"role": "user", "content": "Hello! Confirm 1M context & tool calling."}
    ]
  }'
```

In Telegram / OpenClaw chat:
```text
/model ds-v4.1
/status
```

---

## 📄 License

Distributed under the [MIT License](LICENSE).
