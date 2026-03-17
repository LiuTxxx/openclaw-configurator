# OpenClaw Model Configurator

> OpenClaw 模型配置器 — 帮你跑通第一步

[English](README_EN.md)

### 为什么做这个

OpenClaw 在使用自定义服务商时，模型配置的过程会变得复杂很多，容易出错，还有不少 bug。它自带的 Dashboard 配置页乱得我自己都不太会操作，相信很多新手朋友也有同感。不少人为了实惠，通过中转站接的 API，加上本身不怎么懂计算机，照着 CLI 的步骤都搞不来，更别说去手动改配置文件了。

还得是有 GUI 好用。所以写了这个项目，帮各位朋友配好「能聊天」前的最后一步。

诚然，要真正玩好 OpenClaw，高级配置还是得一步步学。但搞半天聊都聊不起来，肯定也挫伤了不少新手朋友的入坑意愿。只要先聊起来，相信大伙也能更有兴致继续探索下去。往后还有很多更深入的东西等着你 — 通过 Channel 接入聊天软件、更多元的高自由度功能等等。希望大家不要因为绊倒在第一步就失去信心，多通过 AI 询问或社区互助来学习和了解这款开源工具。

祝大家玩得开心。

---

## 快速上手（GUI 图形界面）

这是一个**下载即用**的跨平台工具，无需安装 Python、Node.js 或任何依赖，下载一个文件就能运行。

### 第一步：下载

前往 [Releases](../../releases) 页面，根据你的系统下载对应文件：


| 系统                    | 文件名                                       |
| --------------------- | ----------------------------------------- |
| Windows               | `openclaw-configurator-windows-amd64.exe` |
| macOS (Intel)         | `openclaw-configurator-darwin-amd64`      |
| macOS (Apple Silicon) | `openclaw-configurator-darwin-arm64`      |
| Linux                 | `openclaw-configurator-linux-amd64`       |


### 第二步：运行

**Windows：**

双击下载的 `.exe` 文件即可。浏览器会自动打开配置页面。

> 如果 Windows 弹出"Windows 已保护你的电脑"提示，点击"更多信息" → "仍要运行"。

**macOS：**

macOS 不允许直接双击运行未签名的程序，需要执行一次以下操作：

1. 打开"终端"（在启动台搜索"终端"或"Terminal"）
2. 输入以下两行命令：

```bash
chmod +x ~/Downloads/openclaw-configurator-darwin-*
~/Downloads/openclaw-configurator-darwin-arm64    # Apple Silicon (M1/M2/M3/M4)
# 或
~/Downloads/openclaw-configurator-darwin-amd64    # Intel Mac
```

> 如遇到"无法验证开发者"提示，前往 **系统设置 > 隐私与安全性**，点击"仍要打开"。
>
> 之后每次运行只需要再执行一次运行命令（不用重复 `chmod`）。

**Linux：**

大多数桌面 Linux 下载的文件没有执行权限，需要先设置一下：

- **方法一（文件管理器）**：右键文件 → 属性 → 权限 → 勾选"允许作为程序执行"，然后双击运行
- **方法二（终端）**：

```bash
chmod +x ~/Downloads/openclaw-configurator-linux-amd64
~/Downloads/openclaw-configurator-linux-amd64
```

### 第三步：使用界面

运行后浏览器会自动打开配置页面（地址类似 `http://127.0.0.1:19876/?token=...`）。如果没有自动打开，手动复制终端里显示的地址到浏览器即可。

界面分三步引导你完成配置：

**1. 连接方式** — 选择 OpenClaw 安装在哪里：

- **本地** — OpenClaw 装在当前电脑上
- **远程服务器** — 通过 SSH 连接到另一台机器（支持密码和密钥认证）
- **Docker** — OpenClaw 运行在 Docker 容器中

**2. 配置文件路径** — 工具会自动检测 `openclaw.json` 的位置，也可以手动输入自定义路径

**3. 模型配置** — 这是核心：

- 添加/编辑/删除 **服务商**（填入 API 地址和密钥）
- 添加/编辑/删除 **模型**（设置模型 ID、上下文长度、最大输出等）
- 设置 **主模型**（OpenClaw 默认使用哪个模型对话）
- **测试连接** — 验证 API 地址和密钥是否可用
- **导入/导出** — 备份或恢复整个配置文件
- 点击"查看源文件"可直接编辑原始 JSON（敏感字段自动遮蔽）
- 最后点击 **保存** 将配置写入文件

保存后，重启 OpenClaw 即可使用新配置的模型开始对话。

### 没有桌面环境的服务器怎么办？

本工具需要浏览器操作，如果你的 OpenClaw 跑在没有桌面的远程服务器上，有两种办法：

**方法一：在本地电脑运行配置器，通过"远程服务器"模式连过去（推荐）**

工具自带 SSH 连接功能。在你自己的电脑上运行配置器，第一步选"远程服务器"，填入服务器的 SSH 信息即可，不需要在服务器上装任何东西。

**方法二：在服务器上运行配置器，用 SSH 隧道转发到本地浏览器**

如果你想在服务器上运行配置器（比如配置本地 Docker 里的 OpenClaw）：

```bash
# ① 在服务器上下载并运行（以 Linux x86_64 为例）
wget https://github.com/teecert/openclaw-configurator/releases/latest/download/openclaw-configurator-linux-amd64
chmod +x openclaw-configurator-linux-amd64
./openclaw-configurator-linux-amd64 --no-browser
```

运行后终端会显示一个带 token 的地址，类似：

```
http://127.0.0.1:19876/?token=abc123...
```

先别关这个终端。**另开一个终端窗口**，在你的本地电脑上建立 SSH 隧道：

```bash
# ② 在你的电脑上执行（把 user@your-server 换成你的 SSH 登录信息）
ssh -L 19876:127.0.0.1:19876 user@your-server -N
```

然后在本地浏览器打开上面那个地址就能用了。

> Windows 用户可以用 PuTTY 的 Tunnels 功能实现同样的效果：Source port 填 `19876`，Destination 填 `127.0.0.1:19876`。

### 启动参数（可选）

大多数情况下直接运行就行，不需要加任何参数。以下参数仅在有特殊需求时使用：


| 参数             | 说明       | 默认值                 |
| -------------- | -------- | ------------------- |
| `--port`       | 监听端口     | `19876`             |
| `--bind`       | 绑定地址     | `127.0.0.1`（仅本机可访问） |
| `--no-browser` | 不自动打开浏览器 | 自动打开                |
| `--version`    | 显示版本号    | —                   |


```bash
# 端口被占用时换一个
./openclaw-configurator --port 8080
```

---

```markdown
### 命令行选项（CLI）

```
  --port        监听端口（默认 19876）
  --bind        绑定地址（默认 127.0.0.1，仅本机可访问）
  --no-browser  不自动打开浏览器
  --version     显示版本号
```

示例：

```bash
# 使用自定义端口
./openclaw-configurator --port 8080

# 不自动打开浏览器（适合无桌面环境的服务器）
./openclaw-configurator --no-browser
```

```

## 从源码构建

如果你不想下载预编译的二进制，也可以从源码自行编译。需要安装 [Go](https://go.dev/dl/) 1.22 或更新版本。

```bash
# 1. 克隆源码
git clone https://github.com/teecert/openclaw-configurator.git
cd openclaw-configurator

# 2. 编译（在当前目录生成可执行文件）
go build -o openclaw-configurator .

# 3. 运行
./openclaw-configurator
```

Windows 下编译：

```powershell
go build -o openclaw-configurator.exe .
.\openclaw-configurator.exe
```

一次性编译所有平台（需要 Make）：

```bash
make build-all
ls dist/
# dist/openclaw-configurator-linux-amd64
# dist/openclaw-configurator-darwin-arm64
# dist/openclaw-configurator-darwin-amd64
# dist/openclaw-configurator-windows-amd64.exe
```

---

## 工作原理

当你点击保存时，工具会修改 `openclaw.json` 中的三个位置：

- `models.providers` — 服务商定义（API 地址、密钥、模型列表）
- `agents.defaults.models` — 可用模型注册表
- `agents.defaults.model.primary` — 主模型选择

同时会删除 `agents/*/agent/models.json` 文件，OpenClaw 会在下次启动时自动重新生成它们。

## 安全性

- 默认只绑定 `127.0.0.1`，外部机器无法访问
- 每次启动生成随机 256 位 token，所有 API 调用都需要携带
- SSH 密码/密钥仅存于内存，不写入磁盘
- 查看源文件时，所有敏感字段（API Key、Token、Secret、Password 等）自动遮蔽
- 写入前自动备份到 `.bak` 文件
- 写入文件使用 `0600` 权限（仅文件所有者可读写）
- Token 比较使用常量时间算法，防止计时攻击
- API 请求序列化处理，防止并发竞争

## 纯手动配置指南（不用本工具）

如果你更喜欢直接编辑文件，以下是手动配置的完整流程。

### 配置文件位置


| 系统            | 默认路径                                                                              |
| ------------- | --------------------------------------------------------------------------------- |
| Linux / macOS | `~/.openclaw/openclaw.json`                                                       |
| Windows       | `%USERPROFILE%\.openclaw\openclaw.json` 或 `%LOCALAPPDATA%\openclaw\openclaw.json` |


也可通过环境变量指定：`OPENCLAW_CONFIG_PATH`、`OPENCLAW_HOME`、`OPENCLAW_STATE_DIR`

### 结构概览

`openclaw.json` 中与模型相关的部分：

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

### 第 1 步：添加服务商

在 `models.providers` 下添加。名称**必须**以 `custom-` 开头：

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

**API 类型说明：**


| 值                        | 适用场景                        |
| ------------------------ | --------------------------- |
| `openai-completions`     | OpenAI 兼容 API（大多数第三方中转都用这个） |
| `anthropic-messages`     | Anthropic 兼容 API            |
| `openai-codex-responses` | OpenAI Codex（ChatGPT 官方后端）  |


### 第 2 步：添加模型

在服务商的 `models` 数组中添加：

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

**字段说明：**


| 字段              | 必填  | 说明                                    |
| --------------- | --- | ------------------------------------- |
| `id`            | 是   | 发送给 API 的模型标识符                        |
| `name`          | 是   | 显示名称，建议格式：`{id} (Custom Provider)`    |
| `api`           | 否   | 留空则继承服务商的 api 类型                      |
| `reasoning`     | 否   | 模型是否支持推理链（chain-of-thought）           |
| `input`         | 否   | 默认 `["text"]`                         |
| `cost`          | 否   | 自定义服务商全填 `0` 即可                       |
| `contextWindow` | 是   | 最大上下文长度（token 数），推荐 `128000`~`200000` |
| `maxTokens`     | 是   | 最大输出 token 数，推荐 `8192`~`16384`        |


### 第 3 步：注册模型到 Agent 默认配置

在 `agents.defaults.models` 中为每个模型添加引用：

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

格式为 `"服务商名/模型ID": {}`。

### 第 4 步：设置主模型

更新 `agents.defaults.model.primary`：

```json
"model": {
  "primary": "custom-my-api/gpt-4o"
}
```

### 第 5 步：同步 Agent 配置

OpenClaw 会在以下位置生成 Agent 级别的配置文件：

```
~/.openclaw/agents/{agent-id}/agent/models.json
```

修改 `openclaw.json` 后，**删除这些文件**让 OpenClaw 重新生成：

```bash
# Linux / macOS
rm -f ~/.openclaw/agents/*/agent/models.json

# Windows (PowerShell)
Remove-Item -Path "$env:USERPROFILE\.openclaw\agents\*\agent\models.json" -Force -ErrorAction SilentlyContinue
```

重启 OpenClaw 后它会自动重新生成这些文件。

### 完整示例

一个包含一个服务商和两个模型的最小配置：

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

### `models.mode` 说明

- `**merge**`（默认）— 保留内置服务商，你的自定义服务商叠加在上面
- `**replace**` — 只使用你定义的服务商，移除所有内置的

大多数用户应保持 `merge`。

### 常见问题

**编辑后模型不出现？**
删除 Agent 级别的 `models.json` 并重启 OpenClaw：

```bash
rm -f ~/.openclaw/agents/*/agent/models.json
```

**使用模型时报 API 错误？**

- 检查 `baseUrl` 是否正确（有些 API 需要 `/v1` 后缀，有些不需要）
- 确认 `apiKey` 填写正确
- 确保 `api` 类型与服务商的实际协议匹配

## 许可证

MIT