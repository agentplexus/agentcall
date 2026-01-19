---
description: Initiate a phone call to the user
---

# Call

Initiate a phone call to the user

## Usage

```
/call [message]
```

## Arguments

- **message**: Initial message to speak when user answers

## Instructions

Initiate a phone call to the user for real-time voice conversation.

## Usage

```
/call [message]
```

## Arguments

- **message** (optional): The initial message to speak when the user answers. If not provided, you should craft an appropriate greeting based on context.

## Behavior

When this command is invoked:

1. Check if there's context that warrants a call (completed work, blocking issue, decision needed)
2. If no message provided, craft an appropriate opening based on recent conversation
3. Use the `initiate_call` tool to place the call
4. Wait for the user's response
5. Continue the conversation as needed using `continue_call`
6. End the call politely using `end_call` when done

## Examples

**With message:**
```
/call Hey, I finished the feature! Want me to walk you through it?
```

**Without message (Claude crafts based on context):**
```
/call
```

## Notes

- The call will ring the user's configured phone number
- Calls cost approximately $0.02-0.04 per minute
- Use for meaningful interactions, not simple questions

