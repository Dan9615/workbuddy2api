# WorkBuddy OpenClaw Bridge (WorkBuddy 接入 OpenClaw 完整套件)

一鍵將騰訊 **CodeBuddy / WorkBuddy** 的免費用量（特別是 **DeepSeek V4.1 Flash (1M 上下文)**）轉化為 OpenAI 兼容協議，並無縫串接至 **OpenClaw** 作為日常模型與 Sub-agent 執行核心。

---

## 🌟 核心亮點

- **🧠 1,000,000 Tokens (1M) 超大上下文**：單次對話或工程分析可輕鬆消化數十個源碼檔案。
- **🛠️ 原生 Tool Calling 支持**：完美適配 OpenClaw 執行 Shell 指令、檔案讀寫、修復代碼等 Agentic 任務。
- **💸 0 Credit 免費用量**：對接 WorkBuddy 國際版端點，享受免費超長思考鏈（Reasoning）模型。
- **🚀 零侵入架構**：以 Docker 容器自動代理與管理 Token 保活，配合 CLIProxyAPI 轉發，無需修改宿主既有配置。

---

## 🏗️ 架構全景圖

```text
OpenClaw Gateway
       │ (OpenAI-compatible /v1)
       ▼
CLIProxyAPI (Port :8317)
       │ (Reverse Proxy & Model Router)
       ▼
workbuddy2api (Docker Container, Port :7863)
       │ (Multi-account Pool & Token Keepalive)
       ▼
Tencent WorkBuddy Global API
       │
       ▼
DeepSeek V4.1 Flash (1M Context & Native Tool Calling)
```

---

## 🚀 快速上手

### 1. 啟動 workbuddy2api 網關

創建 `docker-compose.yml`：

```yaml
services:
  wb2api:
    image: ghcr.io/ithtelab/workbuddy-manager:latest
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
```

啟動容器：
```bash
docker compose up -d
```

### 2. 登入 WorkBuddy 帳號 (OAuth)

在瀏覽器打開以下授權端點，登入 Google 帳號授權：
```bash
https://www.workbuddy.ai/login?platform=CLI&state=<STATE_UUID>
```
將取得到的 Access Token 與 Refresh Token 儲存至 `./auths/workbuddy-<uid>.json`，上游網關將在 5 秒內熱加載該帳號。

### 3. 配置 OpenClaw 模型

在 `~/.openclaw/openclaw.json` 中的 `models.providers.cliproxy.models` 陣列加入：

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

並在 `agents.defaults.models` 加入 alias：
```json
"cliproxy/deepseek-v4.1-flash": {
  "alias": "ds-v4.1"
}
```

重啟 OpenClaw 生效：
```bash
openclaw gateway restart
```

---

## 👥 將 Sub-agents 編制換軌為 DeepSeek V4.1 Flash

為了發揮 1M 超大上下文優勢，建議將常用 Sub-agent（代碼修復者、文檔員、探索員）全數切換至此模型：

```json
"agents": {
  "entries": {
    "explorer": { "model": "cliproxy/deepseek-v4.1-flash" },
    "librarian": { "model": "cliproxy/deepseek-v4.1-flash" },
    "fixer": { "model": "cliproxy/deepseek-v4.1-flash" }
  }
}
```

---

## 📜 許可證
MIT License
