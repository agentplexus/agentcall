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

### Phase 7: Multi-Agent Support
- Multiple agents in config
- Agent status tracking
- Cross-agent messaging

### Phase 9: Additional Transports
- Slack integration
- SMS via Twilio
- Email notifications

### Phase 10: Cloud Readiness
- PostgreSQL support
- Multi-tenant (tenant_id)
- Row-level security
