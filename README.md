# AgentComms

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go SAST][go-sast-svg]][go-sast-url]
[![Go Report Card][goreport-svg]][goreport-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![Visualization][viz-svg]][viz-url]
[![License][license-svg]][license-url]

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

**[Documentation](https://plexusone.github.io/agentcomms/)** | **[Getting Started](https://plexusone.github.io/agentcomms/getting-started/)** | **[MCP Tools](https://plexusone.github.io/agentcomms/mcp-tools/)**

An MCP plugin that enables voice calls and chat messaging for AI coding assistants. Start a task, walk away. Your phone rings when the AI is done, stuck, or needs a decision. Or get notified via Discord, Slack, Telegram, WhatsApp, or Gmail.

**Supports:** Claude Code, AWS Kiro CLI, Gemini CLI

**Built with the plexusone stack** - showcasing a complete voice and chat AI architecture in Go.

## Features

- 📞 **Phone Calls**: Real voice calls to your phone via Twilio—works with smartphones, smartwatches, landlines, or VoIP
- 💬 **Chat Messaging**: Send messages via Discord, Slack, Telegram, or WhatsApp
- 🔄 **Multi-turn Conversations**: Back-and-forth discussions, not just one-way notifications
- ⚡ **Smart Triggers**: Hooks that suggest calling/messaging when you're stuck or done with work
- 🔀 **Mix and Match**: Use voice, chat, or both based on your needs
- 🧠 **Parallel Execution**: AI continues working while waiting for your response—searching code, running tests, preparing next steps

## How It Works

AgentComms provides **bidirectional communication** between humans and AI agents:

```
                           AgentComms
                    ┌────────────────────┐
                    │                    │
  ┌──────────┐      │   ┌────────────┐   │      ┌──────────┐
  │ AI Agent │ ────▶│   │ MCP Server │   │◀──── │  Human   │
  │ Claude / │      │   │ (OUTBOUND) │   │      │ (Discord │
  │ Codex    │ ◀────│   └────────────┘   │────▶ │  Phone)  │
  └──────────┘      │                    │      └──────────┘
                    │   ┌────────────┐   │
                    │   │   Daemon   │   │
                    │   │ (INBOUND)  │   │
                    │   └────────────┘   │
                    │         │          │
                    │    ┌────┴─────┐    │
                    │    │   tmux   │    │
                    │    │   pane   │    │
                    │    └──────────┘    │
                    │                    │
                    └────────────────────┘
```

**Two communication modes:**

| Mode | Direction | Use Case |
|------|-----------|----------|
| **OUTBOUND** | Agent → Human | AI needs input, reports completion, escalates blockers |
| **INBOUND** | Human → Agent | Interrupt agent, send instructions, coordinate multiple agents |

### OUTBOUND (MCP Server)

1. **AI needs input** → Calls your phone or sends a chat message
2. **You respond** → Voice is transcribed, chat is read directly
3. **AI continues** → Uses your input to complete the task

### INBOUND (Daemon) - Preview

1. **You send a message** → Type in Discord channel or send SMS
2. **Daemon receives** → Routes to the correct agent via tmux
3. **Agent sees it** → Message appears in agent's terminal

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                   AgentComms                                │
├─────────────────────────────────────────────────────────────────────────────┤
│  OUTBOUND (MCP Server) - Agent → Human                                      │
│  ├── Voice Tools   - initiate_call, continue_call, speak_to_user, end_call  │
│  ├── Chat Tools    - send_message, list_channels, get_messages              │
│  ├── Voice Manager - Orchestrates calls via omnivoice                       │
│  └── Chat Manager  - Routes messages via omnichat                           │
├─────────────────────────────────────────────────────────────────────────────┤
│  INBOUND (Daemon)  - Human → Agent                                          │
│  ├── Router        - Actor-style event dispatcher (goroutine per agent)     │
│  ├── AgentBridge   - Adapters for tmux, process, etc.                       │
│  ├── Event Store   - SQLite database via Ent ORM                            │
│  └── Transports    - Discord, Twilio (receives human messages)              │
├─────────────────────────────────────────────────────────────────────────────┤
│  Shared Infrastructure                                                      │
│  ├── omnivoice     - Voice abstraction (TTS, STT, Transport, CallSystem)    │
│  ├── omnichat      - Chat abstraction (Discord, Telegram, WhatsApp)         │
│  ├── mcpkit        - MCP server with ngrok integration                      │
│  └── Ent           - Database ORM with SQLite/PostgreSQL support            │
├─────────────────────────────────────────────────────────────────────────────┤
│  Provider Implementations                                                   │
│  ├── Voice         - ElevenLabs, Deepgram, OpenAI, Twilio                   │
│  └── Chat          - Discord, Slack, Telegram, WhatsApp, Gmail              │
└─────────────────────────────────────────────────────────────────────────────┘
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

- Go 1.25+
- For voice: Twilio account + ngrok account
- For chat: Discord/Slack/Telegram bot token (optional)

### Build

```bash
cd /path/to/agentcomms
go mod tidy
go build -o agentcomms ./cmd/agentcomms
```

## Configuration

AgentComms uses a unified JSON configuration file that combines all settings.

### Quick Setup

```bash
# Generate configuration file
./agentcomms config init

# Or generate minimal config (chat only, no voice)
./agentcomms config init --minimal

# Set environment variables for secrets
export DISCORD_TOKEN=your_discord_bot_token
export SLACK_BOT_TOKEN=xoxb-your-bot-token
export SLACK_APP_TOKEN=xapp-your-app-token
export TWILIO_ACCOUNT_SID=ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
export TWILIO_AUTH_TOKEN=your_auth_token
export ELEVENLABS_API_KEY=your_elevenlabs_key
export DEEPGRAM_API_KEY=your_deepgram_key
export NGROK_AUTHTOKEN=your_ngrok_authtoken

# Validate configuration
./agentcomms config validate
```

### Configuration File

The config file at `~/.agentcomms/config.json` supports environment variable substitution:

```json
{
  "version": "1",
  "server": { "port": 3333 },
  "agents": [
    { "id": "claude", "type": "tmux", "tmux_session": "claude-code" }
  ],
  "voice": {
    "phone": {
      "account_sid": "${TWILIO_ACCOUNT_SID}",
      "auth_token": "${TWILIO_AUTH_TOKEN}",
      "number": "+15551234567",
      "user_number": "+15559876543"
    },
    "tts": { "provider": "elevenlabs", "api_key": "${ELEVENLABS_API_KEY}" },
    "stt": { "provider": "deepgram", "api_key": "${DEEPGRAM_API_KEY}" },
    "ngrok": { "auth_token": "${NGROK_AUTHTOKEN}" }
  },
  "chat": {
    "discord": { "enabled": true, "token": "${DISCORD_TOKEN}" },
    "channels": [
      { "channel_id": "discord:YOUR_CHANNEL_ID", "agent_id": "claude" }
    ]
  }
}
```

See [Configuration Guide](https://plexusone.github.io/agentcomms/configuration/) for full documentation.

## Usage

### Commands

AgentComms provides two main commands:

```bash
# Run MCP server (OUTBOUND - spawned by AI assistant)
./agentcomms serve

# Run daemon (INBOUND - background service for human messages)
./agentcomms daemon
```

Running `./agentcomms` without a subcommand defaults to `serve` for backwards compatibility.

### Running the MCP Server (OUTBOUND)

```bash
./agentcomms serve
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

### Running the Daemon (INBOUND) - Preview

The daemon enables human-to-agent communication. It runs as a background service and routes messages from Discord/Twilio to agents running in tmux.

```bash
./agentcomms daemon
```

Output:

```
INFO starting daemon data_dir=/Users/you/.agentcomms socket=/Users/you/.agentcomms/daemon.sock
INFO database initialized path=/Users/you/.agentcomms/data.db
INFO router initialized
INFO daemon started
```

**Data storage:** `~/.agentcomms/`

- `config.json` - Unified configuration file
- `data.db` - SQLite database (events, agents)
- `daemon.sock` - Unix socket for CLI/API

### Daemon CLI Commands

Once the daemon is running, use these CLI commands to interact with it:

```bash
# Check daemon status
./agentcomms status

# List configured agents
./agentcomms agents

# Send a message to an agent (appears in tmux pane)
./agentcomms send <agent-id> "Your message here"

# Send an interrupt (Ctrl-C) to an agent
./agentcomms interrupt <agent-id>

# View recent events for an agent
./agentcomms events <agent-id> --limit 20

# Send a reply to a chat channel (outbound from agent)
./agentcomms reply discord:123456789 "Task completed!"

# List configured chat channels
./agentcomms channels

# Validate configuration
./agentcomms config validate

# Show current configuration
./agentcomms config show
```

### Daemon Configuration

Generate and edit the configuration:

```bash
# Generate config file
./agentcomms config init

# Edit ~/.agentcomms/config.json with your settings

# Validate configuration
./agentcomms config validate
```

See the [Configuration Guide](https://plexusone.github.io/agentcomms/configuration/) for full details.

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

### Inbound Tools

These tools allow Claude Code to poll for messages sent by humans via the daemon.

#### check_messages

Check for new messages sent to this agent from humans via chat.

```json
{
  "agent_id": "claude",
  "limit": 10
}
```

Returns:

```json
{
  "messages": [
    {
      "id": "evt_01ABC123",
      "channel_id": "discord:123456789",
      "provider": "discord",
      "text": "Hey, can you also add unit tests?",
      "timestamp": "2024-01-15T10:30:00Z",
      "type": "human_message"
    }
  ],
  "agent_id": "claude",
  "has_more": false
}
```

#### get_agent_events

Get all recent events for an agent (messages, interrupts, status changes).

```json
{
  "agent_id": "claude",
  "since_id": "evt_01ABC123",
  "limit": 20
}
```

#### daemon_status

Check if the agentcomms daemon is running.

```json
{}
```

Returns:

```json
{
  "running": true,
  "started_at": "2024-01-15T09:00:00Z",
  "agents": 1,
  "providers": ["discord", "telegram"]
}
```

### Multi-Agent Tools

These tools enable agent-to-agent communication for task delegation and coordination.

#### list_agents

List all available agents and their status.

```json
{
  "include_offline": false
}
```

Returns:

```json
{
  "agents": [
    {"id": "backend", "type": "tmux", "status": "online", "target": "tmux:dev:0"},
    {"id": "frontend", "type": "tmux", "status": "online", "target": "tmux:dev:1"}
  ]
}
```

#### send_agent_message

Send a message to another agent.

```json
{
  "to_agent_id": "backend",
  "message": "Can you help me with the API implementation?"
}
```

Messages arrive at the destination agent with source prefix:

```
[from: frontend] Can you help me with the API implementation?
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
│   └── agentcomms/
│       ├── main.go          # CLI entry point (serve, daemon)
│       └── commands.go      # CLI commands (send, interrupt, reply, etc.)
├── internal/                # INBOUND infrastructure
│   ├── daemon/
│   │   ├── daemon.go        # Background daemon service
│   │   ├── server.go        # Unix socket server
│   │   ├── client.go        # Client library for IPC
│   │   ├── protocol.go      # JSON-RPC style protocol
│   │   └── config.go        # Daemon configuration (YAML)
│   ├── router/
│   │   ├── router.go        # Event dispatcher
│   │   └── actor.go         # Per-agent actor (goroutine)
│   ├── bridge/
│   │   ├── adapter.go       # Agent adapter interface
│   │   └── tmux.go          # tmux adapter
│   ├── transport/
│   │   └── chat.go          # Chat transport (omnichat)
│   └── events/
│       └── id.go            # Event ID generation
├── ent/                     # Database schema (Ent ORM)
│   └── schema/
│       ├── event.go         # Event entity
│       └── agent.go         # Agent entity
├── pkg/                     # OUTBOUND infrastructure
│   ├── voice/
│   │   └── manager.go       # Voice call orchestration
│   ├── chat/
│   │   └── manager.go       # Chat message routing
│   ├── config/
│   │   ├── config.go        # Legacy configuration
│   │   └── unified.go       # Unified JSON configuration
│   └── tools/
│       └── tools.go         # MCP tool definitions
├── examples/
│   └── config.json          # Example JSON configuration
├── docs/
│   └── design/              # Architecture documentation
│       ├── FEAT_INBOUND_PRD.md
│       ├── FEAT_INBOUND_TRD.md
│       └── FEAT_INBOUND_PLAN.md
├── go.mod
└── README.md
```

### Dependencies

- `github.com/plexusone/omnivoice` - Batteries-included voice abstraction
- `github.com/plexusone/omnichat` - Chat messaging abstraction
- `github.com/plexusone/omnivoice-twilio` - Twilio transport and call system
- `github.com/plexusone/mcpkit` - MCP server runtime
- `github.com/modelcontextprotocol/go-sdk` - MCP protocol SDK
- `entgo.io/ent` - Entity framework for Go (database ORM)
- `modernc.org/sqlite` - Pure Go SQLite driver

## Cost Estimate

| Service | Cost |
|---------|------|
| Twilio outbound calls | ~$0.014/min |
| Twilio phone number | ~$1.15/month |
| ElevenLabs TTS | ~$0.30/1K chars (~$0.03/min of speech) |
| ElevenLabs STT | ~$0.10/min (Scribe) |
| Deepgram TTS | ~$0.015/1K chars |
| Deepgram STT | ~$0.0043/min (Nova-2) |
| OpenAI TTS | ~$0.015/1K chars |
| OpenAI STT | ~$0.006/min (Whisper) |
| Discord/Telegram/Slack | Free |
| Gmail API | Free (500 emails/day) |
| ngrok (free tier) | $0 |

**Provider Recommendations:**

| Priority | TTS Provider | STT Provider | Total Cost/min | Notes |
|----------|--------------|--------------|----------------|-------|
| Lowest Cost | Deepgram | Deepgram | ~$0.03 | Best value, good quality |
| Best Quality | ElevenLabs | Deepgram | ~$0.05 | Premium voices, fast transcription |
| Balanced | OpenAI | OpenAI | ~$0.04 | Single API key, consistent quality |

*Costs are approximate and exclude Twilio phone charges (~$0.014/min).*

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
