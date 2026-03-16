# FEAT_INBOUND: Technical Requirements Document

## Architecture Decision: Daemon as Hub

The daemon is the central hub for all agent communication. All clients (MCP server, CLI, direct API) connect to the daemon.

```
                         ┌──────────────────────┐
                         │       Daemon         │
                         │     (always on)      │
                         │                      │
   Discord ←────────────→│   Transport Layer    │
   Twilio  ←────────────→│                      │
                         │                      │
                         │   Event Store        │
                         │   (Ent + SQLite)     │
                         │                      │
                         │   Actor Router       │
                         │                      │
                         │   AgentBridge        │
                         │                      │
                         └──────────┬───────────┘
                                    │
                 ┌──────────────────┼──────────────────┐
                 │                  │                  │
                 ▼                  ▼                  ▼
           MCP Server             CLI            Direct API
         (thin client)                          (HTTP/gRPC)
                 │                  │                  │
                 └──────────────────┼──────────────────┘
                                    │
                                    ▼
                              Coding Agent
                         (Claude Code, etc.)
```

### Rationale

1. **Process lifecycle**: AI assistants spawn MCP servers ephemerally; daemon runs independently via launchd
2. **Single connection**: Only daemon connects to Discord/Twilio, avoiding token conflicts
3. **Unified event log**: All communication flows through daemon's event store
4. **Flexibility**: Agents can use MCP, CLI, or direct API based on their capabilities

## Component Specifications

### 1. Daemon

**Responsibilities:**
- Own Discord/Twilio connections (transports)
- Store all events (Ent + SQLite)
- Route inbound messages to agents via AgentBridge
- Expose API for outbound messages from clients
- Manage agent lifecycle (online/offline status)

**Process management:**
- macOS: launchd (`~/Library/LaunchAgents/com.agentcomms.plist`)
- Linux: systemd (future)
- Startup: `agentcomms daemon`

**API (local socket or HTTP):**

```
POST /api/v1/send
  - agent_id: string
  - message: string
  → event_id: string

POST /api/v1/interrupt
  - agent_id: string
  → success: bool

GET /api/v1/events
  - agent_id: string
  - since: timestamp (optional)
  → events: []Event

GET /api/v1/agents
  → agents: []Agent

GET /api/v1/health
  → status: string
```

**Socket location:** `~/.agentcomms/daemon.sock` (Unix socket for local IPC)

### 2. Event Store (Ent + SQLite)

**Database location:** `~/.agentcomms/data.db`

**Schema:**

```go
// ent/schema/event.go
package schema

import (
    "time"
    "entgo.io/ent"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/index"
)

type Event struct {
    ent.Schema
}

func (Event) Fields() []ent.Field {
    return []ent.Field{
        field.String("id").
            Unique().
            Immutable(),
        field.String("tenant_id").
            Default("local"),
        field.String("agent_id"),
        field.String("channel_id"),
        field.Enum("type").
            Values("human_message", "agent_message", "interrupt", "system"),
        field.Enum("role").
            Values("human", "agent", "system"),
        field.Time("timestamp").
            Default(time.Now),
        field.JSON("payload", map[string]any{}),
        field.Enum("status").
            Values("new", "delivered", "failed").
            Default("new"),
    }
}

func (Event) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("agent_id", "timestamp"),
        index.Fields("channel_id"),
        index.Fields("tenant_id"),
    }
}
```

```go
// ent/schema/agent.go
package schema

import (
    "entgo.io/ent"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/index"
)

type Agent struct {
    ent.Schema
}

func (Agent) Fields() []ent.Field {
    return []ent.Field{
        field.String("id").
            Unique().
            Immutable(),
        field.String("tenant_id").
            Default("local"),
        field.Enum("type").
            Values("tmux", "process"),
        field.JSON("config", map[string]any{}),
        field.String("channel_id"),
        field.Enum("status").
            Values("online", "offline").
            Default("offline"),
    }
}

func (Agent) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("channel_id").Unique(),
        index.Fields("tenant_id"),
    }
}
```

**Event ID format:** `evt_{ulid}` (sortable, unique)

### 3. Actor Router

Each agent has a dedicated goroutine with an inbox channel.

```go
type Router struct {
    agents map[string]*AgentActor
    mu     sync.RWMutex
}

type AgentActor struct {
    id      string
    inbox   chan *ent.Event
    adapter AgentAdapter
    store   *ent.Client
}

func (a *AgentActor) Run(ctx context.Context) {
    for {
        select {
        case evt := <-a.inbox:
            a.handle(ctx, evt)
        case <-ctx.Done():
            return
        }
    }
}

func (a *AgentActor) handle(ctx context.Context, evt *ent.Event) {
    switch evt.Type {
    case event.TypeHumanMessage:
        text := evt.Payload["text"].(string)
        if err := a.adapter.Send(text); err != nil {
            a.updateStatus(ctx, evt.ID, event.StatusFailed)
            return
        }
        a.updateStatus(ctx, evt.ID, event.StatusDelivered)

    case event.TypeInterrupt:
        _ = a.adapter.Interrupt()
        a.updateStatus(ctx, evt.ID, event.StatusDelivered)
    }
}
```

**Key properties:**
- Events processed sequentially per agent (no race conditions)
- Buffered channels prevent blocking transports
- Failed deliveries marked in event store

### 4. AgentBridge Adapters

```go
type AgentAdapter interface {
    Send(message string) error
    Interrupt() error
    Close() error
}
```

**tmux Adapter:**

```go
type TmuxAdapter struct {
    session string
    pane    string
}

func (t *TmuxAdapter) Send(msg string) error {
    // Escape special characters for tmux
    escaped := shellescape.Quote(msg)
    cmd := exec.Command("tmux", "send-keys", "-t",
        fmt.Sprintf("%s:%s", t.session, t.pane),
        escaped, "Enter")
    return cmd.Run()
}

func (t *TmuxAdapter) Interrupt() error {
    cmd := exec.Command("tmux", "send-keys", "-t",
        fmt.Sprintf("%s:%s", t.session, t.pane),
        "C-c")
    return cmd.Run()
}

func (t *TmuxAdapter) Close() error {
    return nil // No cleanup needed
}
```

**Config structure:**
```go
type TmuxConfig struct {
    Session string `json:"session"`
    Pane    string `json:"pane"`
}
```

### 5. Transport Layer

**Discord Transport:**

Extends existing `pkg/chat` to handle inbound messages.

```go
type DiscordTransport struct {
    session   *discordgo.Session
    router    *Router
    store     *ent.Client
    channelMap map[string]string // channel_id -> agent_id
}

func (d *DiscordTransport) Start(ctx context.Context) error {
    d.session.AddHandler(d.onMessageCreate)
    return d.session.Open()
}

func (d *DiscordTransport) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
    // Ignore bot's own messages
    if m.Author.ID == s.State.User.ID {
        return
    }

    // Find agent for this channel
    agentID, ok := d.channelMap[m.ChannelID]
    if !ok {
        return // Not a monitored channel
    }

    // Create event
    evt, err := d.store.Event.Create().
        SetID(newEventID()).
        SetAgentID(agentID).
        SetChannelID(fmt.Sprintf("discord:%s", m.ChannelID)).
        SetType(event.TypeHumanMessage).
        SetRole(event.RoleHuman).
        SetPayload(map[string]any{
            "text":      m.Content,
            "author_id": m.Author.ID,
            "author":    m.Author.Username,
        }).
        Save(context.Background())

    if err != nil {
        slog.Error("failed to save event", "error", err)
        return
    }

    // Route to agent
    d.router.Dispatch(agentID, evt)
}

func (d *DiscordTransport) Send(channelID, message string) error {
    _, err := d.session.ChannelMessageSend(channelID, message)
    return err
}
```

### 6. Configuration

**Location:** `~/.agentcomms/config.yaml`

```yaml
# Daemon configuration
daemon:
  socket: ~/.agentcomms/daemon.sock
  data_dir: ~/.agentcomms

# Discord configuration
discord:
  token: ${DISCORD_TOKEN}

# Agent definitions
agents:
  default:
    type: tmux
    config:
      session: main
      pane: "0"
    channel: discord:1234567890  # Discord channel ID
```

**Environment variables:**
- `AGENTCOMMS_CONFIG` - Config file path (default: `~/.agentcomms/config.yaml`)
- `DISCORD_TOKEN` - Discord bot token (can also be in config)

### 7. CLI

```
agentcomms daemon              Start the daemon
agentcomms daemon --foreground Run in foreground (for debugging)

agentcomms send <agent> <msg>  Send message to agent
agentcomms interrupt <agent>   Interrupt agent (Ctrl-C)

agentcomms agents              List configured agents
agentcomms agents status       Show agent online/offline status

agentcomms events <agent>      Show recent events
agentcomms events <agent> -f   Follow events (tail -f style)

agentcomms config init         Create default config
agentcomms config validate     Validate config file
```

**CLI connects to daemon via socket:**

```go
func sendMessage(agentID, message string) error {
    conn, err := net.Dial("unix", socketPath)
    if err != nil {
        return fmt.Errorf("daemon not running: %w", err)
    }
    defer conn.Close()

    req := &SendRequest{AgentID: agentID, Message: message}
    // ... send request, read response
}
```

### 8. MCP Server (Thin Client)

The existing MCP server becomes a thin client that proxies to the daemon.

```go
// Updated tool handler
func (t *Tools) sendMessage(ctx context.Context, in SendMessageInput) error {
    // Instead of direct Discord call:
    // return t.chatManager.SendMessage(ctx, in.Provider, in.ChatID, in.Message)

    // Proxy to daemon:
    return t.daemonClient.Send(in.ChatID, in.Message)
}
```

**Daemon client:**

```go
type DaemonClient struct {
    socketPath string
}

func (c *DaemonClient) Send(channelID, message string) error {
    conn, err := net.Dial("unix", c.socketPath)
    // ...
}
```

## Data Flow

### Inbound (Human → Agent)

```
1. Human types in Discord channel
2. Discord → DiscordTransport.onMessageCreate()
3. Create Event (type=human_message, status=new)
4. Save to Ent store
5. Router.Dispatch(agentID, event)
6. AgentActor receives event via inbox channel
7. AgentActor.handle() → TmuxAdapter.Send()
8. tmux send-keys delivers message to pane
9. Update Event status=delivered
```

### Outbound (Agent → Human)

```
1. Agent calls MCP tool or CLI
2. MCP Server / CLI → Daemon socket
3. Daemon creates Event (type=agent_message)
4. Daemon → DiscordTransport.Send()
5. Discord delivers message
6. Update Event status=delivered
```

## Error Handling

| Error | Handling |
|-------|----------|
| Daemon not running | CLI/MCP return clear error, suggest `agentcomms daemon` |
| Discord disconnected | Reconnect with backoff, queue events |
| tmux session not found | Mark agent offline, log error, don't crash |
| Event save failed | Log error, attempt retry, don't block transport |

## Testing Strategy

### Unit Tests
- Event store CRUD operations
- Actor router dispatch logic
- Config parsing

### Integration Tests
- Discord transport with mock Discord API
- tmux adapter with real tmux (in CI with tmux installed)
- Full flow: mock Discord → daemon → tmux

### Manual Testing
- Local Discord server for testing
- Real tmux sessions

## Security Considerations

1. **Socket permissions**: Daemon socket readable only by owner (0600)
2. **Config file**: Contains Discord token, should be 0600
3. **Input sanitization**: Escape shell characters before tmux send-keys
4. **Rate limiting**: Prevent message flooding to agents (future)

## Dependencies

**New:**
- `entgo.io/ent` - ORM and schema management
- `github.com/oklog/ulid/v2` - Event ID generation

**Existing (reuse):**
- `github.com/bwmarrin/discordgo` - Already in go.mod
- `github.com/spf13/cobra` - CLI framework (if not present)
- `gopkg.in/yaml.v3` - Config parsing

## File Structure

```
agentcomms/
├── cmd/
│   └── agentcomms/
│       └── main.go              # CLI entry point
├── internal/
│   ├── daemon/
│   │   ├── daemon.go            # Main daemon logic
│   │   ├── server.go            # Unix socket server
│   │   └── client.go            # Client for CLI/MCP
│   ├── router/
│   │   ├── router.go            # Actor router
│   │   └── actor.go             # Agent actor
│   ├── bridge/
│   │   ├── adapter.go           # Interface
│   │   └── tmux.go              # tmux adapter
│   ├── transport/
│   │   └── discord.go           # Discord transport
│   └── config/
│       └── config.go            # Config loading
├── ent/
│   ├── schema/
│   │   ├── event.go
│   │   └── agent.go
│   └── generate.go              # go:generate directive
├── pkg/                         # Existing packages
│   ├── chat/
│   ├── voice/
│   ├── config/
│   └── tools/
└── docs/
    └── design/
        ├── FEAT_INBOUND_PRD.md
        ├── FEAT_INBOUND_TRD.md
        └── FEAT_INBOUND_PLAN.md
```
