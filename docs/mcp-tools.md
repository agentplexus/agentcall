# MCP Tools

AgentComms provides MCP (Model Context Protocol) tools that AI assistants can use for communication.

## Voice Tools

These tools enable phone calls via Twilio.

### initiate_call

Start a new call to the user.

**Input:**

```json
{
  "message": "Hey! I finished implementing the feature. Want me to walk you through it?"
}
```

**Output:**

```json
{
  "call_id": "call-1-1234567890",
  "response": "Sure, go ahead and explain what you built."
}
```

**When to use:**

- Reporting significant task completion
- Requesting urgent clarification when blocked
- Discussing complex decisions
- Walking through code changes

### continue_call

Continue an active call with another message.

**Input:**

```json
{
  "call_id": "call-1-1234567890",
  "message": "I added authentication using JWT. Should I also add refresh tokens?"
}
```

**Output:**

```json
{
  "response": "Yes, add refresh tokens for better security."
}
```

### speak_to_user

Speak without waiting for a response.

**Input:**

```json
{
  "call_id": "call-1-1234567890",
  "message": "Let me search for that in the codebase. Give me a moment..."
}
```

**Output:**

```json
{
  "success": true
}
```

**When to use:**

- Acknowledgments before time-consuming operations
- Status updates during a call

### end_call

End the call with an optional goodbye message.

**Input:**

```json
{
  "call_id": "call-1-1234567890",
  "message": "Perfect! I'll get started on that. Talk soon!"
}
```

**Output:**

```json
{
  "duration_seconds": 120.5
}
```

## Chat Tools

These tools enable messaging via Discord, Telegram, and WhatsApp.

### send_message

Send a message to a chat channel.

**Input:**

```json
{
  "provider": "discord",
  "chat_id": "123456789",
  "message": "I've finished the PR! Here's the link: https://github.com/...",
  "reply_to": "optional_message_id"
}
```

**Output:**

```json
{
  "success": true
}
```

**Provider options:** `discord`, `telegram`, `whatsapp`

### list_channels

List available chat channels and their status.

**Input:**

```json
{}
```

**Output:**

```json
{
  "channels": [
    {"provider_name": "discord", "status": "connected"},
    {"provider_name": "telegram", "status": "connected"}
  ]
}
```

### get_messages

Get recent messages from a chat conversation.

**Input:**

```json
{
  "provider": "telegram",
  "chat_id": "987654321",
  "limit": 5
}
```

**Output:**

```json
{
  "messages": [
    {
      "id": "msg_123",
      "content": "Can you also add unit tests?",
      "author": "user123",
      "timestamp": "2024-01-15T10:30:00Z"
    }
  ]
}
```

## Inbound Tools

These tools allow AI to check for messages sent by humans via the daemon.

!!! note "Requires Daemon"
    These tools require the agentcomms daemon to be running.
    Start it with `agentcomms daemon`.

### check_messages

Check for new messages sent to this agent from humans via chat.

**Input:**

```json
{
  "agent_id": "claude",
  "limit": 10
}
```

**Output:**

```json
{
  "messages": [
    {
      "id": "evt_01ABC123",
      "channel_id": "discord:123456789",
      "provider": "discord",
      "text": "Hey, can you also add unit tests?",
      "timestamp": "2024-01-15T10:30:00Z",
      "type": "human_message"
    }
  ],
  "agent_id": "claude",
  "has_more": false,
  "last_seen_id": "evt_01ABC123"
}
```

**When to use:**

- Periodically during long-running tasks
- After completing a subtask
- When waiting for build/test results

**Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `agent_id` | string | env var or "default" | Agent to check messages for |
| `limit` | int | 10 | Maximum messages to return |

### get_agent_events

Get all recent events for an agent (messages, interrupts, status changes).

**Input:**

```json
{
  "agent_id": "claude",
  "since_id": "evt_01ABC123",
  "limit": 20
}
```

**Output:**

```json
{
  "events": [
    {
      "id": "evt_01ABC456",
      "agent_id": "claude",
      "channel_id": "discord:123456789",
      "type": "human_message",
      "role": "human",
      "timestamp": "2024-01-15T10:35:00Z",
      "status": "delivered",
      "payload": {"text": "Thanks!"}
    }
  ],
  "agent_id": "claude"
}
```

**Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `agent_id` | string | env var or "default" | Agent to get events for |
| `since_id` | string | (none) | Only return events after this ID |
| `limit` | int | 20 | Maximum events to return |

### daemon_status

Check if the agentcomms daemon is running.

**Input:**

```json
{}
```

**Output (running):**

```json
{
  "running": true,
  "started_at": "2024-01-15T09:00:00Z",
  "agents": 1,
  "providers": ["discord", "telegram"]
}
```

**Output (not running):**

```json
{
  "running": false
}
```

## Agent ID Resolution

The inbound tools need to know which agent to query. Resolution order:

1. `agent_id` parameter in the tool call
2. `AGENTCOMMS_AGENT_ID` environment variable
3. Default: `"default"`

Set the environment variable in your MCP server configuration:

```json
{
  "mcpServers": {
    "agentcomms": {
      "command": "/path/to/agentcomms",
      "env": {
        "AGENTCOMMS_AGENT_ID": "claude"
      }
    }
  }
}
```

## Usage Patterns

### Polling During Long Tasks

```
1. Start long-running operation
2. Every few minutes, call check_messages
3. If messages found, process them
4. Continue operation
```

### Responding to Messages

```
1. Call check_messages
2. Read message from human
3. Process the request
4. Call send_message to respond
```

### Checking Before Major Decisions

```
1. Before making a significant change
2. Call check_messages to see if human sent updates
3. Incorporate any new information
4. Proceed with decision
```
