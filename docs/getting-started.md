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

## Configuration

AgentComms uses a single JSON configuration file for all settings.

### 1. Generate Configuration

```bash
# Full config with voice settings
./agentcomms config init

# Minimal config (chat only, no voice)
./agentcomms config init --minimal
```

This creates `~/.agentcomms/config.json`.

### 2. Set Environment Variables for Secrets

```bash
# Discord token
export DISCORD_TOKEN=your_discord_bot_token

# Telegram token (if using)
export TELEGRAM_BOT_TOKEN=your_telegram_bot_token

# Slack tokens (if using)
export SLACK_BOT_TOKEN=xoxb-your-bot-token
export SLACK_APP_TOKEN=xapp-your-app-token

# Gmail (if using)
export GMAIL_CREDENTIALS_FILE=~/.agentcomms/gmail_credentials.json

# Voice providers (if using voice)
export TWILIO_ACCOUNT_SID=ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
export TWILIO_AUTH_TOKEN=your_auth_token
export ELEVENLABS_API_KEY=your_elevenlabs_key
export DEEPGRAM_API_KEY=your_deepgram_key
export NGROK_AUTHTOKEN=your_ngrok_authtoken
```

### 3. Edit Configuration

Edit `~/.agentcomms/config.json`:

```json
{
  "version": "1",
  "agents": [
    {
      "id": "claude",
      "type": "tmux",
      "tmux_session": "claude-code",
      "tmux_pane": "0"
    }
  ],
  "chat": {
    "discord": {
      "enabled": true,
      "token": "${DISCORD_TOKEN}",
      "guild_id": "YOUR_GUILD_ID"
    },
    "channels": [
      {
        "channel_id": "discord:YOUR_CHANNEL_ID",
        "agent_id": "claude"
      }
    ]
  }
}
```

### 4. Validate Configuration

```bash
./agentcomms config validate
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
        "TWILIO_ACCOUNT_SID": "ACxxx",
        "TWILIO_AUTH_TOKEN": "xxx",
        "NGROK_AUTHTOKEN": "xxx",
        "DISCORD_TOKEN": "xxx",
        "ELEVENLABS_API_KEY": "xxx",
        "DEEPGRAM_API_KEY": "xxx",
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
