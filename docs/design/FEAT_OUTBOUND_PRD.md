# FEAT_OUTBOUND: Agent-to-Human Communication

## Overview

Enable AI agents to contact humans through communication channels (Discord, WhatsApp, Twilio voice/SMS) via MCP tools.

**Direction terminology:**

- **OUTBOUND** (this feature): Agent → MCP → Channel → Human
- **INBOUND** (planned): Human → Channel → AgentBridge → Agent

## Status

**Implemented** - This is the current functionality of AgentComms.

## Capabilities

### Voice (Twilio)

MCP tools for phone calls:

| Tool | Description |
|------|-------------|
| `initiate_call` | Call the user, speak a message, receive response |
| `continue_call` | Continue conversation within active call |
| `speak_to_user` | Speak without waiting for response |
| `end_call` | End call with optional final message |

### Chat (Discord, Telegram, WhatsApp)

MCP tools for messaging:

| Tool | Description |
|------|-------------|
| `send_message` | Send message to a chat channel |
| `list_channels` | List available channels and status |
| `get_messages` | Retrieve recent messages from a channel |

## Architecture

```
Agent Runtime (Claude Code, etc.)
  │
  ▼
MCP Client
  │
  ▼
AgentComms MCP Server
  │
  ├── Voice Manager (Twilio)
  │     ├── TTS (ElevenLabs / OpenAI)
  │     └── STT (Deepgram / OpenAI)
  │
  └── Chat Manager
        ├── Discord
        ├── Telegram
        └── WhatsApp
```

## Use Cases

1. **Task completion notification**: Agent calls/messages when done
2. **Blocker escalation**: Agent contacts human when stuck
3. **Decision requests**: Agent asks for input on choices
4. **Status updates**: Periodic progress reports

## Configuration

Environment variables:

```bash
# Voice (Twilio)
AGENTCOMMS_PHONE_ACCOUNT_SID=...
AGENTCOMMS_PHONE_AUTH_TOKEN=...
AGENTCOMMS_PHONE_NUMBER=+15551234567
AGENTCOMMS_USER_PHONE_NUMBER=+15559876543
NGROK_AUTHTOKEN=...

# Chat (Discord)
AGENTCOMMS_DISCORD_ENABLED=true
AGENTCOMMS_DISCORD_TOKEN=...
```

## Limitations

1. **Pull-based**: Agent must initiate tool calls; no push notifications
2. **Single direction**: Human cannot interrupt or respond asynchronously
3. **No persistence**: Conversations not stored for replay

These limitations are addressed by FEAT_INBOUND.

## References

- [FEAT_INBOUND_PRD.md](./FEAT_INBOUND_PRD.md) - Human-to-agent communication
- [MCP Specification](https://spec.modelcontextprotocol.io/)
