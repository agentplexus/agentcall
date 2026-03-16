# Architecture

AgentComms provides bidirectional communication between AI agents and humans through two main components.

## System Overview

```
┌───────────────────────────────────────────────────────────────────────────┐
│                           agentcomms                                      │
├───────────────────────────────────────────────────────────────────────────┤
│  OUTBOUND (MCP Server) - Agent → Human                                    │
│  ├── Voice Tools: initiate_call, continue_call, speak_to_user, end_call   │
│  ├── Chat Tools:  send_message, list_channels, get_messages               │
│  ├── Inbound Tools: check_messages, get_agent_events, daemon_status       │
│  ├── Voice Manager - Orchestrates calls via omnivoice                     │
│  └── Chat Manager  - Routes messages via omnichat                         │
├───────────────────────────────────────────────────────────────────────────┤
│  INBOUND (Daemon) - Human → Agent                                         │
│  ├── Router       - Actor-style event dispatcher (goroutine per agent)    │
│  ├── AgentBridge  - Adapters for tmux, process, etc.                      │
│  ├── Event Store  - SQLite database via Ent ORM                           │
│  └── Transports   - Discord, Telegram, WhatsApp (receives human messages) │
├───────────────────────────────────────────────────────────────────────────┤
│  Shared Infrastructure                                                    │
│  ├── omnivoice    - Voice abstraction (TTS, STT, Transport, CallSystem)   │
│  ├── omnichat     - Chat abstraction (Discord, Telegram, WhatsApp)        │
│  ├── mcpkit       - MCP server with ngrok integration                     │
│  └── Ent          - Database ORM with SQLite/PostgreSQL support           │
└───────────────────────────────────────────────────────────────────────────┘
```

## OUTBOUND: MCP Server

The MCP server handles AI → Human communication.

### Components

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  AI Agent   │────▶│  MCP Server  │────▶│   Human     │
│  (Claude)   │     │              │     │  (Phone/    │
│             │◀────│  Tools:      │◀────│   Discord)  │
└─────────────┘     │  - Voice     │     └─────────────┘
                    │  - Chat      │
                    │  - Inbound   │
                    └──────────────┘
```

### Voice Flow

1. AI calls `initiate_call` with a message
2. Voice Manager creates call via Twilio
3. Message converted to speech (TTS)
4. Human responds via phone
5. Speech converted to text (STT)
6. Response returned to AI

### Chat Flow

1. AI calls `send_message` with provider and message
2. Chat Manager routes to appropriate provider
3. Message sent via Discord/Telegram/WhatsApp
4. Human receives notification

### Inbound Polling Flow

1. AI calls `check_messages`
2. MCP server queries daemon via Unix socket
3. Daemon returns recent human messages
4. AI processes messages and responds

## INBOUND: Daemon

The daemon handles Human → Agent communication.

### Components

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Human     │────▶│   Daemon     │────▶│  AI Agent   │
│  (Discord)  │     │              │     │  (tmux)     │
└─────────────┘     │  ┌────────┐  │     └─────────────┘
                    │  │ Router │  │
                    │  └───┬────┘  │
                    │      │       │
                    │  ┌───▼────┐  │
                    │  │ Event  │  │
                    │  │ Store  │  │
                    │  └────────┘  │
                    └──────────────┘
```

### Message Flow

1. Human sends message in Discord channel
2. Chat transport receives via omnichat
3. Event created and stored in SQLite
4. Router dispatches to agent's actor
5. Actor sends to tmux pane via adapter
6. Message appears in agent's terminal

### Actor Model

Each agent runs in its own goroutine (actor):

```go
type agentActor struct {
    id      string
    adapter Adapter
    events  chan *ent.Event
}
```

Benefits:

- Isolated failure domains
- No shared mutable state
- Simple message passing
- Easy to add/remove agents

### Event Store

All events are stored in SQLite via Ent ORM:

```
Event
├── id          (evt_01ABC123 - ULID format)
├── agent_id    (target agent)
├── channel_id  (discord:123, telegram:456)
├── type        (human_message, agent_message, interrupt)
├── role        (human, agent)
├── payload     (JSON - message text, metadata)
├── status      (pending, delivered, failed)
└── timestamp
```

## IPC: Unix Socket

The daemon exposes a JSON-RPC style API over Unix socket.

### Protocol

```
Request:  {"id": "...", "method": "...", "params": {...}}
Response: {"id": "...", "result": {...}} or {"id": "...", "error": {...}}
```

### Methods

| Method | Description |
|--------|-------------|
| `ping` | Health check |
| `status` | Daemon status |
| `agents` | List agents |
| `send` | Send message to agent |
| `interrupt` | Send Ctrl-C to agent |
| `events` | Get agent events |
| `reply` | Send to chat channel |
| `channels` | List channel mappings |

## Project Structure

```
agentcomms/
├── cmd/
│   └── agentcomms/
│       ├── main.go          # CLI entry point
│       └── commands.go      # CLI commands
├── internal/                # INBOUND infrastructure
│   ├── daemon/
│   │   ├── daemon.go        # Background service
│   │   ├── server.go        # Unix socket server
│   │   ├── client.go        # Client library
│   │   ├── protocol.go      # JSON-RPC protocol
│   │   └── config.go        # YAML configuration
│   ├── router/
│   │   ├── router.go        # Event dispatcher
│   │   └── actor.go         # Per-agent actor
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
│   ├── tools/
│   │   ├── tools.go         # Voice/chat tools
│   │   └── inbound.go       # Inbound message tools
│   ├── voice/
│   │   └── manager.go       # Voice call orchestration
│   ├── chat/
│   │   └── manager.go       # Chat message routing
│   └── config/
│       └── config.go        # Configuration
└── docs/                    # Documentation
```

## Dependencies

### plexusone Stack

| Package | Role |
|---------|------|
| omnivoice | Voice abstraction (TTS, STT, Transport) |
| omnichat | Chat abstraction (Discord, Telegram, WhatsApp) |
| omnivoice-twilio | Twilio transport and call system |
| mcpkit | MCP server runtime with ngrok |
| elevenlabs-go | ElevenLabs TTS/STT |
| omnivoice-deepgram | Deepgram TTS/STT |
| omnivoice-openai | OpenAI TTS/STT |

### Other Dependencies

| Package | Role |
|---------|------|
| entgo.io/ent | Entity framework for Go |
| modernc.org/sqlite | Pure Go SQLite driver |
| github.com/spf13/cobra | CLI framework |
| github.com/oklog/ulid | ULID generation |

## Security Considerations

### Socket Permissions

The Unix socket is created with mode 0600 (owner only).

### Token Storage

Chat tokens should be stored in:

- Environment variables (preferred)
- Config file with restricted permissions

### Event Data

All events are stored locally in SQLite. Consider:

- Database encryption for sensitive data
- Log rotation for large deployments
- Backup procedures for event history
