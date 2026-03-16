# Configuration

AgentComms uses two configuration methods:

1. **Environment variables** - For the MCP server (OUTBOUND)
2. **YAML config file** - For the daemon (INBOUND)

## MCP Server Configuration (Environment)

The MCP server is configured via environment variables.

### Voice Settings

```bash
# Twilio credentials (required for voice)
AGENTCOMMS_PHONE_ACCOUNT_SID=ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
AGENTCOMMS_PHONE_AUTH_TOKEN=your_auth_token
AGENTCOMMS_PHONE_NUMBER=+15551234567       # Your Twilio number
AGENTCOMMS_USER_PHONE_NUMBER=+15559876543  # Recipient phone

# ngrok (required for voice webhooks)
NGROK_AUTHTOKEN=your_ngrok_authtoken
AGENTCOMMS_NGROK_DOMAIN=myapp.ngrok.io     # Optional custom domain

# Voice provider selection
AGENTCOMMS_TTS_PROVIDER=elevenlabs   # elevenlabs, deepgram, openai
AGENTCOMMS_STT_PROVIDER=deepgram     # elevenlabs, deepgram, openai

# Provider API keys
AGENTCOMMS_ELEVENLABS_API_KEY=xxx    # or ELEVENLABS_API_KEY
AGENTCOMMS_DEEPGRAM_API_KEY=xxx      # or DEEPGRAM_API_KEY
AGENTCOMMS_OPENAI_API_KEY=xxx        # or OPENAI_API_KEY

# Optional voice settings
AGENTCOMMS_TTS_VOICE=Rachel
AGENTCOMMS_TTS_MODEL=eleven_turbo_v2_5
AGENTCOMMS_STT_MODEL=nova-2
AGENTCOMMS_STT_LANGUAGE=en-US
```

### Chat Settings

```bash
# Discord
AGENTCOMMS_DISCORD_ENABLED=true
AGENTCOMMS_DISCORD_TOKEN=your_bot_token    # or DISCORD_TOKEN
AGENTCOMMS_DISCORD_GUILD_ID=optional_id

# Telegram
AGENTCOMMS_TELEGRAM_ENABLED=true
AGENTCOMMS_TELEGRAM_TOKEN=your_bot_token   # or TELEGRAM_BOT_TOKEN

# WhatsApp
AGENTCOMMS_WHATSAPP_ENABLED=true
AGENTCOMMS_WHATSAPP_DB_PATH=./whatsapp.db
```

### Inbound Settings

```bash
# Agent ID for inbound message tools
AGENTCOMMS_AGENT_ID=claude

# Server port
AGENTCOMMS_PORT=3333
```

## Daemon Configuration (YAML)

The daemon configuration file is located at `~/.agentcomms/config.yaml`.

### Full Example

```yaml
# Logging level: debug, info, warn, error
log_level: info

# Agent definitions
agents:
  # Coding agent receiving messages via tmux
  - id: claude
    type: tmux
    tmux_session: claude-code
    tmux_pane: "0"

  # Another agent in a different session
  - id: assistant
    type: tmux
    tmux_session: assistant
    tmux_pane: "0"

# Chat configuration (via omnichat)
chat:
  # Discord configuration
  discord:
    token: "${DISCORD_TOKEN}"      # Can use env vars
    guild_id: "YOUR_GUILD_ID"      # Optional: filter to specific server

  # Telegram configuration
  telegram:
    token: "${TELEGRAM_BOT_TOKEN}"

  # WhatsApp configuration
  whatsapp:
    db_path: "${HOME}/.agentcomms/whatsapp.db"

  # Map chat channels to agents
  channels:
    - channel_id: "discord:123456789012345678"
      agent_id: claude

    - channel_id: "discord:987654321098765432"
      agent_id: assistant

    - channel_id: "telegram:123456789"
      agent_id: claude
```

### Agent Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique agent identifier |
| `type` | string | Yes | Agent type: `tmux` |
| `tmux_session` | string | For tmux | tmux session name |
| `tmux_pane` | string | No | tmux pane (default: "0") |

### Chat Provider Configuration

#### Discord

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `token` | string | Yes | Discord bot token |
| `guild_id` | string | No | Filter to specific server |

Get your bot token from [Discord Developer Portal](https://discord.com/developers/applications).

#### Telegram

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `token` | string | Yes | Telegram bot token |

Get your bot token from [@BotFather](https://t.me/botfather).

#### WhatsApp

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `db_path` | string | Yes | SQLite database path for session |

WhatsApp requires scanning a QR code on first connection.

### Channel Mapping

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `channel_id` | string | Yes | Full channel ID (`provider:chatid`) |
| `agent_id` | string | Yes | Target agent ID |

Channel ID format:

- Discord: `discord:CHANNEL_ID`
- Telegram: `telegram:CHAT_ID`
- WhatsApp: `whatsapp:JID`

## Environment Variable Substitution

The YAML config supports environment variable substitution:

```yaml
chat:
  discord:
    token: "${DISCORD_TOKEN}"      # Uses $DISCORD_TOKEN
    guild_id: "${DISCORD_GUILD}"   # Uses $DISCORD_GUILD
```

## Validating Configuration

Check your configuration is valid:

```bash
agentcomms config validate
```

This checks:

- YAML syntax
- Required fields
- Agent configuration
- Tmux session existence
- Chat provider tokens
- Channel mapping references

## Data Directory

The daemon stores data in `~/.agentcomms/`:

```
~/.agentcomms/
├── config.yaml    # Configuration file
├── data.db        # SQLite database
├── daemon.sock    # Unix socket for IPC
└── whatsapp.db    # WhatsApp session (if used)
```

## Provider Cost Estimates

| Service | Cost |
|---------|------|
| Twilio outbound calls | ~$0.014/min |
| Twilio phone number | ~$1.15/month |
| ElevenLabs TTS | ~$0.30/1K chars |
| ElevenLabs STT | ~$0.10/min |
| Deepgram TTS | ~$0.015/1K chars |
| Deepgram STT | ~$0.0043/min |
| OpenAI TTS | ~$0.015/1K chars |
| OpenAI STT | ~$0.006/min |
| Discord/Telegram | Free |

**Provider Recommendations:**

| Priority | TTS | STT | Total/min | Notes |
|----------|-----|-----|-----------|-------|
| Lowest Cost | Deepgram | Deepgram | ~$0.03 | Best value |
| Best Quality | ElevenLabs | Deepgram | ~$0.05 | Premium voices |
| Balanced | OpenAI | OpenAI | ~$0.04 | Single API key |
