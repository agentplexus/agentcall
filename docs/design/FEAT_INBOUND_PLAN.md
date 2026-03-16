# FEAT_INBOUND: Implementation Plan

## Scope

**In scope (MVP):**
- Single agent: Discord channel → tmux pane
- Daemon with Unix socket API
- Ent + SQLite event store
- CLI commands: daemon, send, interrupt, agents, events
- Basic config file

**Deferred:**
- Multi-agent support
- Twilio/WhatsApp transports
- MCP server integration (proxy to daemon)
- ConversationID, HumanID, Refs fields
- Cloud deployment (PostgreSQL, RLS)
- launchd plist generator

## Phases

### Phase 1: Foundation ✅ COMPLETE

**Goal:** Ent schema, event store, basic daemon skeleton

**Completed:**

- [x] 1.1. Initialize Ent
- [x] 1.2. Implement Event schema (per TRD)
- [x] 1.3. Implement Agent schema (per TRD)
- [x] 1.4. Generate Ent code
- [x] 1.5. Create daemon skeleton (`internal/daemon/daemon.go`)
- [x] 1.6. Add CLI with cobra (`serve`, `daemon` subcommands)

**Files created:**

- `ent/schema/event.go` - Event entity
- `ent/schema/agent.go` - Agent entity
- `ent/*.go` - Generated Ent code
- `internal/daemon/daemon.go` - Daemon with SQLite init
- `cmd/agentcomms/main.go` - Refactored with cobra CLI

---

### Phase 2: Actor Router + tmux Adapter ✅ COMPLETE

**Goal:** Route events to tmux pane

**Completed:**

- [x] 2.1. Implement AgentAdapter interface
- [x] 2.2. Implement TmuxAdapter (send-keys, interrupt)
- [x] 2.3. Implement AgentActor (goroutine per agent)
- [x] 2.4. Implement Router (dispatch events)
- [x] 2.5. Wire router to daemon
- [x] 2.6. Add tests (tmux adapter, router)

**Files created:**

- `internal/bridge/adapter.go` - Adapter interface
- `internal/bridge/tmux.go` - tmux adapter
- `internal/bridge/tmux_test.go` - tmux tests
- `internal/router/router.go` - Event router
- `internal/router/actor.go` - Per-agent actor
- `internal/router/router_test.go` - Router tests
- `internal/events/id.go` - Event ID generation (evt_{ulid})

**Test results:** All tests passing, 0 lint issues

---

### Phase 3: Chat Transport (omnichat) ✅ COMPLETE

**Goal:** Chat messages (Discord, Telegram, WhatsApp) create events, flow to tmux

**Completed:**

- [x] 3.1. Implement ChatTransport using omnichat
  - Uses omnichat Router for multi-provider support
  - Registers message handler for all providers
  - Maps channel ID (provider:chatid) → agent ID

- [x] 3.2. Handle inbound messages
  - Filter by channel mapping
  - Create Event with type=human_message
  - Save to store with provider metadata
  - Dispatch to router

- [x] 3.3. Wire to daemon
  - Load chat config from config.yaml
  - Register Discord, Telegram, WhatsApp providers
  - Start transport on daemon start
  - Graceful disconnect on shutdown

- [x] 3.4. Add config parsing
  - `~/.agentcomms/config.yaml`
  - Multi-provider support (Discord, Telegram, WhatsApp)
  - Channel → agent mappings

**Files created:**

- `internal/transport/chat.go` - Chat transport using omnichat
- `internal/daemon/config.go` - Updated for multi-provider config
- `internal/daemon/config_test.go` - Config validation tests
- `examples/config.yaml` - Sample configuration

**Test:**
```bash
# Setup
1. Create Discord bot, add to server
2. Create ~/.agentcomms/config.yaml with bot token and channel mapping
3. Start tmux session
4. Start daemon

# Test
5. Type message in Discord channel
6. Verify message appears in tmux pane
```

**Note:** Using omnichat instead of direct discordgo gives us:
- Multi-provider support (Discord, Telegram, WhatsApp)
- Consistent message handling across providers
- Future voice support via omnivoice integration

---

### Phase 4: CLI Commands ✅ COMPLETE

**Goal:** CLI for sending messages and viewing events

**Completed:**

- [x] 4.1. Implement Unix socket server in daemon
  - `~/.agentcomms/daemon.sock`
  - JSON-RPC style protocol (`internal/daemon/protocol.go`)
  - Handlers: ping, status, send, interrupt, agents, events

- [x] 4.2. Implement daemon client library
  - `internal/daemon/client.go`
  - Connect to socket, send/receive messages
  - Typed methods: Ping, Status, Send, Interrupt, Agents, Events

- [x] 4.3. Implement CLI commands
  - `agentcomms send <agent> <message>` - send message
  - `agentcomms interrupt <agent>` - send Ctrl-C
  - `agentcomms agents` - list agents
  - `agentcomms events <agent>` - list recent events
  - `agentcomms status` - daemon health check

- [x] 4.4. Add tests
  - Protocol unit tests
  - Server/client integration tests

**Files created:**

- `internal/daemon/protocol.go` - RPC protocol definitions
- `internal/daemon/protocol_test.go` - Protocol tests
- `internal/daemon/server.go` - Unix socket server
- `internal/daemon/server_test.go` - Server/client tests
- `internal/daemon/client.go` - Client library
- `cmd/agentcomms/commands.go` - CLI commands

**Test:**
```bash
agentcomms daemon &
agentcomms status
agentcomms agents
agentcomms send claude "Hello from CLI"
agentcomms events claude
agentcomms interrupt claude
```

---

### Phase 5: Outbound (Agent → Human) ✅ COMPLETE

**Goal:** Messages sent via CLI/API appear in chat channels

**Completed:**

- [x] 5.1. Add outbound event handling
  - Event type=agent_message
  - ChatTransport.SendMessage() method via ChatSender interface
  - Status tracking (delivered/failed)

- [x] 5.2. Implement reply API endpoint
  - `MethodReply` and `MethodChannels` added to protocol
  - `handleReply` creates event and sends via ChatSender
  - `handleChannels` lists configured channel mappings

- [x] 5.3. Add CLI commands
  - `agentcomms reply <channel-id> <message>` - send to chat channel
  - `agentcomms channels` - list mapped channels

- [x] 5.4. Add tests
  - mockChatSender for testing
  - TestServerReplyAndChannels integration test

**Files modified:**

- `internal/daemon/protocol.go` - Added Reply/Channels types
- `internal/daemon/server.go` - Added ChatSender interface, handlers
- `internal/daemon/client.go` - Added Reply/Channels methods
- `internal/daemon/server_test.go` - Added reply/channels tests
- `cmd/agentcomms/commands.go` - Added reply/channels commands

**Test:**
```bash
agentcomms channels
agentcomms reply discord:123456 "Task complete!"
# Verify message appears in Discord channel
```

**Test results:** All 27 tests passing, 0 lint issues

---

### Phase 6: Polish ✅ COMPLETE

**Goal:** Production readiness for single-agent use

**Completed:**

- [x] 6.1. Error handling
  - Clear error messages for connection failures
  - Warnings for missing tmux sessions (graceful degradation)
  - Config validation catches errors early

- [x] 6.2. Logging
  - Structured logging (slog) throughout
  - Log levels configurable in config.yaml
  - Component-tagged log entries (server, router, actor)

- [x] 6.3. Config validation
  - `agentcomms config validate` command
  - Checks YAML syntax and required fields
  - Validates agent configuration
  - Checks tmux session existence
  - Validates Discord token format
  - Reports warnings for common issues
  - `agentcomms config show` displays current config

- [x] 6.4. Documentation
  - README updated with daemon CLI commands
  - README updated with daemon configuration
  - Example config.yaml with all options
  - Updated project structure documentation

- [x] 6.5. Basic tests
  - Unit tests for tmux adapter
  - Unit tests for router
  - Unit tests for config validation
  - Integration tests for server/client
  - All 27 tests passing, 0 lint issues

**Files modified:**

- `cmd/agentcomms/commands.go` - Added config validate/show commands
- `README.md` - Updated with CLI documentation
- `docs/design/FEAT_INBOUND_PLAN.md` - Updated status

**Deliverable:** Production-ready single-agent system

---

## Milestone Summary

| Phase | Deliverable | Status |
|-------|-------------|--------|
| 1 | Ent + daemon skeleton | ✅ Complete |
| 2 | Actor router + tmux | ✅ Complete |
| 3 | Chat transport (omnichat) | ✅ Complete |
| 4 | CLI commands | ✅ Complete |
| 5 | Outbound messages | ✅ Complete |
| 6 | Polish | ✅ Complete |

**FEAT_INBOUND complete!** Production-ready bidirectional human-agent communication.

## Implementation Order

```
Phase 1 (Foundation) ✅
    │
    ▼
Phase 2 (Router + tmux) ✅
    │
    ▼
Phase 3 (Chat/omnichat) ✅ ───► Chat → tmux works
    │                            (Discord, Telegram, WhatsApp)
    ▼
Phase 4 (CLI) ✅
    │
    ▼
Phase 5 (Outbound) ✅ ─────────► MVP: Bidirectional works
    │
    ▼
Phase 6 (Polish) ✅ ───────────► Production ready

ALL PHASES COMPLETE
```

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| tmux escaping issues | Test with special characters early; use shellescape library |
| Discord rate limits | Implement backoff; queue outbound messages |
| Ent learning curve | Start with simple queries; defer complex relationships |
| Socket permission issues | Test on fresh macOS; document chmod requirements |

## Future Phases (Post-MVP)

### Phase 7: Multi-Agent
- Multiple agents in config
- Multiple Discord channels
- Agent status tracking

### Phase 8: MCP Integration
- MCP server proxies to daemon
- Existing tools work via daemon

### Phase 9: Additional Transports
- Twilio SMS
- WhatsApp
- Slack

### Phase 10: Cloud Readiness
- PostgreSQL support
- tenant_id propagation
- RLS policies

## Getting Started

After approval, begin with:

```bash
# Phase 1.1 - Initialize Ent
go install entgo.io/ent/cmd/ent@latest
mkdir -p ent/schema
```

Then implement schemas per TRD.
