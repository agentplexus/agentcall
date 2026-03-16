# AgentComms Tasks

## Completed

### FEAT_INBOUND (Phases 1-6) ✅

Bidirectional human-agent communication via Discord, Telegram, WhatsApp.

- [x] Phase 1: Foundation (Ent schemas, daemon skeleton)
- [x] Phase 2: Actor Router + tmux Adapter
- [x] Phase 3: Chat Transport (omnichat integration)
- [x] Phase 4: CLI Commands (send, interrupt, events, status)
- [x] Phase 5: Outbound Messages (reply, channels)
- [x] Phase 6: Polish (config validate, documentation)

### Phase 8: MCP Integration for Inbound ✅

**Goal:** Enable Claude Code to poll for inbound messages via MCP tools.

**Problem:** Currently inbound messages go to tmux via send-keys, but Claude Code can't read them programmatically during a session.

**Solution:** Add MCP tools that query the daemon for pending messages.

#### Tasks

- [x] 8.1. Add daemon client to MCP server
  - InboundManager wraps daemon.Client
  - Lazy connection (connects on first use)
  - Handles case where daemon is not running

- [x] 8.2. Implement `check_messages` MCP tool
  - Returns human messages for the agent
  - Filters by role=human and type=human_message
  - Parameters: `agent_id`, `limit`

- [x] 8.3. Implement `get_agent_events` MCP tool
  - Returns all events (messages, interrupts)
  - Supports pagination via `since_id`
  - Parameters: `agent_id`, `since_id`, `limit`

- [x] 8.4. Implement `daemon_status` MCP tool
  - Check if daemon is running
  - Returns agent count and providers

- [x] 8.5. Agent ID resolution
  - Uses AGENTCOMMS_AGENT_ID env var
  - Falls back to "default" if not set
  - Can be overridden per-call

- [x] 8.6. Update documentation
  - Added inbound tools to README
  - Documented check_messages, get_agent_events, daemon_status

#### Files Created

- `pkg/tools/inbound.go` - InboundManager and MCP tools
- `pkg/tools/inbound_test.go` - Unit tests

### MkDocs Documentation Site ✅

Created comprehensive documentation site using MkDocs.

#### Files Created

- `mkdocs.yml` - MkDocs configuration
- `docs/index.md` - Home/overview page
- `docs/getting-started.md` - Installation and setup guide
- `docs/cli.md` - CLI commands reference
- `docs/mcp-tools.md` - MCP tools reference
- `docs/configuration.md` - Configuration guide
- `docs/architecture.md` - System architecture

#### Features

- Material theme with dark mode toggle
- Code syntax highlighting
- Navigation tabs and sections
- Search functionality
- Links to existing design documents

### Unified JSON Configuration ✅

Migrated from split configuration (env vars + YAML) to a single unified JSON config.

#### Features

- Single `config.json` file combines MCP server + daemon config
- Environment variable substitution (`${VAR}` syntax) for secrets
- `agentcomms config init` command to generate template
- Backward compatible with legacy YAML config
- Full validation with helpful error messages

#### Files Created/Modified

- `pkg/config/unified.go` - UnifiedConfig struct with JSON tags
- `pkg/config/unified_test.go` - Unit tests
- `examples/config.json` - Example JSON config
- `cmd/agentcomms/commands.go` - Added config init command
- `docs/configuration.md` - Updated for JSON config
- `docs/getting-started.md` - Updated setup instructions
- `docs/cli.md` - Added config init documentation

### Phase 7: Multi-Agent Support ✅

Enabled multiple AI agents to coordinate via AgentComms.

#### Features

- Agent status tracking (online/offline lifecycle)
- Source agent field in events for agent-to-agent messages
- `list_agents` MCP tool - discover available agents
- `send_agent_message` MCP tool - send message to another agent
- `agent_message` IPC method for cross-agent routing
- Agent message formatting with source prefix: `[from: agent_a] ...`

#### Tasks

- [x] 7.1. Add `source_agent_id` field to Event schema
  - Distinguishes human→agent vs agent→agent messages
  - Generated via `go generate ./ent`

- [x] 7.2. Implement agent status tracking
  - Router tracks online status during RegisterAgent/UnregisterAgent
  - AgentStatuses() method returns map of agent→status
  - Database updated with status changes

- [x] 7.3. Add daemon IPC method `agent_message`
  - Creates event with source_agent_id and agent_id
  - Routes to destination agent's actor

- [x] 7.4. Add MCP tools
  - `list_agents` - lists agents with status
  - `send_agent_message` - sends to another agent

- [x] 7.5. Update actor to handle agent messages
  - Formats messages with source prefix
  - Delivers to tmux pane via adapter

- [x] 7.6. Add tests
  - TestServerAgentMessage - IPC method
  - TestRouterAgentStatuses - status tracking
  - Unit tests for new types

- [x] 7.7. Update documentation
  - docs/mcp-tools.md - new tools
  - docs/architecture.md - multi-agent flow

#### Files Modified

- `ent/schema/event.go` - Added source_agent_id field
- `internal/router/router.go` - Status tracking, AgentStatuses()
- `internal/router/actor.go` - handleAgentMessage()
- `internal/daemon/server.go` - handleAgentMessage()
- `internal/daemon/protocol.go` - AgentMessageParams, AgentMessageResult
- `internal/daemon/client.go` - AgentMessage()
- `pkg/tools/inbound.go` - list_agents, send_agent_message

## In Progress

None

## Design Notes

### Phase 8 Design Notes

**Message Flow:**
```
Human (Discord) → Daemon → Event Store
                              ↓
Claude Code ←── check_messages (MCP tool) ←── Daemon Client
```

**Agent ID Resolution:**
- MCP server needs to know which agent it represents
- Options:
  1. Environment variable: `AGENTCOMMS_AGENT_ID`
  2. Auto-register with daemon on startup
  3. Pass as parameter to each tool call

**Event Filtering:**
- Filter by `agent_id` and `role=human`
- Support `since_id` for pagination
- Return newest first or oldest first (configurable)

## Future

### Phase 9: Additional Transports

- Slack integration
- SMS via Twilio
- Email notifications

### Phase 10: Cloud Readiness

- PostgreSQL support
- Multi-tenant (tenant_id)
- Row-level security

### Voice Enhancements

- Telnyx phone provider (50% cheaper than Twilio via omnivoice-telnyx)
- Call recording for later review
- SMS fallback when call not answered

### Multi-Tool Expansion

- Gemini CLI adapter (generates gemini-extension.json, agents/*.toml)
- Gemini CLI plugin generation for agentcomms

### Core Enhancements

- Multi-user support (calling different users based on context)
