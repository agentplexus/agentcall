# Configuration

AgentComms uses a unified JSON configuration file that combines all settings in one place.

## Quick Start

Generate a configuration file:

```bash
# Generate full config with voice settings
agentcomms config init

# Generate minimal config (chat only)
agentcomms config init --minimal

# Generate to specific path
agentcomms config init -o /path/to/config.json
```

## Configuration File

The configuration file is located at `~/.agentcomms/config.json`.

### Full Example

```json
{
  "version": "1",
  "server": {
    "port": 3333,
    "data_dir": "${HOME}/.agentcomms"
  },
  "logging": {
    "level": "info"
  },
  "agents": [
    {
      "id": "claude",
      "type": "tmux",
      "tmux_session": "claude-code",
      "tmux_pane": "0"
    }
  ],
  "voice": {
    "phone": {
      "provider": "twilio",
      "account_sid": "${TWILIO_ACCOUNT_SID}",
      "auth_token": "${TWILIO_AUTH_TOKEN}",
      "number": "+15551234567",
      "user_number": "+15559876543"
    },
    "tts": {
      "provider": "elevenlabs",
      "api_key": "${ELEVENLABS_API_KEY}",
      "voice": "Rachel",
      "model": "eleven_turbo_v2_5"
    },
    "stt": {
      "provider": "deepgram",
      "api_key": "${DEEPGRAM_API_KEY}",
      "model": "nova-2",
      "language": "en-US",
      "silence_duration_ms": 800
    },
    "ngrok": {
      "auth_token": "${NGROK_AUTHTOKEN}"
    },
    "transcript_timeout_ms": 180000
  },
  "chat": {
    "discord": {
      "enabled": true,
      "token": "${DISCORD_TOKEN}",
      "guild_id": ""
    },
    "telegram": {
      "enabled": false,
      "token": "${TELEGRAM_BOT_TOKEN}"
    },
    "whatsapp": {
      "enabled": false,
      "db_path": "${HOME}/.agentcomms/whatsapp.db"
    },
    "slack": {
      "enabled": false,
      "bot_token": "${SLACK_BOT_TOKEN}",
      "app_token": "${SLACK_APP_TOKEN}"
    },
    "gmail": {
      "enabled": false,
      "credentials_file": "${HOME}/.agentcomms/gmail_credentials.json",
      "token_file": "${HOME}/.agentcomms/gmail_token.json",
      "from_address": "me"
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

## Configuration Sections

### Server

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `port` | int | 3333 | Server port for MCP |
| `data_dir` | string | `~/.agentcomms` | Data directory path |

### Logging

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `level` | string | `info` | Log level: debug, info, warn, error |

### Agents

Define AI agents that receive messages via tmux.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique agent identifier |
| `type` | string | Yes | Agent type: `tmux` |
| `tmux_session` | string | For tmux | tmux session name |
| `tmux_pane` | string | No | tmux pane (default: "0") |

### Voice

Voice calling configuration (optional, omit if not using voice).

#### Phone

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `provider` | string | No | Phone provider: `twilio` (default) |
| `account_sid` | string | Yes | Twilio account SID |
| `auth_token` | string | Yes | Twilio auth token |
| `number` | string | Yes | Your Twilio phone number (E.164 format) |
| `user_number` | string | Yes | Recipient phone number (E.164 format) |

#### TTS (Text-to-Speech)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `provider` | string | `elevenlabs` | Provider: elevenlabs, deepgram, openai |
| `api_key` | string | Required | Provider API key |
| `voice` | string | `Rachel` | Voice ID (provider-specific) |
| `model` | string | `eleven_turbo_v2_5` | Model ID (provider-specific) |

#### STT (Speech-to-Text)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `provider` | string | `deepgram` | Provider: elevenlabs, deepgram, openai |
| `api_key` | string | Required | Provider API key |
| `model` | string | `nova-2` | Model ID (provider-specific) |
| `language` | string | `en-US` | BCP-47 language code |
| `silence_duration_ms` | int | 800 | Silence duration to detect end of speech |

#### Ngrok

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `auth_token` | string | Yes | Ngrok auth token |
| `domain` | string | No | Custom ngrok domain |

### Chat

Chat provider configuration for Discord, Telegram, WhatsApp.

#### Discord

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `enabled` | bool | No | Enable Discord integration |
| `token` | string | When enabled | Discord bot token |
| `guild_id` | string | No | Filter to specific server |

Get your bot token from [Discord Developer Portal](https://discord.com/developers/applications).

#### Telegram

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `enabled` | bool | No | Enable Telegram integration |
| `token` | string | When enabled | Telegram bot token |

Get your bot token from [@BotFather](https://t.me/botfather).

#### WhatsApp

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `enabled` | bool | No | Enable WhatsApp integration |
| `db_path` | string | When enabled | SQLite database path for session |

WhatsApp requires scanning a QR code on first connection.

#### Slack

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `enabled` | bool | No | Enable Slack integration |
| `bot_token` | string | When enabled | Slack bot token (xoxb-...) |
| `app_token` | string | When enabled | Slack app token for Socket Mode (xapp-...) |

Slack uses Socket Mode, which means no public webhook URL is required. See the [Slack Setup Guide](slack-setup.md) for detailed instructions.

#### Gmail

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `enabled` | bool | No | Enable Gmail integration |
| `credentials_file` | string | When enabled | Path to Google OAuth credentials JSON (client_secret.json) |
| `token_file` | string | No | Path to store OAuth token (default: `~/.agentcomms/gmail_token.json`) |
| `from_address` | string | No | Email address to send from (default: `me` for authenticated user) |

Gmail uses OAuth 2.0 for authentication. On first run, you'll need to complete the OAuth flow in your browser. See the [Gmail Setup Guide](gmail-setup.md) for detailed instructions.

#### Channel Mappings

Map chat channels to agents.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `channel_id` | string | Yes | Full channel ID (`provider:chatid`) |
| `agent_id` | string | Yes | Target agent ID |

Channel ID format:

- Discord: `discord:CHANNEL_ID`
- Telegram: `telegram:CHAT_ID`
- WhatsApp: `whatsapp:JID`
- Slack: `slack:CHANNEL_ID`
- Gmail: `gmail:EMAIL_ADDRESS`

## Environment Variable Substitution

The JSON config supports environment variable substitution using `${VAR}` or `$VAR` syntax:

```json
{
  "chat": {
    "discord": {
      "token": "${DISCORD_TOKEN}"
    }
  }
}
```

Use this pattern for secrets rather than hardcoding them in the config file.

## Validating Configuration

Check your configuration is valid:

```bash
agentcomms config validate
```

This checks:

- JSON syntax
- Required fields
- Agent configuration
- Tmux session existence
- Chat provider tokens
- Channel mapping references

## Viewing Configuration

Display the current configuration:

```bash
agentcomms config show
```

## Data Directory

The daemon stores data in `~/.agentcomms/`:

```
~/.agentcomms/
├── config.json    # Configuration file
├── data.db        # SQLite database
├── daemon.sock    # Unix socket for IPC
└── whatsapp.db    # WhatsApp session (if used)
```

## Legacy YAML Configuration

YAML configuration (`config.yaml`) is still supported for backward compatibility but is deprecated. Consider migrating to JSON:

```bash
# Generate new JSON config
agentcomms config init

# Edit to match your YAML settings
# Then remove the YAML file
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
