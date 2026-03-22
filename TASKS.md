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

### FEAT_VOICE-ABSTRACTION: CallSystem Registry ✅

Multi-provider voice support via omnivoice registry pattern.

#### Features

- CallSystem registry in omnivoice (RegisterCallSystemProvider/GetCallSystemProvider)
- Telnyx CallSystem provider (omnivoice-telnyx) - 50% cheaper than Twilio
- Provider selection via config (`phone_provider: "twilio"` or `"telnyx"`)
- Observability hooks for TTS/STT/CallSystem (VoiceObserver, TTSHook, STTHook)

#### Repositories Modified

- `omnivoice-core` - Registry types, observability interfaces, CallSystem client
- `omnivoice` - Registry implementation, provider registration
- `omnivoice-telnyx` - New Telnyx CallSystem provider
- `agentcomms` - Switched to registry-based provider lookup

#### Files Modified (agentcomms)

- `pkg/voice/manager.go` - Uses omnivoice.GetCallSystemProvider()
- `go.mod` - Added omnivoice-telnyx replace directive

### Voice Enhancements: Recording & SMS Fallback ✅

Added call recording and SMS fallback when calls are not answered.

#### Features

- Call recording via `AGENTCOMMS_ENABLE_RECORDING=true`
- SMS fallback when call not answered via `AGENTCOMMS_SMS_FALLBACK_ENABLED=true`
- Customizable SMS message template with `{message}` placeholder
- SMS support in both Twilio and Telnyx providers

#### Configuration

```bash
# Enable call recording
AGENTCOMMS_ENABLE_RECORDING=true

# Enable SMS fallback
AGENTCOMMS_SMS_FALLBACK_ENABLED=true

# Custom SMS message (optional)
AGENTCOMMS_SMS_FALLBACK_MESSAGE="I tried calling but couldn't reach you: {message}"
```

#### Repositories Modified

- `omnivoice-core` - Added SMSProvider interface (`callsystem/sms.go`)
- `omnivoice-twilio` - Added SendSMS/SendSMSFrom methods
- `omnivoice-telnyx` - Added SendSMS/SendSMSFrom methods
- `agentcomms` - Config options, voice manager integration

#### Files Modified (agentcomms)

- `pkg/config/config.go` - Added EnableRecording, SMSFallbackEnabled, SMSFallbackMessage
- `pkg/voice/manager.go` - Recording option, SMS fallback logic

### SMS Inbound Transport & Webhook Server ✅

Added inbound SMS as a chat transport and webhook server for receiving Twilio/Telnyx callbacks.

#### Features

- Webhook server for Twilio and Telnyx callbacks
- Inbound SMS handled as chat messages (like Discord/Telegram)
- Support for voice call status events
- Twilio signature validation for security
- Automatic integration with omnichat router

#### Configuration

```bash
# Enable webhook server
AGENTCOMMS_WEBHOOK_ENABLED=true
AGENTCOMMS_WEBHOOK_PORT=3334  # Default: 3334

# Enable SMS as chat transport
AGENTCOMMS_SMS_ENABLED=true
```

#### Webhook Endpoints

| Endpoint | Description |
|----------|-------------|
| `/webhook/twilio/sms` | Incoming SMS from Twilio |
| `/webhook/twilio/voice` | Voice status callbacks from Twilio |
| `/webhook/telnyx/sms` | Incoming SMS from Telnyx |
| `/webhook/telnyx/voice` | Voice status callbacks from Telnyx |
| `/health` | Health check endpoint |

#### Files Created

- `internal/webhook/server.go` - HTTP webhook server
- `internal/webhook/telnyx.go` - Telnyx webhook parsing
- `pkg/chat/sms/provider.go` - SMS provider for omnichat

#### Files Modified

- `pkg/config/config.go` - Added SMSEnabled, WebhookEnabled, WebhookPort
- `pkg/chat/manager.go` - Added InitializeSMS() and SMSProvider()

### Phase 10: PostgreSQL + Multi-Tenancy ✅

Added PostgreSQL support with Row-Level Security (RLS) for multi-tenancy.

#### Features

- Database abstraction layer supporting SQLite and PostgreSQL
- Tenant context management (`internal/tenant`)
- Ent privacy rules for application-level tenant filtering
- PostgreSQL RLS policies for database-level isolation
- Backward compatible with single-tenant SQLite mode

#### Configuration

```yaml
# Multi-tenant PostgreSQL with RLS
database:
  driver: postgres
  dsn: postgres://user:pass@localhost:5432/agentcomms?sslmode=disable
  multi_tenant: true
  use_rls: true
```

#### Architecture

1. **Application-level filtering** (Ent privacy rules) - Works on both SQLite and PostgreSQL
2. **Database-level RLS** (PostgreSQL only) - Uses `SET LOCAL app.current_tenant` per transaction

#### Files Created

- `internal/tenant/context.go` - Tenant context management
- `internal/tenant/context_test.go` - Unit tests (100% coverage)
- `internal/database/database.go` - Database abstraction with driver selection
- `internal/database/database_test.go` - Unit tests
- `internal/database/rls.go` - PostgreSQL RLS policy application
- `internal/database/rls_test.go` - Unit tests
- `internal/database/postgres_driver.go` - RLS driver wrapper
- `internal/database/postgres_driver_test.go` - Unit tests
- `ent/rule/tenant.go` - Ent privacy rule for tenant filtering

#### Files Modified

- `internal/daemon/config.go` - Added DatabaseConfig struct
- `internal/daemon/daemon.go` - Uses database.Open(), applies RLS policies
- `internal/daemon/server.go` - Extracts tenant from request, adds to context
- `internal/daemon/protocol.go` - Added TenantID to Request struct
- `ent/schema/agent.go` - Added Policy() method
- `ent/schema/event.go` - Added Policy() method
- `ent/generate.go` - Enabled privacy and entql features
- `go.mod` - Added github.com/lib/pq v1.11.2

#### Test Coverage

| Package | Coverage |
|---------|----------|
| `internal/tenant` | 100.0% |
| `internal/database` | 29.1% |

Note: Database coverage is limited because PostgreSQL-specific code requires a real PostgreSQL instance.

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

### Phase 9: Slack Integration ✅

Added Slack as a chat transport provider.

#### Features

- Socket Mode connection (no public webhook URL required)
- Real-time message handling via Events API
- Thread support (replies in threads)
- Reaction events (added/removed)
- Member joined/left events
- Message edited/deleted events

#### Configuration

```bash
# Enable Slack
AGENTCOMMS_SLACK_ENABLED=true
AGENTCOMMS_SLACK_BOT_TOKEN=xoxb-your-bot-token
AGENTCOMMS_SLACK_APP_TOKEN=xapp-your-app-token
```

#### Repositories Modified

- `omnichat` - New Slack provider (`providers/slack/adapter.go`)
- `agentcomms` - Config, chat manager integration

#### Files Created (omnichat)

- `providers/slack/adapter.go` - Slack provider implementation

#### Files Modified (agentcomms)

- `pkg/config/config.go` - Added SlackEnabled, SlackBotToken, SlackAppToken
- `pkg/chat/manager.go` - Added Slack provider initialization
- `go.mod` - Added omnichat replace directive

#### Documentation

- `docs/slack-setup.md` - Slack App Setup Guide
- `docs/configuration.md` - Added Slack config section
- `docs/getting-started.md` - Added Slack env vars
- `README.md` - Added Slack to supported providers
- `mkdocs.yml` - Added Slack Setup to navigation

### Phase 12: Gmail Email Integration ✅

Added Gmail as a chat provider for outbound email notifications.

#### Features

- Gmail API with OAuth 2.0 authentication
- Automatic browser-based OAuth flow on first run
- Token storage for subsequent sessions
- HTML and plain text email support
- Customizable subject via metadata
- `me` shorthand for authenticated user's email

#### Configuration

```bash
# Enable Gmail
AGENTCOMMS_GMAIL_ENABLED=true
AGENTCOMMS_GMAIL_CREDENTIALS_FILE=~/.agentcomms/gmail_credentials.json

# Optional
AGENTCOMMS_GMAIL_TOKEN_FILE=~/.agentcomms/gmail_token.json
AGENTCOMMS_GMAIL_FROM_ADDRESS=me
```

#### Repositories Modified

- `gogoogle` - Added `SendSimple()` method to gmailutil for easier email sending
- `omnichat` - New Gmail provider (`providers/email/gmail/adapter.go`)
- `agentcomms` - Config, chat manager integration

#### Files Created (omnichat)

- `providers/email/gmail/adapter.go` - Gmail provider implementation

#### Files Modified (gogoogle)

- `gmailutil/v1/service.go` - Added `SendSimpleOpts` and `SendSimple()` method

#### Files Modified (agentcomms)

- `pkg/config/config.go` - Added GmailEnabled, GmailCredentialsFile, GmailTokenFile, GmailFromAddress
- `pkg/chat/manager.go` - Added Gmail provider initialization
- `go.mod` - Added gogoogle replace directive for local development

#### Documentation

- `docs/gmail-setup.md` - Gmail Setup Guide
- `docs/configuration.md` - Added Gmail config section
- `docs/getting-started.md` - Added Gmail env vars
- `mkdocs.yml` - Added Gmail Setup to navigation

## Future

### Phase 11: PostgreSQL Integration Tests

Add Docker-based integration tests for PostgreSQL-specific functionality.

#### Tasks

- [ ] 11.1. Add Docker Compose for PostgreSQL test environment
  - PostgreSQL 16 container
  - Test database initialization

- [ ] 11.2. Add integration tests with `//go:build integration` tag
  - `openPostgres` - test actual PostgreSQL connection
  - `ApplyRLSPolicies` - test RLS policy creation
  - `DropRLSPolicies` - test RLS policy removal
  - `setTenantContext` - test SET LOCAL execution
  - `Exec`/`Query` - test tenant context propagation

- [ ] 11.3. Add tenant isolation integration tests
  - Create events with different tenant IDs
  - Verify tenant A cannot see tenant B's data
  - Verify RLS enforcement at database level

- [ ] 11.4. Add CI workflow for integration tests
  - GitHub Actions with PostgreSQL service
  - Run on PR and push to main

#### Expected Coverage After

| Package | Current | Target |
|---------|---------|--------|
| `internal/tenant` | 100.0% | 100.0% |
| `internal/database` | 29.1% | 90%+ |

### Voice Enhancements

- Multi-user support (calling different users based on context)

### Multi-Tool Expansion

- Gemini CLI adapter (generates gemini-extension.json, agents/*.toml)
- Gemini CLI plugin generation for agentcomms
