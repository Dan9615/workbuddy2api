# WorkBuddy to OpenAI API Gateway

> Turn your **Tencent WorkBuddy / CodeBuddy** account into a standard **OpenAI-compatible API** (`/v1/chat/completions`) with a custom **Base URL** and **API Key**.  
> Enjoy **1,000,000 Tokens (1M) Context Window** and **Free DeepSeek V4.1 Flash (`x0.00` credits)** across any AI client (NextChat, Chatbox, LobeChat, Cursor, Roo Code, Python/Node.js SDK, OpenClaw, etc.).

---

## 🌟 Features

- **Standard OpenAI Format**: Exposes standard `/v1/chat/completions` and `/v1/models` endpoints.
- **Any Client Compatible**: Drop-in replacement for OpenAI endpoints in Chatbox, NextChat, LobeChat, Cursor, OpenWebUI, LangChain, etc.
- **🧠 1,000,000 Tokens (1M) Context**: High-capacity context window for large codebases, research papers, and deep reasoning.
- **🛠️ Native Function Calling (Tool Calls)**: Full support for standard OpenAI `tool_calls`.
- **💸 0 Credit Cost (Free Tier)**: Route requests through WorkBuddy Global (`workbuddy.ai`) where DeepSeek V4.1 Flash is 100% free (`credits: x0.00`).
- **🔄 Multi-Account Connection Pool**: Automatic round-robin, health checks, rate-limit cooldowns, and session stickiness.
- **⚡️ Built-in Token Keepalive**: Automatically refreshes credentials in the background before expiry.

---

## 🚀 3-Minute Quick Setup

### Step 1: Clone and Configure

Clone this repository to your server or local machine:

```bash
git clone https://github.com/Dan9615/workbuddy-openclaw.git
cd workbuddy-openclaw
```

Copy the example configuration:
```bash
cp config.example.json config.json
```

Open `config.json` and set your preferred **API Key** (e.g., `sk-my-secret-key`):

```json
{
  "listen": ":7863",
  "api_key": "sk-my-secret-key",
  "auth_dir": "./auths",
  "state_file": "./data/state.json",
  "global": {
    "enabled": true
  },
  "pool": {
    "max_in_flight": 5
  }
}
```

---

### Step 2: Start the Gateway (Docker)

Launch the gateway container using Docker Compose:

```bash
docker compose up -d --build
```

The gateway will now be listening on port `7863`.

---

### Step 3: Login to WorkBuddy (OAuth)

Generate a one-time Google OAuth authorization link:

```bash
docker compose exec -it wb2api ./login --realm=global url
```

1. **Open the returned URL** in your browser.
2. Sign in with your **Google Account** on WorkBuddy.
3. Poll and save the authorization token into the container:
   ```bash
   docker compose exec -it wb2api ./login --realm=global poll
   ```

> 💡 **Auto Hot-Reload**: The gateway automatically scans `./auths/` every 5 seconds. Once the token is saved, your account is immediately active in the connection pool—no container restart required!

---

## 🔑 Your API Credentials

You now have a fully functional OpenAI API endpoint:

| Parameter | Value |
|---|---|
| **Base URL** | `http://<YOUR_SERVER_IP>:7863/v1` (or `http://localhost:7863/v1`) |
| **API Key** | The `api_key` you set in `config.json` (e.g., `sk-my-secret-key`) |
| **Model Name** | `global:deepseek-v4.1-flash` (or `global:default-model`) |

---

## 💻 How to Use in Popular Clients

### 1. `curl` (Terminal Test)

```bash
curl -X POST http://localhost:7863/v1/chat/completions \
  -H "Authorization: Bearer sk-my-secret-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "global:deepseek-v4.1-flash",
    "messages": [
      {"role": "user", "content": "Hello! Reply in 5 words."}
    ]
  }'
```

---

### 2. Python (OpenAI Official SDK)

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:7863/v1",
    api_key="sk-my-secret-key"
)

response = client.chat.completions.create(
    model="global:deepseek-v4.1-flash",
    messages=[
        {"role": "user", "content": "Explain quantum computing in one sentence."}
    ]
)

print(response.choices[0].message.content)
```

---

### 3. Chatbox / NextChat / LobeChat / OpenWebUI

In your client's settings:
1. **Model Provider**: `OpenAI` (or `Custom OpenAI`)
2. **API Host / Base URL**: `http://<YOUR_IP>:7863/v1`
3. **API Key**: `sk-my-secret-key`
4. **Custom Model**: Add `global:deepseek-v4.1-flash`

---

### 4. Cursor / Roo Code / Cline (VS Code)

1. Open Extension / AI Settings.
2. Choose **OpenAI Compatible**.
3. Set **Base URL** to `http://localhost:7863/v1`.
4. Set **API Key** to `sk-my-secret-key`.
5. Model ID: `global:deepseek-v4.1-flash`.

---

### 5. OpenClaw (Optional)

Add to `~/.openclaw/openclaw.json` under `models.providers`:

```json
{
  "cliproxy": {
    "baseUrl": "http://127.0.0.1:8317/v1",
    "apiKey": "sk-my-secret-key",
    "models": [
      {
        "id": "deepseek-v4.1-flash",
        "name": "DeepSeek V4.1 Flash (WorkBuddy)",
        "api": "openai-completions",
        "reasoning": true,
        "contextWindow": 1000000,
        "input": ["text", "image"]
      }
    ]
  }
}
```

---

## 📋 Available Models

Query available models anytime via:

```bash
curl -H "Authorization: Bearer sk-my-secret-key" http://localhost:7863/v1/models
```

Popular models in the pool:
- `global:deepseek-v4.1-flash` (1M Context, **0 Credit Free**)
- `global:default-model` (Auto fast coding model)
- `global:primary-model` (High quality reasoning)
- `global:glm-5.3` / `global:kimi-k3`

---

## 📄 License

Distributed under the [MIT License](LICENSE).
