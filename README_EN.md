# OpenClaw Model Configurator

> Get your first model running — the easy way

[中文](README.md)

### Why this exists

Configuring models with custom providers in OpenClaw gets complicated fast — easy to mess up, and there are bugs. The built-in dashboard is so cluttered that even I struggle to navigate it, let alone newcomers. Many people connect through API proxies for better pricing, and if you're not particularly tech-savvy, following CLI instructions step-by-step can already be a challenge — editing config files by hand is a whole other level.

A GUI just makes it easier. So I built this project to help you get through that last step before you can actually chat.

Of course, to truly master OpenClaw you'll need to learn the advanced configs one step at a time. But if you spend hours and can't even get a conversation going, that's going to kill anyone's motivation. Once you get chatting, the enthusiasm to explore further comes naturally. There's a lot more waiting for you down the road — chat app integrations via Channels, power-user features, and more. Don't let the first step trip you up. Lean on AI assistants or the community to keep learning.

Have fun.

---

## Quick Start (GUI)

This is a **download-and-run** tool. No Python, no Node.js, no `pip install`, no `npm install`.

### Step 1: Download

Go to [Releases](../../releases) and grab the file for your system:

| OS | File |
|----|------|
| Windows | `openclaw-configurator-windows-amd64.exe` |
| macOS (Intel) | `openclaw-configurator-darwin-amd64` |
| macOS (Apple Silicon) | `openclaw-configurator-darwin-arm64` |
| Linux | `openclaw-configurator-linux-amd64` |

### Step 2: Run

**Windows:**

Double-click the downloaded `.exe` file. Your browser opens automatically.

> If Windows shows "Windows protected your PC", click "More info" → "Run anyway".

**macOS:**

macOS won't let you double-click unsigned binaries. You need to do this once:

1. Open Terminal (search "Terminal" in Spotlight)
2. Run these two commands:

```bash
chmod +x ~/Downloads/openclaw-configurator-darwin-*
~/Downloads/openclaw-configurator-darwin-arm64    # Apple Silicon (M1/M2/M3/M4)
# or
~/Downloads/openclaw-configurator-darwin-amd64    # Intel Mac
```

> If you see "cannot verify developer", go to **System Settings > Privacy & Security** and click "Open Anyway".
>
> After the first time, you only need the run command (no need to repeat `chmod`).

**Linux:**

Downloaded files usually don't have execute permission. Set it first:

- **File Manager**: Right-click → Properties → Permissions → check "Allow executing as program", then double-click
- **Terminal**:

```bash
chmod +x ~/Downloads/openclaw-configurator-linux-amd64
~/Downloads/openclaw-configurator-linux-amd64
```

### Step 3: Use the Interface

Your browser opens automatically at `http://127.0.0.1:19876/?token=...`. If it doesn't, copy the URL shown in the terminal.

The interface guides you through three steps:

**1. Connection** — Choose where OpenClaw is installed:
- **Local** — OpenClaw is on this machine
- **Remote Server** — SSH into another machine (password or key auth)
- **Docker** — OpenClaw is running inside a Docker container

**2. Config Path** — The tool auto-detects your `openclaw.json` location. You can also enter a custom path.

**3. Models** — This is the core:
- Add/edit/delete **providers** (API URL + key)
- Add/edit/delete **models** (model ID, context window, max tokens)
- Set the **primary model** (which model OpenClaw uses by default)
- **Test Connection** — verify the API URL and key actually work
- **Import/Export** — back up or restore the entire config file
- Click "View Source" to directly edit the raw JSON (secrets are auto-masked)
- Hit **Save** to write changes to disk

After saving, restart OpenClaw and your new models are ready to chat.

### What about headless servers?

This tool requires a browser. If your OpenClaw runs on a server without a desktop, you have two options:

**Option A: Run the configurator on your local machine and connect remotely (recommended)**

The tool has built-in SSH support. Run the configurator on your own computer, pick "Remote Server" in step 1, enter your SSH credentials, and you're done — nothing to install on the server.

**Option B: Run the configurator on the server, use SSH tunneling to access it locally**

If you need to run the configurator on the server itself (e.g., to configure OpenClaw inside a Docker container on that machine):

```bash
# ① On the server: download and run (Linux x86_64 example)
wget https://github.com/teecert/openclaw-configurator/releases/latest/download/openclaw-configurator-linux-amd64
chmod +x openclaw-configurator-linux-amd64
./openclaw-configurator-linux-amd64 --no-browser
```

It will print a URL with a token, like:

```
http://127.0.0.1:19876/?token=abc123...
```

Keep that terminal open. **In a new terminal on your local machine**, set up an SSH tunnel:

```bash
# ② On your local machine (replace user@your-server with your SSH info)
ssh -L 19876:127.0.0.1:19876 user@your-server -N
```

Now open that URL in your local browser — it just works.

> Windows users: use PuTTY's Tunnels feature. Set Source port to `19876`, Destination to `127.0.0.1:19876`.

### Launch Options (optional)

In most cases, just run the binary directly — no flags needed. These are only for special situations:

| Flag | Description | Default |
|------|-------------|---------|
| `--port` | Port to listen on | `19876` |
| `--bind` | Address to bind to | `127.0.0.1` (local only) |
| `--no-browser` | Don't auto-open browser | auto-open |
| `--version` | Show version | — |

```bash
# Port already in use? Pick another
./openclaw-configurator --port 8080
```

---

## Build from Source

If you prefer to compile yourself instead of downloading a prebuilt binary, you'll need [Go](https://go.dev/dl/) 1.22+.

```bash
# 1. Clone the repo
git clone https://github.com/teecert/openclaw-configurator.git
cd openclaw-configurator

# 2. Build
go build -o openclaw-configurator .

# 3. Run
./openclaw-configurator
```

On Windows:
```powershell
go build -o openclaw-configurator.exe .
.\openclaw-configurator.exe
```

Cross-compile all platforms (requires Make):
```bash
make build-all
ls dist/
# dist/openclaw-configurator-linux-amd64
# dist/openclaw-configurator-darwin-arm64
# dist/openclaw-configurator-darwin-amd64
# dist/openclaw-configurator-windows-amd64.exe
```

---

## How It Works

When you save, the tool updates three sections in `openclaw.json`:

- `models.providers` — Provider definitions (baseUrl, apiKey, api type, models)
- `agents.defaults.models` — Registry of available model references
- `agents.defaults.model.primary` — Primary model selection

It also removes `agents/*/agent/models.json` files. OpenClaw regenerates them automatically on next startup.

## Security

- Binds to `127.0.0.1` only — not accessible from other machines
- Every session gets a random 256-bit token; all API calls require it
- SSH credentials stay in memory only — never written to disk
- Raw config view masks all sensitive fields (API Keys, Tokens, Secrets, Passwords, etc.)
- Config is backed up to `.bak` before every write
- Written files use `0600` permissions (owner-only read/write)
- Token comparison uses constant-time algorithm to prevent timing attacks
- API requests are serialized to prevent race conditions

## Manual Configuration Guide (Without This Tool)

If you prefer editing files by hand, here's the complete walkthrough.

### Config File Location

| OS | Default Path |
|----|-------------|
| Linux / macOS | `~/.openclaw/openclaw.json` |
| Windows | `%USERPROFILE%\.openclaw\openclaw.json` or `%LOCALAPPDATA%\openclaw\openclaw.json` |

Can also be set via environment variables: `OPENCLAW_CONFIG_PATH`, `OPENCLAW_HOME`, `OPENCLAW_STATE_DIR`

### Structure Overview

The model-related sections in `openclaw.json`:

```json
{
  "models": {
    "mode": "merge",
    "providers": { ... }
  },
  "agents": {
    "defaults": {
      "model": { "primary": "provider-name/model-id" },
      "models": { "provider-name/model-id": {}, ... }
    }
  }
}
```

### Step 1: Add a Provider

Under `models.providers`, add a new key. The name **must** start with `custom-`:

```json
{
  "models": {
    "mode": "merge",
    "providers": {
      "custom-my-api": {
        "baseUrl": "https://api.example.com/v1",
        "apiKey": "sk-your-api-key-here",
        "api": "openai-completions",
        "models": []
      }
    }
  }
}
```

**API type options:**
| Value | Use When |
|-------|----------|
| `openai-completions` | OpenAI-compatible APIs (most third-party providers) |
| `anthropic-messages` | Anthropic-compatible APIs |
| `openai-codex-responses` | OpenAI Codex (official ChatGPT backend) |

### Step 2: Add Models

Inside the provider's `models` array:

```json
"models": [
  {
    "id": "gpt-4o",
    "name": "gpt-4o (Custom Provider)",
    "api": "openai-completions",
    "reasoning": true,
    "input": ["text"],
    "cost": { "input": 0, "output": 0, "cacheRead": 0, "cacheWrite": 0 },
    "contextWindow": 200000,
    "maxTokens": 16384
  }
]
```

**Field reference:**

| Field | Required | Description |
|-------|----------|-------------|
| `id` | Yes | Model identifier sent to the API |
| `name` | Yes | Display name, convention: `{id} (Custom Provider)` |
| `api` | No | Defaults to provider's api type |
| `reasoning` | No | `true` if model supports chain-of-thought |
| `input` | No | Defaults to `["text"]` |
| `cost` | No | Set all to `0` for custom providers |
| `contextWindow` | Yes | Max context in tokens (recommended: `128000`~`200000`) |
| `maxTokens` | Yes | Max output tokens (recommended: `8192`~`16384`) |

### Step 3: Register Models in Agent Defaults

Add a reference for each model in `agents.defaults.models`:

```json
"agents": {
  "defaults": {
    "model": {
      "primary": "custom-my-api/gpt-4o"
    },
    "models": {
      "custom-my-api/gpt-4o": {}
    }
  }
}
```

Format: `"provider-name/model-id": {}`.

### Step 4: Set the Primary Model

Update `agents.defaults.model.primary`:

```json
"model": {
  "primary": "custom-my-api/gpt-4o"
}
```

### Step 5: Sync Agent Configs

OpenClaw generates per-agent files at:
```
~/.openclaw/agents/{agent-id}/agent/models.json
```

After editing `openclaw.json`, **delete these files** so OpenClaw regenerates them:

```bash
# Linux / macOS
rm -f ~/.openclaw/agents/*/agent/models.json

# Windows (PowerShell)
Remove-Item -Path "$env:USERPROFILE\.openclaw\agents\*\agent\models.json" -Force -ErrorAction SilentlyContinue
```

OpenClaw recreates them automatically on next startup.

### Complete Example

A minimal config with one provider and two models:

```json
{
  "models": {
    "mode": "merge",
    "providers": {
      "custom-my-api": {
        "baseUrl": "https://api.example.com/v1",
        "apiKey": "sk-your-key",
        "api": "openai-completions",
        "models": [
          {
            "id": "gpt-4o",
            "name": "gpt-4o (Custom Provider)",
            "contextWindow": 200000,
            "maxTokens": 16384
          },
          {
            "id": "claude-sonnet-4",
            "name": "claude-sonnet-4 (Custom Provider)",
            "contextWindow": 200000,
            "maxTokens": 16384
          }
        ]
      }
    }
  },
  "agents": {
    "defaults": {
      "model": { "primary": "custom-my-api/gpt-4o" },
      "models": {
        "custom-my-api/gpt-4o": {},
        "custom-my-api/claude-sonnet-4": {}
      }
    }
  }
}
```

### Understanding `models.mode`

- **`merge`** (default) — Keeps built-in providers and adds your custom ones on top
- **`replace`** — Only uses providers you explicitly define; removes all built-in ones

Most users should keep `merge`.

### Troubleshooting

**Models not showing up after editing?**
Delete agent-level `models.json` and restart OpenClaw:
```bash
rm -f ~/.openclaw/agents/*/agent/models.json
```

**API errors when using a model?**
- Verify `baseUrl` ends correctly (some APIs need `/v1`, others don't)
- Check `apiKey` is correct
- Ensure `api` type matches the provider's actual protocol

## License

MIT
