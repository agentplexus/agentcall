# AgentComms

Bidirectional communication between AI agents and humans via voice calls and chat messaging.

## Overview

AgentComms enables AI coding assistants like Claude Code to communicate with humans through:

- **Phone Calls**: Real voice calls via Twilio
- **Chat Messaging**: Discord, Telegram, WhatsApp via omnichat

**Two communication modes:**

| Mode | Direction | Use Case |
|------|-----------|----------|
| **OUTBOUND** | Agent → Human | AI needs input, reports completion, escalates blockers |
| **INBOUND** | Human → Agent | Interrupt agent, send instructions, coordinate tasks |

## How It Works

```
                           AgentComms
                    ┌──────────────────────┐
                    │                      │
  ┌──────────┐      │   ┌────────────┐     │      ┌──────────┐
  │ AI Agent │ ────▶│   │ MCP Server │     │◀──── │  Human   │
  │ Claude / │      │   │ (OUTBOUND) │     │      │ (Discord │
  │ Codex    │ ◀────│   └────────────┘     │────▶ │  Phone)  │
  └──────────┘      │                      │      └──────────┘
                    │   ┌────────────┐     │
                    │   │   Daemon   │     │
                    │   │ (INBOUND)  │     │
                    │   └────────────┘     │
                    │         │            │
                    │    ┌────┴────┐       │
                    │    │  tmux   │       │
                    │    │  pane   │       │
                    │    └─────────┘       │
                    └──────────────────────┘
```

### OUTBOUND (MCP Server)

1. **AI needs input** → Calls your phone or sends a chat message
2. **You respond** → Voice is transcribed, chat is read directly
3. **AI continues** → Uses your input to complete the task

### INBOUND (Daemon)

1. **You send a message** → Type in Discord channel
2. **Daemon receives** → Routes to the correct agent via tmux
3. **Agent sees it** → Message appears in agent's terminal
4. **Agent polls** → Uses `check_messages` MCP tool to read messages

## Features

- **Phone Calls**: Real voice calls to your phone via Twilio
- **Chat Messaging**: Send messages via Discord, Telegram, or WhatsApp
- **Multi-turn Conversations**: Back-and-forth discussions
- **Smart Triggers**: Hooks that suggest calling/messaging when stuck
- **Inbound Polling**: AI can check for messages during long tasks
- **Event Store**: SQLite database tracks all communication

## Quick Start

```bash
# Build
go build -o agentcomms ./cmd/agentcomms

# Run MCP server (for AI → Human)
./agentcomms serve

# Run daemon (for Human → AI)
./agentcomms daemon
```

See [Getting Started](getting-started.md) for detailed setup instructions.

## The plexusone Stack

AgentComms is built with the plexusone voice and chat AI stack:

| Package | Role |
|---------|------|
| **omnivoice** | Voice abstraction (TTS/STT) |
| **omnichat** | Chat abstraction (Discord, Telegram, WhatsApp) |
| **omnivoice-twilio** | Phone calls via Twilio |
| **mcpkit** | MCP server runtime |
| **elevenlabs-go** | ElevenLabs TTS/STT |
| **omnivoice-deepgram** | Deepgram TTS/STT |
