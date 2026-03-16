# Getting Started

## Prerequisites

- Go 1.21+
- For voice: Twilio account + ngrok account
- For chat: Discord/Telegram bot token (optional)

## Installation

### Build from Source

```bash
git clone https://github.com/plexusone/agentcomms.git
cd agentcomms
go mod tidy
go build -o agentcomms ./cmd/agentcomms
```

## Environment Variables

### Voice Configuration

```bash
# Twilio credentials
export AGENTCOMMS_PHONE_ACCOUNT_SID=ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
export AGENTCOMMS_PHONE_AUTH_TOKEN=your_auth_token
export AGENTCOMMS_PHONE_NUMBER=+15551234567      # Your Twilio number
export AGENTCOMMS_USER_PHONE_NUMBER=+15559876543  # Your personal phone

# Voice provider selection
export AGENTCOMMS_TTS_PROVIDER=elevenlabs  # "elevenlabs", "deepgram", or "openai"
export AGENTCOMMS_STT_PROVIDER=deepgram    # "elevenlabs", "deepgram", or "openai"

# API keys (based on selected providers)
export AGENTCOMMS_ELEVENLABS_API_KEY=your_elevenlabs_key
export AGENTCOMMS_DEEPGRAM_API_KEY=your_deepgram_key
export AGENTCOMMS_OPENAI_API_KEY=your_openai_key

# ngrok (required for voice)
export NGROK_AUTHTOKEN=your_ngrok_authtoken
```

### Chat Configuration

```bash
# Discord
export AGENTCOMMS_DISCORD_ENABLED=true
export AGENTCOMMS_DISCORD_TOKEN=your_discord_bot_token

# Telegram
export AGENTCOMMS_TELEGRAM_ENABLED=true
export AGENTCOMMS_TELEGRAM_TOKEN=your_telegram_bot_token

# WhatsApp
export AGENTCOMMS_WHATSAPP_ENABLED=true
export AGENTCOMMS_WHATSAPP_DB_PATH=./whatsapp.db
```

### Inbound Configuration

```bash
# Agent ID for MCP tools
export AGENTCOMMS_AGENT_ID=claude
```

## Running the MCP Server (OUTBOUND)

The MCP server enables AI agents to call humans and send chat messages.

```bash
./agentcomms serve
```

Output:

```
Starting agentcomms MCP server...
Voice providers: tts=elevenlabs stt=deepgram
Chat providers: [discord telegram]
MCP server ready
  Local:  http://localhost:3333/mcp
  Public: https://abc123.ngrok.io/mcp
```

## Running the Daemon (INBOUND)

The daemon enables humans to send messages to AI agents.

### 1. Create Configuration

```bash
mkdir -p ~/.agentcomms
cp examples/config.yaml ~/.agentcomms/config.yaml
```

Edit `~/.agentcomms/config.yaml`:

```yaml
log_level: info

agents:
  - id: claude
    type: tmux
    tmux_session: claude-code
    tmux_pane: "0"

chat:
  discord:
    token: "${DISCORD_TOKEN}"
    guild_id: "YOUR_GUILD_ID"

  channels:
    - channel_id: "discord:YOUR_CHANNEL_ID"
      agent_id: claude
```

### 2. Validate Configuration

```bash
./agentcomms config validate
```

### 3. Start the Daemon

```bash
./agentcomms daemon
```

Output:

```
INFO starting daemon data_dir=/Users/you/.agentcomms
INFO database initialized path=/Users/you/.agentcomms/data.db
INFO router initialized
INFO registered agent agent_id=claude type=tmux
INFO daemon started
```

## Claude Code Integration

### Option 1: MCP Configuration

Add to `~/.claude/settings.json`:

```json
{
  "mcpServers": {
    "agentcomms": {
      "command": "/path/to/agentcomms",
      "env": {
        "AGENTCOMMS_PHONE_ACCOUNT_SID": "ACxxx",
        "AGENTCOMMS_PHONE_AUTH_TOKEN": "xxx",
        "AGENTCOMMS_PHONE_NUMBER": "+15551234567",
        "AGENTCOMMS_USER_PHONE_NUMBER": "+15559876543",
        "NGROK_AUTHTOKEN": "xxx",
        "AGENTCOMMS_DISCORD_ENABLED": "true",
        "AGENTCOMMS_DISCORD_TOKEN": "xxx",
        "AGENTCOMMS_AGENT_ID": "claude"
      }
    }
  }
}
```

### Option 2: Generate Plugin Files

```bash
go run ./cmd/generate-plugin claude .
```

This creates:

- `.claude-plugin/plugin.json` - Plugin manifest
- `skills/phone-input/SKILL.md` - Voice calling skill
- `skills/chat-messaging/SKILL.md` - Chat messaging skill

## Testing the Setup

### Test Outbound (Agent → Human)

In Claude Code, the AI can use:

```
initiate_call: Call your phone
send_message: Send a Discord/Telegram message
```

### Test Inbound (Human → Agent)

```bash
# Check daemon is running
./agentcomms status

# Send a test message
./agentcomms send claude "Can you also add unit tests?"

# In Claude Code, AI can check for messages:
# Uses check_messages MCP tool
```

### Test Bidirectional

1. Start daemon in tmux session `claude-code`
2. Run Claude Code in that tmux session
3. Send message via Discord
4. AI uses `check_messages` to see your message
5. AI responds via `send_message`
