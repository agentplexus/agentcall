# AgentCall

[![Build Status][build-status-svg]][build-status-url]
[![Lint Status][lint-status-svg]][lint-status-url]
[![Go Report Card][goreport-svg]][goreport-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![License][license-svg]][license-url]

An MCP plugin that enables voice calls via phone for AI coding assistants. Start a task, walk away. Your phone rings when the AI is done, stuck, or needs a decision.

**Supports:** Claude Code, AWS Kiro CLI, Gemini CLI

**Built with the agentplexus stack** - showcasing a complete voice AI architecture in Go.

## Inspiration

This project is inspired by [ZeframLou/call-me](https://github.com/ZeframLou/call-me), an excellent TypeScript MCP plugin that pioneered the "AI calls you" pattern. We wanted to:

1. **Build a Go implementation** - Leverage Go's deployment simplicity and performance
2. **Exercise the AgentPlexus libraries** - Demonstrate the omnivoice abstraction layer with pluggable providers
3. **Use premium voice providers** - ElevenLabs for natural TTS, Deepgram for accurate STT

### Comparison

| Aspect | agentcall (Go) | call-me (TypeScript) |
|--------|----------------|----------------------|
| **Deployable size** | 53 MB (single binary) | 68 MB (node_modules) |
| **Runtime required** | None | Node.js/Bun |
| **Dependencies** | Compiled in | 122 npm packages |
| **Distribution** | Single file copy | npm install |
| **TTS Provider** | ElevenLabs or Deepgram (configurable) | OpenAI |
| **STT Provider** | ElevenLabs or Deepgram (configurable) | OpenAI |
| **Phone Provider** | Twilio | Twilio/Telnyx |

The Go binary is self-contained with no runtime dependencies, making deployment as simple as copying a single file.

## Architecture

```
┌────────────────────────────────────────────────────────────────────────────┐
│                           agentcall                                        │
├────────────────────────────────────────────────────────────────────────────┤
│  MCP Tools (via mcpkit)                                                    │
│  ├── initiate_call  - Start a new call to the user                         │
│  ├── continue_call  - Continue conversation on active call                 │
│  ├── speak_to_user  - Speak without waiting for response                   │
│  └── end_call       - End the call with optional goodbye                   │
├────────────────────────────────────────────────────────────────────────────┤
│  Call Manager                                                              │
│  - Orchestrates calls, TTS, STT                                            │
│  - Manages call state and conversation history                             │
├────────────────────────────────────────────────────────────────────────────┤
│  omnivoice (abstraction layer)                                             │
│  ├── tts.StreamingProvider  - Text-to-Speech interface                     │
│  ├── stt.StreamingProvider  - Speech-to-Text interface                     │
│  ├── transport.Transport    - Audio streaming interface                    │
│  └── callsystem.CallSystem  - Phone call management interface              │
├────────────────────────────────────────────────────────────────────────────┤
│  Provider Implementations (configurable per function)                      │
│  ├── go-elevenlabs       - ElevenLabs TTS/STT (natural voices)             │
│  ├── omnivoice-deepgram  - Deepgram TTS/STT (accurate transcripts)         │
│  └── omnivoice-twilio                                                      │
│      ├── Transport via Twilio Media Streams WebSocket                      │
│      └── CallSystem via Twilio REST API                                    │
├────────────────────────────────────────────────────────────────────────────┤
│  mcpkit                                                                    │
│  - MCP server with HTTP/SSE transport                                      │
│  - Built-in ngrok integration for public webhooks                          │
│  - Library-mode for direct function calls                                  │
└────────────────────────────────────────────────────────────────────────────┘
```

## The agentplexus Stack

This project demonstrates the agentplexus voice AI stack:

| Package | Role | Description |
|---------|------|-------------|
| **omnivoice** | Abstraction | Provider-agnostic interfaces for TTS, STT, Transport, CallSystem |
| **go-elevenlabs** | Voice Provider | ElevenLabs streaming TTS and STT |
| **omnivoice-deepgram** | Voice Provider | Deepgram streaming TTS and STT |
| **omnivoice-twilio** | Phone Provider | Twilio transport and call system |
| **mcpkit** | Server | MCP server runtime with ngrok and multiple transport modes |

### Why This Architecture?

1. **Provider Independence**: Switch providers via configuration without code changes
2. **Mix and Match**: Use different providers for TTS and STT (e.g., ElevenLabs TTS + Deepgram STT)
3. **Testability**: Mock interfaces for unit testing without real phone calls
4. **Premium Quality**: Choose the best provider for each function

## Installation

### Prerequisites

- Go 1.24+
- Twilio account with:
  - Account SID and Auth Token
  - Phone number capable of making outbound calls
- ngrok account with auth token

### Build

```bash
cd /path/to/agentcall
go mod tidy
go build -o agentcall ./cmd/agentcall
```

## Configuration

Set the following environment variables:

```bash
# Required: Twilio credentials
export AGENTCALL_PHONE_ACCOUNT_SID=ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
export AGENTCALL_PHONE_AUTH_TOKEN=your_auth_token
export AGENTCALL_PHONE_NUMBER=+15551234567      # Your Twilio number
export AGENTCALL_USER_PHONE_NUMBER=+15559876543  # Your personal phone

# Optional: Voice provider selection (default: elevenlabs for TTS, deepgram for STT)
export AGENTCALL_TTS_PROVIDER=elevenlabs  # "elevenlabs" or "deepgram"
export AGENTCALL_STT_PROVIDER=deepgram    # "elevenlabs" or "deepgram"

# Required if using ElevenLabs (TTS or STT)
export AGENTCALL_ELEVENLABS_API_KEY=your_elevenlabs_key
# or: export ELEVENLABS_API_KEY=your_elevenlabs_key

# Required if using Deepgram (TTS or STT)
export AGENTCALL_DEEPGRAM_API_KEY=your_deepgram_key
# or: export DEEPGRAM_API_KEY=your_deepgram_key

# Required: ngrok
export NGROK_AUTHTOKEN=your_ngrok_authtoken

# Optional: ElevenLabs voice (default: Rachel)
export AGENTCALL_TTS_VOICE=Rachel

# Optional: ElevenLabs model (default: eleven_turbo_v2_5)
export AGENTCALL_TTS_MODEL=eleven_turbo_v2_5

# Optional: Deepgram model (default: nova-2)
export AGENTCALL_STT_MODEL=nova-2

# Optional: STT language (default: en-US)
export AGENTCALL_STT_LANGUAGE=en-US

# Optional: Server port (default: 3333)
export AGENTCALL_PORT=3333

# Optional: Custom ngrok domain (requires paid plan)
export AGENTCALL_NGROK_DOMAIN=myapp.ngrok.io
```

### ElevenLabs Voices

Popular voices: `Rachel`, `Adam`, `Antoni`, `Arnold`, `Bella`, `Domi`, `Elli`, `Josh`, `Sam`

See [ElevenLabs Voice Library](https://elevenlabs.io/voice-library) for the full list.

## Usage

### Running the Server

```bash
./agentcall
```

Output:

```
Starting agentcall MCP server...
Using agentplexus stack:
  - omnivoice (voice abstraction)
  - omnivoice-twilio (Twilio implementation)
  - mcpruntime (MCP server with ngrok)
MCP server ready
  Local:  http://localhost:3333/mcp
  Public: https://abc123.ngrok.io/mcp
Twilio webhooks configured:
  Voice:   https://abc123.ngrok.io/voice
  Stream:  https://abc123.ngrok.io/media-stream
  Status:  https://abc123.ngrok.io/status
```

### Multi-Tool Support

agentcall supports multiple AI coding assistants. Generate configuration files for your preferred tool:

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
- `commands/call.md` - `/call` slash command
- `.claude/settings.json` - Lifecycle hooks

**Option 2: Manual MCP configuration**

Add to `~/.claude/settings.json` or `.claude/settings.json`:

```json
{
  "mcpServers": {
    "agentcall": {
      "command": "/path/to/agentcall",
      "env": {
        "AGENTCALL_PHONE_ACCOUNT_SID": "ACxxx",
        "AGENTCALL_PHONE_AUTH_TOKEN": "xxx",
        "AGENTCALL_PHONE_NUMBER": "+15551234567",
        "AGENTCALL_USER_PHONE_NUMBER": "+15559876543",
        "NGROK_AUTHTOKEN": "xxx"
      }
    }
  }
}
```

### Kiro CLI Integration

```bash
go run ./cmd/generate-plugin kiro .
```

This creates:
- `.kiro/settings/mcp.json` - MCP server configuration
- `.kiro/agents/voice-caller.json` - Voice calling agent

**Manual configuration** - Add to `.kiro/settings/mcp.json`:

```json
{
  "mcpServers": {
    "agentcall": {
      "command": "/path/to/agentcall",
      "env": {
        "AGENTCALL_PHONE_ACCOUNT_SID": "${AGENTCALL_PHONE_ACCOUNT_SID}",
        "AGENTCALL_PHONE_AUTH_TOKEN": "${AGENTCALL_PHONE_AUTH_TOKEN}",
        "AGENTCALL_PHONE_NUMBER": "${AGENTCALL_PHONE_NUMBER}",
        "AGENTCALL_USER_PHONE_NUMBER": "${AGENTCALL_USER_PHONE_NUMBER}",
        "NGROK_AUTHTOKEN": "${NGROK_AUTHTOKEN}"
      }
    }
  }
}
```

### Gemini CLI Integration

```bash
go run ./cmd/generate-plugin gemini .
```

This creates:
- `gemini-extension.json` - Extension manifest
- `commands/call.toml` - Call command
- `agents/voice-caller.toml` - Voice calling agent

## MCP Tools

### initiate_call

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

### continue_call

Continue an active call with another message.

```json
{
  "call_id": "call-1-1234567890",
  "message": "I added authentication using JWT. Should I also add refresh tokens?"
}
```

### speak_to_user

Speak without waiting for a response (useful for status updates).

```json
{
  "call_id": "call-1-1234567890",
  "message": "Let me search for that in the codebase. Give me a moment..."
}
```

### end_call

End the call with an optional goodbye message.

```json
{
  "call_id": "call-1-1234567890",
  "message": "Perfect! I'll get started on that. Talk soon!"
}
```

## Use Cases

**Ideal for:**

- Reporting significant task completion
- Requesting clarification when blocked
- Discussing complex decisions
- Walking through code changes
- Multi-step processes needing back-and-forth

**Not ideal for:**

- Simple yes/no questions (use text)
- Status updates that don't need discussion
- Information already in the conversation

## Development

### Project Structure

```
agentcall/
├── cmd/
│   └── agentcall/
│       └── main.go          # Entry point
├── pkg/
│   ├── callmanager/
│   │   └── manager.go       # Call orchestration
│   ├── config/
│   │   └── config.go        # Configuration
│   └── tools/
│       └── tools.go         # MCP tool definitions
├── go.mod
└── README.md
```

### Dependencies

- `github.com/agentplexus/omnivoice` - Voice abstraction layer
- `github.com/agentplexus/go-elevenlabs` - ElevenLabs TTS provider
- `github.com/agentplexus/omnivoice-deepgram` - Deepgram STT provider
- `github.com/agentplexus/omnivoice-twilio` - Twilio transport and call system
- `github.com/agentplexus/mcpkit` - MCP server runtime
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
| ngrok (free tier) | $0 |

**Example configurations:**

| Config | TTS | STT | Approx. Cost/min |
|--------|-----|-----|------------------|
| ElevenLabs + Deepgram (default) | ElevenLabs | Deepgram | ~$0.05/min |
| All Deepgram | Deepgram | Deepgram | ~$0.03/min |
| All ElevenLabs | ElevenLabs | ElevenLabs | ~$0.15/min |

*Note: Costs vary by plan and usage. ElevenLabs and Deepgram offer free tiers for testing.*

## License

MIT

## Credits

Inspired by [ZeframLou/call-me](https://github.com/ZeframLou/call-me) (TypeScript).

Built with the agentplexus stack:

- [omnivoice](https://github.com/agentplexus/omnivoice) - Voice abstraction layer
- [go-elevenlabs](https://github.com/agentplexus/go-elevenlabs) - ElevenLabs TTS provider
- [omnivoice-deepgram](https://github.com/agentplexus/omnivoice-deepgram) - Deepgram STT provider
- [omnivoice-twilio](https://github.com/agentplexus/omnivoice-twilio) - Twilio transport and call system
- [mcpkit](https://github.com/agentplexus/mcpkit) - MCP server runtime
- [assistantkit](https://github.com/agentplexus/assistantkit) - Multi-tool plugin configuration

 [build-status-svg]: https://github.com/agentplexus/agentcall/actions/workflows/ci.yaml/badge.svg?branch=main
 [build-status-url]: https://github.com/agentplexus/agentcall/actions/workflows/ci.yaml
 [lint-status-svg]: https://github.com/agentplexus/agentcall/actions/workflows/lint.yaml/badge.svg?branch=main
 [lint-status-url]: https://github.com/agentplexus/agentcall/actions/workflows/lint.yaml
 [goreport-svg]: https://goreportcard.com/badge/github.com/agentplexus/agentcall
 [goreport-url]: https://goreportcard.com/report/github.com/agentplexus/agentcall
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/agentplexus/agentcall
 [docs-godoc-url]: https://pkg.go.dev/github.com/agentplexus/agentcall
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/agentplexus/agentcall/blob/master/LICENSE
 [used-by-svg]: https://sourcegraph.com/github.com/agentplexus/agentcall/-/badge.svg
 [used-by-url]: https://sourcegraph.com/github.com/agentplexus/agentcall?badge