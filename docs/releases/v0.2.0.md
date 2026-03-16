# Release Notes: v0.2.0

**Release Date:** 2026-03-07

## Overview

AgentComms v0.2.0 expands from voice-only to a unified communications plugin for AI coding assistants. This release adds chat messaging support (Discord, Telegram, WhatsApp) alongside the existing phone call capabilities, and includes a module rename from `agentcall` to `agentcomms`.

## Highlights

- **Renamed from agentcall to agentcomms** with expanded scope: voice calls AND chat messaging for AI assistants
- **Chat messaging support** via Discord, Telegram, and WhatsApp using the omnichat stack
- **Simplified imports** with omnivoice v0.6.0 re-exported types

## Breaking Changes

| Change | Migration |
|--------|-----------|
| Module renamed to `github.com/plexusone/agentcomms` | Update import paths |
| Environment prefix changed to `AGENTCOMMS_*` | Rename env vars (legacy `AGENTCALL_*` still works) |

## New Features

### Chat Messaging Tools

- `send_message` - Send message via Discord, Telegram, or WhatsApp
- `list_channels` - List available chat providers and their status
- `get_messages` - Retrieve conversation history from a chat channel

### Voice Provider Selection

- Configurable TTS/STT providers: ElevenLabs, Deepgram, or OpenAI
- `AGENTCOMMS_TTS_PROVIDER` and `AGENTCOMMS_STT_PROVIDER` environment variables
- API keys only required for selected providers

## Security

- Fixed potential log injection vulnerability in user input handling

## Dependencies

| Package | Version | Change |
|---------|---------|--------|
| omnivoice | v0.6.0 | Re-exports callsystem types |
| omnichat | v0.3.0 | New - chat messaging |
| modelcontextprotocol/go-sdk | v1.4.0 | Updated |
| assistantkit | v0.11.0 | Updated |
| mcpkit | v0.4.0 | Updated |

## Upgrade Guide

1. **Update import paths:**
   ```go
   // Before
   import "github.com/agentplexus/agentcall/..."

   // After
   import "github.com/plexusone/agentcomms/..."
   ```

2. **Update environment variables (optional):**
   ```bash
   # Before
   export AGENTCALL_PHONE_NUMBER=+15551234567

   # After (recommended)
   export AGENTCOMMS_PHONE_NUMBER=+15551234567
   ```

   Note: Legacy `AGENTCALL_*` variables still work for backwards compatibility.

## Full Changelog

See [CHANGELOG.md](CHANGELOG.md) for the complete list of changes.
