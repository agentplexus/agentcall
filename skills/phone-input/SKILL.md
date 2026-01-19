---
name: phone-input
description: Voice calling capability for multi-turn phone conversations
triggers: [call, phone, voice, ring]
---

# Phone Input

Voice calling capability for multi-turn phone conversations

## Instructions

# Phone Input Skill

This skill enables Claude to call the user on the phone for real-time voice conversations.

## When to Use

Use phone calling when:

- **Task Completion**: You've finished significant work and need to discuss next steps
- **Blocked**: You're stuck and need urgent clarification that would take too long via text
- **Complex Decisions**: The situation requires back-and-forth discussion
- **Milestone Reached**: You want to walk through completed work verbally
- **Multi-step Process**: The task needs iterative input from the user

## When NOT to Use

Don't use phone calling for:

- Simple yes/no questions (use text instead)
- Status updates that don't require discussion
- Information already provided in the conversation
- Quick clarifications that can be typed

## Available Tools

### initiate_call
Start a new call to the user. Use when beginning a conversation.

**Example:**
```json
{
  "message": "Hey! I finished implementing the authentication system. Want me to walk you through what I built?"
}
```

### continue_call
Continue an active call with another message. Use for multi-turn conversations.

**Example:**
```json
{
  "call_id": "call-1-123456",
  "message": "Should I also add refresh token support, or is the basic JWT implementation sufficient?"
}
```

### speak_to_user
Speak without waiting for a response. Use for acknowledgments before time-consuming operations.

**Example:**
```json
{
  "call_id": "call-1-123456",
  "message": "Let me search through the codebase for that. Give me a moment..."
}
```

### end_call
End the call with an optional goodbye message.

**Example:**
```json
{
  "call_id": "call-1-123456",
  "message": "Perfect! I'll get started on the tests. Talk soon!"
}
```

## Best Practices

1. **Be conversational**: Speak naturally, as if talking to a colleague
2. **Be concise**: Phone time is valuable; get to the point
3. **Wait for response**: After asking a question, always wait for the user's answer
4. **Handle silence**: If the user doesn't respond, ask if they're still there
5. **Confirm understanding**: Repeat back important decisions before ending the call

## Cost Consideration

Phone calls cost approximately $0.02-0.04 per minute. Use them judiciously for high-value interactions.


