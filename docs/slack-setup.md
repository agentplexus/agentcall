# Slack App Setup Guide

This guide walks you through creating a Slack app for use with AgentComms.

## Overview

AgentComms uses Slack's **Socket Mode** for real-time messaging, which means:

- No public webhook URL required
- Works behind firewalls and NAT
- Requires two tokens: Bot Token and App Token

## Step 1: Create a Slack App

1. Go to [api.slack.com/apps](https://api.slack.com/apps)
2. Click **Create New App**
3. Choose **From scratch**
4. Enter an app name (e.g., "AgentComms Bot")
5. Select your workspace
6. Click **Create App**

## Step 2: Enable Socket Mode

1. In your app settings, go to **Socket Mode** (left sidebar)
2. Toggle **Enable Socket Mode** to ON
3. You'll be prompted to create an App-Level Token:
   - Token Name: `agentcomms-socket`
   - Scopes: `connections:write`
4. Click **Generate**
5. **Copy the token** (starts with `xapp-`) - this is your `SLACK_APP_TOKEN`

## Step 3: Configure Bot Permissions

1. Go to **OAuth & Permissions** (left sidebar)
2. Scroll to **Scopes** > **Bot Token Scopes**
3. Add these scopes:

| Scope | Purpose |
|-------|---------|
| `chat:write` | Send messages to channels |
| `channels:history` | Read messages in public channels |
| `channels:read` | View channel info |
| `groups:history` | Read messages in private channels |
| `groups:read` | View private channel info |
| `im:history` | Read direct messages |
| `im:read` | View DM info |
| `im:write` | Start DMs with users |
| `mpim:history` | Read group DMs |
| `mpim:read` | View group DM info |
| `reactions:read` | View emoji reactions |
| `users:read` | View user info |

## Step 4: Enable Event Subscriptions

1. Go to **Event Subscriptions** (left sidebar)
2. Toggle **Enable Events** to ON
3. Under **Subscribe to bot events**, add:

| Event | Purpose |
|-------|---------|
| `message.channels` | Messages in public channels |
| `message.groups` | Messages in private channels |
| `message.im` | Direct messages |
| `message.mpim` | Group direct messages |
| `reaction_added` | Emoji reactions added |
| `reaction_removed` | Emoji reactions removed |
| `member_joined_channel` | User joins a channel |
| `member_left_channel` | User leaves a channel |

4. Click **Save Changes**

## Step 5: Install the App

1. Go to **Install App** (left sidebar)
2. Click **Install to Workspace**
3. Review permissions and click **Allow**
4. **Copy the Bot User OAuth Token** (starts with `xoxb-`) - this is your `SLACK_BOT_TOKEN`

## Step 6: Invite Bot to Channels

The bot can only see messages in channels it's been invited to:

1. Open the Slack channel where you want AgentComms
2. Type `/invite @YourBotName` or click the channel name > Integrations > Add apps

## Step 7: Configure AgentComms

### Environment Variables

```bash
export SLACK_BOT_TOKEN=xoxb-your-bot-token
export SLACK_APP_TOKEN=xapp-your-app-token
```

Or use the `AGENTCOMMS_` prefix:

```bash
export AGENTCOMMS_SLACK_BOT_TOKEN=xoxb-your-bot-token
export AGENTCOMMS_SLACK_APP_TOKEN=xapp-your-app-token
export AGENTCOMMS_SLACK_ENABLED=true
```

### JSON Configuration

```json
{
  "chat": {
    "slack": {
      "enabled": true,
      "bot_token": "${SLACK_BOT_TOKEN}",
      "app_token": "${SLACK_APP_TOKEN}"
    },
    "channels": [
      {
        "channel_id": "slack:C0123456789",
        "agent_id": "claude"
      }
    ]
  }
}
```

## Finding Channel IDs

To get a Slack channel ID:

1. Open Slack in a web browser
2. Navigate to the channel
3. The URL will be: `https://app.slack.com/client/TXXXXXXXX/C0123456789`
4. The channel ID is the part starting with `C` (e.g., `C0123456789`)

Or right-click the channel name > View channel details > scroll to the bottom to see the Channel ID.

## Testing the Integration

1. Start AgentComms:

   ```bash
   ./agentcomms daemon
   ```

2. You should see:

   ```
   INFO Slack provider registered
   INFO slack bot connected user_id=U0123456789 team=YourWorkspace
   ```

3. Send a message in the configured channel
4. Check events:

   ```bash
   ./agentcomms events claude --limit 5
   ```

## Thread Replies

AgentComms supports Slack threads:

- Messages in threads are detected and marked as `ChatTypeThread`
- Use `ReplyTo` field with the thread timestamp to reply in a thread
- The `ReplyTo` value is the Slack message timestamp (e.g., `1234567890.123456`)

## Troubleshooting

### "slack auth test: missing_scope"

You're missing required OAuth scopes. Go to **OAuth & Permissions** and add the missing scope, then reinstall the app.

### Bot doesn't see messages

1. Ensure the bot is invited to the channel (`/invite @BotName`)
2. Check that `message.channels` (or appropriate event) is subscribed
3. Verify Socket Mode is enabled

### "slack app token required for socket mode"

You need both tokens:

- `SLACK_BOT_TOKEN` (xoxb-...) - from OAuth & Permissions
- `SLACK_APP_TOKEN` (xapp-...) - from Socket Mode settings

### Connection drops

Socket Mode connections may drop occasionally. AgentComms will automatically reconnect. If issues persist, check:

1. App-level token has `connections:write` scope
2. No firewall blocking outbound WebSocket connections

## Security Notes

- Never commit tokens to version control
- Use environment variables or a secrets manager
- Rotate tokens if compromised via **OAuth & Permissions** > **Regenerate**
- The App Token can be regenerated in **Socket Mode** settings
