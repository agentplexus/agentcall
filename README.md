# AgentComms

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go SAST][go-sast-svg]][go-sast-url]
[![Go Report Card][goreport-svg]][goreport-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![Visualization][viz-svg]][viz-url]
[![License][license-svg]][license-url]

An MCP plugin that enables voice calls and chat messaging for AI coding assistants. Start a task, walk away. Your phone rings when the AI is done, stuck, or needs a decision. Or get notified via Discord, Telegram, or WhatsApp.

**Supports:** Claude Code, AWS Kiro CLI, Gemini CLI

**Built with the plexusone stack** - showcasing a complete voice and chat AI architecture in Go.

## Features

- **Phone Calls**: Real voice calls to your phone via Twilio
- **Chat Messaging**: Send messages via Discord, Telegram, or WhatsApp
- **Multi-turn Conversations**: Back-and-forth discussions, not just one-way notifications
- **Smart Triggers**: Hooks that suggest calling/messaging when you're stuck or done with work
- **Mix and Match**: Use voice, chat, or both based on your needs

## Architecture

```
┌───────────────────────────────────────────────────────────────────────────┐
│                           agentcomms                                      │
├───────────────────────────────────────────────────────────────────────────┤
│  MCP Tools (via mcpkit)                                                   │
│  ├── Voice Tools                                                          │
│  │   ├── initiate_call  - Start a new call to the user                    │
│  │   ├── continue_call  - Continue conversation on active call            │
│  │   ├── speak_to_user  - Speak without waiting for response              │
│  │   └── end_call       - End the call with optional goodbye              │
│  └── Chat Tools                                                           │
│      ├── send_message   - Send message via Discord/Telegram/WhatsApp      │
│      ├── list_channels  - List available chat channels                    │
│      └── get_messages   - Get recent messages from a channel              │
├───────────────────────────────────────────────────────────────────────────┤
│  Managers                                                                 │
│  ├── Voice Manager - Orchestrates calls, TTS, STT                         │
│  └── Chat Manager  - Routes messages across chat providers                │
├───────────────────────────────────────────────────────────────────────────┤
│  omnivoice (voice abstraction layer)                                      │
│  ├── tts.Provider       - Text-to-Speech interface                        │
│  ├── stt.Provider       - Speech-to-Text interface                        │
│  ├── transport.Transport - Audio streaming interface                      │
│  └── callsystem.CallSystem - Phone call management interface              │
├───────────────────────────────────────────────────────────────────────────┤
│  omnichat (chat abstraction layer)                                        │
│  ├── provider.Provider  - Chat provider interface                         │
│  └── provider.Router    - Message routing and handling                    │
├───────────────────────────────────────────────────────────────────────────┤
│  Provider Implementations                                                 │
│  ├── Voice: ElevenLabs, Deepgram, OpenAI, Twilio                          │
│  └── Chat:  Discord, Telegram, WhatsApp                                   │
├───────────────────────────────────────────────────────────────────────────┤
│  mcpkit                                                                   │
│  - MCP server with HTTP/SSE transport                                     │
│  - Built-in ngrok integration for public webhooks                         │
└───────────────────────────────────────────────────────────────────────────┘
```

## The plexusone Stack

This project demonstrates the plexusone voice and chat AI stack:

| Package | Role | Description |
|---------|------|-------------|
| **omnivoice** | Voice Abstraction | Batteries-included TTS/STT with registry-based provider lookup |
| **omnichat** | Chat Abstraction | Provider-agnostic chat messaging interface |
| **elevenlabs-go** | Voice Provider | ElevenLabs streaming TTS and STT |
| **omnivoice-deepgram** | Voice Provider | Deepgram streaming TTS and STT |
| **omnivoice-openai** | Voice Provider | OpenAI TTS and STT |
| **omnivoice-twilio** | Phone Provider | Twilio transport and call system |
| **mcpkit** | Server | MCP server runtime with ngrok and multiple transport modes |

## Installation

### Prerequisites

- Go 1.24+
- For voice: Twilio account + ngrok account
- For chat: Discord/Telegram bot token (optional)

### Build

```bash
cd /path/to/agentcomms
go mod tidy
go build -o agentcomms ./cmd/agentcomms
```

## Configuration

### Environment Variables

```bash
# ===== Voice (optional) =====

# Twilio credentials
export AGENTCOMMS_PHONE_ACCOUNT_SID=ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
export AGENTCOMMS_PHONE_AUTH_TOKEN=your_auth_token
export AGENTCOMMS_PHONE_NUMBER=+15551234567      # Your Twilio number
export AGENTCOMMS_USER_PHONE_NUMBER=+15559876543  # Your personal phone

# Voice provider selection (default: elevenlabs for TTS, deepgram for STT)
export AGENTCOMMS_TTS_PROVIDER=elevenlabs  # "elevenlabs", "deepgram", or "openai"
export AGENTCOMMS_STT_PROVIDER=deepgram    # "elevenlabs", "deepgram", or "openai"

# API keys (based on selected providers)
export AGENTCOMMS_ELEVENLABS_API_KEY=your_elevenlabs_key  # or ELEVENLABS_API_KEY
export AGENTCOMMS_DEEPGRAM_API_KEY=your_deepgram_key      # or DEEPGRAM_API_KEY
export AGENTCOMMS_OPENAI_API_KEY=your_openai_key          # or OPENAI_API_KEY

# ngrok (required for voice)
export NGROK_AUTHTOKEN=your_ngrok_authtoken

# Optional voice settings
export AGENTCOMMS_TTS_VOICE=Rachel           # ElevenLabs voice
export AGENTCOMMS_TTS_MODEL=eleven_turbo_v2_5
export AGENTCOMMS_STT_MODEL=nova-2
export AGENTCOMMS_STT_LANGUAGE=en-US

# ===== Chat (optional) =====

# Discord
export AGENTCOMMS_DISCORD_ENABLED=true
export AGENTCOMMS_DISCORD_TOKEN=your_discord_bot_token  # or DISCORD_TOKEN
export AGENTCOMMS_DISCORD_GUILD_ID=optional_guild_id

# Telegram
export AGENTCOMMS_TELEGRAM_ENABLED=true
export AGENTCOMMS_TELEGRAM_TOKEN=your_telegram_bot_token  # or TELEGRAM_BOT_TOKEN

# WhatsApp
export AGENTCOMMS_WHATSAPP_ENABLED=true
export AGENTCOMMS_WHATSAPP_DB_PATH=./whatsapp.db

# ===== Server =====
export AGENTCOMMS_PORT=3333
export AGENTCOMMS_NGROK_DOMAIN=myapp.ngrok.io  # Optional custom domain
```

### Legacy Environment Variables

For backwards compatibility, `AGENTCALL_*` variables are also supported with `AGENTCOMMS_*` taking precedence.

## Usage

### Running the Server

```bash
./agentcomms
```

Output:

```
Starting agentcomms MCP server...
Using plexusone stack:
  - omnivoice (voice abstraction)
  - omnichat (chat abstraction)
  - mcpkit (MCP server)
Voice providers: tts=elevenlabs stt=deepgram
Chat providers: [discord telegram]
MCP server ready
  Local:  http://localhost:3333/mcp
  Public: https://abc123.ngrok.io/mcp
```

### Multi-Tool Support

agentcomms supports multiple AI coding assistants. Generate configuration files for your preferred tool:

```bash
# Generate for a specific tool
go run ./cmd/generate-plugin claude .   # Claude Code
go run ./cmd/generate-plugin kiro .     # AWS Kiro CLI
go run ./cmd/generate-plugin gemini .   # Gemini CLI

# Generate for all tools
go run ./cmd/generate-plugin all ./plugins
```

### Claude Code Integration

**Option 1: Use generated plugin files**

```bash
go run ./cmd/generate-plugin claude .
```

This creates:

- `.claude-plugin/plugin.json` - Plugin manifest
- `skills/phone-input/SKILL.md` - Voice calling skill
- `skills/chat-messaging/SKILL.md` - Chat messaging skill
- `commands/call.md` - `/call` slash command
- `commands/message.md` - `/message` slash command
- `.claude/settings.json` - Lifecycle hooks

**Option 2: Manual MCP configuration**

Add to `~/.claude/settings.json` or `.claude/settings.json`:

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
        "AGENTCOMMS_DISCORD_TOKEN": "xxx"
      }
    }
  }
}
```

## MCP Tools

### Voice Tools

#### initiate_call

Start a new call to the user.

```json
{
  "message": "Hey! I finished implementing the feature. Want me to walk you through it?"
}
```

Returns:

```json
{
  "call_id": "call-1-1234567890",
  "response": "Sure, go ahead and explain what you built."
}
```

#### continue_call

Continue an active call with another message.

```json
{
  "call_id": "call-1-1234567890",
  "message": "I added authentication using JWT. Should I also add refresh tokens?"
}
```

#### speak_to_user

Speak without waiting for a response (useful for status updates).

```json
{
  "call_id": "call-1-1234567890",
  "message": "Let me search for that in the codebase. Give me a moment..."
}
```

#### end_call

End the call with an optional goodbye message.

```json
{
  "call_id": "call-1-1234567890",
  "message": "Perfect! I'll get started on that. Talk soon!"
}
```

### Chat Tools

#### send_message

Send a message to a chat channel.

```json
{
  "provider": "discord",
  "chat_id": "123456789",
  "message": "I've finished the PR! Here's the link: https://github.com/..."
}
```

#### list_channels

List available chat channels and their status.

```json
{}
```

Returns:

```json
{
  "channels": [
    {"provider_name": "discord", "status": "connected"},
    {"provider_name": "telegram", "status": "connected"}
  ]
}
```

#### get_messages

Get recent messages from a chat conversation.

```json
{
  "provider": "telegram",
  "chat_id": "987654321",
  "limit": 5
}
```

## Use Cases

**Phone calls are ideal for:**

- Reporting significant task completion
- Requesting urgent clarification when blocked
- Discussing complex decisions
- Walking through code changes
- Multi-step processes needing back-and-forth

**Chat messaging is ideal for:**

- Asynchronous status updates
- Sharing links, code, or formatted content
- Non-urgent notifications
- Follow-up summaries

## Development

### Project Structure

```
agentcomms/
├── cmd/
│   ├── agentcomms/
│   │   └── main.go          # Entry point
│   ├── generate-plugin/
│   │   └── main.go          # Plugin generator
│   └── publish/
│       └── main.go          # Marketplace publisher
├── pkg/
│   ├── voice/
│   │   └── manager.go       # Voice call orchestration
│   ├── chat/
│   │   └── manager.go       # Chat message routing
│   ├── config/
│   │   └── config.go        # Configuration
│   └── tools/
│       └── tools.go         # MCP tool definitions
├── go.mod
└── README.md
```

### Dependencies

- `github.com/plexusone/omnivoice` - Batteries-included voice abstraction
- `github.com/plexusone/omnichat` - Chat messaging abstraction
- `github.com/plexusone/omnivoice-twilio` - Twilio transport and call system
- `github.com/plexusone/mcpkit` - MCP server runtime
- `github.com/modelcontextprotocol/go-sdk` - MCP protocol SDK

## Cost Estimate

| Service | Cost |
|---------|------|
| Twilio outbound calls | ~$0.014/min |
| Twilio phone number | ~$1.15/month |
| ElevenLabs TTS | ~$0.30/1K chars (~$0.03/min of speech) |
| ElevenLabs STT | ~$0.10/min (Scribe) |
| Deepgram TTS | ~$0.015/1K chars |
| Deepgram STT | ~$0.0043/min (Nova-2) |
| Discord/Telegram | Free |
| ngrok (free tier) | $0 |

## License

MIT

## Credits

Inspired by [ZeframLou/call-me](https://github.com/ZeframLou/call-me) (TypeScript).

Built with the plexusone stack:

- [omnivoice](https://github.com/plexusone/omnivoice) - Voice abstraction layer
- [omnichat](https://github.com/plexusone/omnichat) - Chat messaging abstraction
- [elevenlabs-go](https://github.com/plexusone/elevenlabs-go) - ElevenLabs provider
- [omnivoice-deepgram](https://github.com/plexusone/omnivoice-deepgram) - Deepgram provider
- [omnivoice-twilio](https://github.com/plexusone/omnivoice-twilio) - Twilio provider
- [mcpkit](https://github.com/plexusone/mcpkit) - MCP server runtime
- [assistantkit](https://github.com/plexusone/assistantkit) - Multi-tool plugin configuration

 [go-ci-svg]: https://github.com/plexusone/agentcomms/actions/workflows/go-ci.yaml/badge.svg?branch=main
 [go-ci-url]: https://github.com/plexusone/agentcomms/actions/workflows/go-ci.yaml
 [go-lint-svg]: https://github.com/plexusone/agentcomms/actions/workflows/go-lint.yaml/badge.svg?branch=main
 [go-lint-url]: https://github.com/plexusone/agentcomms/actions/workflows/go-lint.yaml
 [go-sast-svg]: https://github.com/plexusone/agentcomms/actions/workflows/go-sast-codeql.yaml/badge.svg?branch=main
 [go-sast-url]: https://github.com/plexusone/agentcomms/actions/workflows/go-sast-codeql.yaml
 [goreport-svg]: https://goreportcard.com/badge/github.com/plexusone/agentcomms
 [goreport-url]: https://goreportcard.com/report/github.com/plexusone/agentcomms
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/plexusone/agentcomms
 [docs-godoc-url]: https://pkg.go.dev/github.com/plexusone/agentcomms
 [viz-svg]: https://img.shields.io/badge/visualizaton-Go-blue.svg
 [viz-url]: https://mango-dune-07a8b7110.1.azurestaticapps.net/?repo=plexusone%2Fagentcomms
 [loc-svg]: https://tokei.rs/b1/github/plexusone/agentcomms
 [repo-url]: https://github.com/plexusone/agentcomms
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/plexusone/agentcomms/blob/master/LICENSE
