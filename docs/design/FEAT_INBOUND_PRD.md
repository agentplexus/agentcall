# FEAT_INBOUND: Human-to-Agent Communication

## Overview

Enable humans to send messages to AI coding agents through communication channels (Discord, WhatsApp, Twilio), allowing real-time interruption and interaction with local agent runtimes.

**Direction terminology:**

- **OUTBOUND** (existing): Agent → MCP → Channel → Human
- **INBOUND** (this feature): Human → Channel → AgentBridge → Agent

## Problem Statement

Current AgentComms enables agents to contact humans (OUTBOUND), but humans cannot interrupt or message agents in return. MCP is pull-based (agent initiates tool calls), so human messages sit undelivered until the agent happens to poll.

For coding assistants like Claude Code or Codex CLI running in tmux, users need the ability to:

1. Interrupt a running agent ("stop", "pause")
2. Send follow-up instructions without waiting for the agent to ask
3. Coordinate multiple agents through separate channels
4. Monitor agent activity remotely

## Goals

1. **Real-time delivery**: Human messages reach agents within seconds
2. **Interrupt support**: Agents can be stopped/paused mid-task
3. **Multi-agent routing**: Each agent has its own channel/conversation
4. **Local-first**: Runs on a laptop without cloud infrastructure
5. **Cloud-ready**: Architecture supports future multi-tenant deployment
6. **Event-sourced**: All communication is logged for replay/debugging

## Non-Goals

1. Building a new agent runtime (use existing Claude Code, Codex CLI, etc.)
2. Replacing MCP (INBOUND complements OUTBOUND, both coexist)
3. Real-time collaboration features (Google Docs-style)

## Architecture

### High-Level Flow

```
Human
  │
  ▼
Communication Channel (Discord / WhatsApp / Twilio)
  │
  ▼
AgentComms Daemon
  │
  ├── Event Store (JSONL + SQLite)
  │
  ├── Actor Router
  │
  └── AgentBridge
        │
        ▼
      Local Agent (tmux / CLI process)
```

### Components

#### 1. Event Store

Append-only event log with query index.

**Storage:**

- `~/.agentcomms/events/{agent_id}.jsonl` - append-only log (source of truth)
- `~/.agentcomms/events.db` - SQLite index for queries

**Event Schema:**

```go
type Event struct {
    ID             string         `json:"id"`
    TenantID       string         `json:"tenant_id"`       // "local" for single-tenant
    ConversationID string         `json:"conversation_id"`
    AgentID        string         `json:"agent_id"`
    HumanID        *string        `json:"human_id,omitempty"`
    ChannelID      string         `json:"channel_id"`
    Type           string         `json:"type"`
    Role           string         `json:"role"`
    Timestamp      time.Time      `json:"timestamp"`
    Payload        map[string]any `json:"payload"`
    Refs           []string       `json:"refs,omitempty"`
    Status         string         `json:"status"`
}
```

**Event Types:**

| Type | Description |
|------|-------------|
| `human_message` | Message from human via chat/phone |
| `agent_message` | Agent output to human or another agent |
| `tool_call` | Agent invoking a tool |
| `tool_result` | Result from tool execution |
| `interrupt` | Stop/pause/cancel request |
| `system` | System events (agent started, errors) |
| `voice_transcript` | Phone call transcript |

#### 2. Actor Router

Goroutine-per-agent architecture to prevent race conditions.

```go
type AgentActor struct {
    id    string
    inbox chan Event
}

func (a *AgentActor) Start(ctx context.Context) {
    go func() {
        for {
            select {
            case evt := <-a.inbox:
                a.handle(evt)
            case <-ctx.Done():
                return
            }
        }
    }()
}
```

**Key properties:**

- Each agent has its own goroutine and channel
- Events processed sequentially per agent (no locks needed)
- Router dispatches events to correct agent actor

#### 3. AgentBridge

Adapters connecting events to local agent runtimes.

**tmux Adapter:**

```go
type TmuxAdapter struct {
    session string
    pane    string
}

func (t *TmuxAdapter) Send(msg string) error {
    return exec.Command("tmux", "send-keys", "-t",
        fmt.Sprintf("%s:%s", t.session, t.pane),
        msg, "Enter").Run()
}

func (t *TmuxAdapter) Interrupt() error {
    return exec.Command("tmux", "send-keys", "-t",
        fmt.Sprintf("%s:%s", t.session, t.pane),
        "C-c").Run()
}
```

**Process Adapter:**

For agents running as child processes with stdin/stdout.

#### 4. Transport Integration

Extend existing chat/voice managers to publish INBOUND events.

```go
// In Discord handler
func (d *DiscordTransport) onMessage(m *discordgo.MessageCreate) {
    evt := Event{
        Type:      "human_message",
        ChannelID: fmt.Sprintf("discord:%s", m.ChannelID),
        Payload:   map[string]any{"text": m.Content},
    }
    d.eventBus.Publish(evt)
}
```

### Directory Structure

```
agentcomms/
├── cmd/agentcomms/
├── internal/
│   ├── events/
│   │   ├── event.go         # Event type
│   │   ├── bus.go           # Event bus (pub/sub)
│   │   └── store.go         # JSONL + SQLite store
│   ├── router/
│   │   ├── router.go        # Actor router
│   │   └── agent_actor.go   # Per-agent actor
│   ├── agentbridge/
│   │   ├── adapter.go       # Adapter interface
│   │   ├── tmux.go          # tmux adapter
│   │   └── process.go       # Process adapter
│   └── transports/
│       ├── discord.go       # Extended for INBOUND
│       └── twilio.go        # Extended for INBOUND
├── pkg/
│   ├── chat/                # (existing)
│   ├── voice/               # (existing)
│   └── config/              # (existing)
└── ent/
    └── schema/
        ├── event.go
        ├── agent.go
        └── conversation.go
```

## Configuration

### Agent Registration

```yaml
# ~/.agentcomms/config.yaml
agents:
  backend:
    type: tmux
    session: agents
    pane: "1"
    channel: "discord:agent-backend"

  frontend:
    type: tmux
    session: agents
    pane: "2"
    channel: "discord:agent-frontend"

  devops:
    type: process
    command: ["claude-code", "--agent"]
    channel: "discord:agent-devops"
```

### Channel Mapping

Discord channels map to agents:

| Discord Channel | Agent |
|-----------------|-------|
| #agent-backend | backend |
| #agent-frontend | frontend |
| #agent-devops | devops |

## macOS Deployment

### launchd Service

```xml
<!-- ~/Library/LaunchAgents/com.agentcomms.plist -->
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.agentcomms</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/agentcomms</string>
        <string>daemon</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/agentcomms.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/agentcomms.err</string>
</dict>
</plist>
```

### CLI Commands

```bash
# Start daemon
agentcomms daemon

# Send message to agent
agentcomms send backend "run the tests"

# Interrupt agent
agentcomms interrupt backend

# List agents
agentcomms agents

# Tail agent events
agentcomms logs backend --follow
```

## Cloud Evolution

### Multi-Tenant Support

Local deployment uses `tenant_id = "local"`.

Cloud deployment:

- PostgreSQL replaces SQLite
- Row-Level Security (RLS) enforces tenant isolation
- NATS/Kafka replaces in-memory event bus

```sql
-- PostgreSQL RLS policy
CREATE POLICY tenant_isolation ON events
    USING (tenant_id = current_setting('app.tenant_id'));
```

### Ent Schema

Same schema works for SQLite (local) and PostgreSQL (cloud):

```go
// ent/schema/event.go
func (Event) Fields() []ent.Field {
    return []ent.Field{
        field.String("id").Unique(),
        field.String("tenant_id").Default("local"),
        field.String("conversation_id"),
        field.String("agent_id"),
        field.String("type"),
        field.String("role"),
        field.Time("timestamp"),
        field.JSON("payload", map[string]any{}),
        field.Strings("refs").Optional(),
        field.String("status").Default("new"),
    }
}
```

## Implementation Phases

### Phase 1: Core Infrastructure

- [ ] Event type and bus
- [ ] JSONL event store
- [ ] SQLite index with Ent
- [ ] Actor router

### Phase 2: AgentBridge

- [ ] Adapter interface
- [ ] tmux adapter
- [ ] Process adapter
- [ ] Agent configuration

### Phase 3: Transport Integration

- [ ] Discord INBOUND handler
- [ ] Twilio INBOUND handler (SMS)
- [ ] WhatsApp INBOUND handler

### Phase 4: CLI & Daemon

- [ ] `agentcomms daemon` command
- [ ] `agentcomms send/interrupt/logs` commands
- [ ] launchd plist generator
- [ ] Log rotation

### Phase 5: Cloud Readiness

- [ ] PostgreSQL driver support
- [ ] tenant_id propagation
- [ ] Cloud sync client (optional)

## Success Metrics

1. Message delivery latency < 2 seconds
2. Zero race conditions under concurrent agent load
3. Event replay reproduces exact conversation state
4. Daemon stability > 7 days without restart

## Open Questions

1. **MCP integration**: Should INBOUND events also be exposed via MCP resources/notifications for agents that poll?
2. **Voice interrupts**: How should phone call interrupts work (human says "stop" → transcription → interrupt event)?
3. **Agent discovery**: Auto-detect tmux panes vs explicit configuration?
4. **Rate limiting**: Prevent message flooding to agents?

## References

- [Beads](https://github.com/steveyegge/beads) - Task graph architecture inspiration
- [MCP Specification](https://spec.modelcontextprotocol.io/) - Model Context Protocol
- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) - Target agent runtime
