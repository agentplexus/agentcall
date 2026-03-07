// Package main generates AI assistant integration files using assistantkit bundle.
//
// This tool generates configuration files for multiple AI assistant tools:
//
//   - Claude Code: .claude-plugin/, skills/, commands/, .claude/
//   - Kiro CLI: .kiro/agents/, .kiro/settings/
//   - Gemini CLI: gemini-extension.json, agents/
//
// Usage:
//
//	# Generate for Claude Code (default)
//	go run ./cmd/generate-plugin
//
//	# Generate for a specific tool
//	go run ./cmd/generate-plugin claude
//	go run ./cmd/generate-plugin kiro
//	go run ./cmd/generate-plugin gemini
//
//	# Generate for all tools
//	go run ./cmd/generate-plugin all
//
//	# Generate to a specific directory
//	go run ./cmd/generate-plugin claude ./output
package main

import (
	"log"
	"os"

	"github.com/plexusone/assistantkit/bundle"
	"github.com/plexusone/assistantkit/hooks/core"
)

func main() {
	// Parse arguments
	tool := "claude"
	outputDir := "."

	if len(os.Args) > 1 {
		tool = os.Args[1]
	}
	if len(os.Args) > 2 {
		outputDir = os.Args[2]
	}

	// Create the bundle
	b := createBundle()

	// Generate
	log.Printf("Generating %s integration files to %s...\n", tool, outputDir) //nolint:gosec // G706: Values from validated CLI args

	var err error
	if tool == "all" {
		err = b.GenerateAll(outputDir)
	} else {
		err = b.Generate(tool, outputDir)
	}

	if err != nil {
		log.Fatalf("Failed to generate: %v", err)
	}

	log.Printf("Integration files generated successfully!")
}

// createBundle builds the agentcomms bundle with all components.
func createBundle() *bundle.Bundle {
	b := bundle.New("agentcomms", "0.2.0", "Voice calling and chat messaging for AI assistants")
	b.Plugin.Author = "plexusone"
	b.Plugin.License = "MIT"
	b.Plugin.Repository = "https://github.com/plexusone/agentcomms"
	b.Plugin.Homepage = "https://github.com/plexusone/agentcomms"

	// Add MCP server
	//nolint:gosec // G101: These are env var templates, not credentials
	b.AddMCPServer("agentcomms", bundle.MCPServer{
		Command: "./agentcomms",
		Env: map[string]string{
			// Voice (optional)
			"AGENTCOMMS_PHONE_ACCOUNT_SID": "${AGENTCOMMS_PHONE_ACCOUNT_SID}",
			"AGENTCOMMS_PHONE_AUTH_TOKEN":  "${AGENTCOMMS_PHONE_AUTH_TOKEN}",
			"AGENTCOMMS_PHONE_NUMBER":      "${AGENTCOMMS_PHONE_NUMBER}",
			"AGENTCOMMS_USER_PHONE_NUMBER": "${AGENTCOMMS_USER_PHONE_NUMBER}",
			"NGROK_AUTHTOKEN":              "${NGROK_AUTHTOKEN}",
			"AGENTCOMMS_TTS_VOICE":         "${AGENTCOMMS_TTS_VOICE:-Rachel}",
			// Chat (optional)
			"AGENTCOMMS_DISCORD_ENABLED":  "${AGENTCOMMS_DISCORD_ENABLED}",
			"AGENTCOMMS_DISCORD_TOKEN":    "${AGENTCOMMS_DISCORD_TOKEN}",
			"AGENTCOMMS_TELEGRAM_ENABLED": "${AGENTCOMMS_TELEGRAM_ENABLED}",
			"AGENTCOMMS_TELEGRAM_TOKEN":   "${AGENTCOMMS_TELEGRAM_TOKEN}",
		},
	})

	// Add dependencies
	b.Plugin.AddOptionalDependency("ngrok", "ngrok")

	// Add skills
	b.AddSkill(createPhoneSkill())
	b.AddSkill(createChatSkill())

	// Add commands
	b.AddCommand(createCallCommand())
	b.AddCommand(createMessageCommand())

	// Add hooks
	b.SetHooks(createHooks())

	// Add agents (for Kiro CLI)
	b.AddAgent(createVoiceAgent())
	b.AddAgent(createChatAgent())

	return b
}

// createPhoneSkill creates the phone-input skill.
func createPhoneSkill() *bundle.Skill {
	skill := bundle.NewSkill("phone-input", "Voice calling capability for multi-turn phone conversations")

	skill.Instructions = `# Phone Input Skill

This skill enables AI assistants to call the user on the phone for real-time voice conversations.

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
` + "```json\n{\n  \"message\": \"Hey! I finished implementing the authentication system. Want me to walk you through what I built?\"\n}\n```" + `

### continue_call
Continue an active call with another message. Use for multi-turn conversations.

**Example:**
` + "```json\n{\n  \"call_id\": \"call-1-123456\",\n  \"message\": \"Should I also add refresh token support, or is the basic JWT implementation sufficient?\"\n}\n```" + `

### speak_to_user
Speak without waiting for a response. Use for acknowledgments before time-consuming operations.

**Example:**
` + "```json\n{\n  \"call_id\": \"call-1-123456\",\n  \"message\": \"Let me search through the codebase for that. Give me a moment...\"\n}\n```" + `

### end_call
End the call with an optional goodbye message.

**Example:**
` + "```json\n{\n  \"call_id\": \"call-1-123456\",\n  \"message\": \"Perfect! I'll get started on the tests. Talk soon!\"\n}\n```" + `

## Best Practices

1. **Be conversational**: Speak naturally, as if talking to a colleague
2. **Be concise**: Phone time is valuable; get to the point
3. **Wait for response**: After asking a question, always wait for the user's answer
4. **Handle silence**: If the user doesn't respond, ask if they're still there
5. **Confirm understanding**: Repeat back important decisions before ending the call

## Cost Consideration

Phone calls cost approximately $0.02-0.04 per minute. Use them judiciously for high-value interactions.
`

	skill.AddTrigger("call")
	skill.AddTrigger("phone")
	skill.AddTrigger("voice")
	skill.AddTrigger("ring")

	return skill
}

// createChatSkill creates the chat-messaging skill.
func createChatSkill() *bundle.Skill {
	skill := bundle.NewSkill("chat-messaging", "Chat messaging capability via Discord, Telegram, and WhatsApp")

	skill.Instructions = `# Chat Messaging Skill

This skill enables AI assistants to send messages to users via chat platforms (Discord, Telegram, WhatsApp).

## When to Use

Use chat messaging when:

- **Asynchronous Updates**: You need to notify the user but don't need an immediate response
- **Share Links/Code**: You have URLs, code snippets, or formatted content to share
- **Non-urgent Communication**: The matter can wait for the user to check their messages
- **Follow-up**: You want to send a summary after completing work

## Available Tools

### send_message
Send a message to a chat channel.

**Example:**
` + "```json\n{\n  \"provider\": \"discord\",\n  \"chat_id\": \"123456789\",\n  \"message\": \"I've finished the PR! Here's the link: https://github.com/...\"\n}\n```" + `

### list_channels
List available chat channels and their status.

### get_messages
Get recent messages from a chat conversation.

**Example:**
` + "```json\n{\n  \"provider\": \"telegram\",\n  \"chat_id\": \"987654321\",\n  \"limit\": 5\n}\n```" + `

## Supported Providers

- **discord**: Discord servers and DMs
- **telegram**: Telegram chats
- **whatsapp**: WhatsApp conversations

## Best Practices

1. **Choose the right channel**: Use the platform the user prefers
2. **Keep messages focused**: One topic per message
3. **Include context**: Reference what task the message relates to
4. **Use formatting**: Markdown is supported on most platforms
`

	skill.AddTrigger("message")
	skill.AddTrigger("chat")
	skill.AddTrigger("notify")
	skill.AddTrigger("discord")
	skill.AddTrigger("telegram")

	return skill
}

// createCallCommand creates the /call command.
func createCallCommand() *bundle.Command {
	cmd := bundle.NewCommand("call", "Initiate a phone call to the user")

	cmd.Arguments = []bundle.Argument{
		{
			Name:        "message",
			Type:        "string",
			Required:    false,
			Description: "Initial message to speak when user answers",
		},
	}

	cmd.Instructions = `Initiate a phone call to the user for real-time voice conversation.

## Usage

` + "```\n/call [message]\n```" + `

## Arguments

- **message** (optional): The initial message to speak when the user answers. If not provided, craft an appropriate greeting based on context.

## Behavior

When this command is invoked:

1. Check if there's context that warrants a call (completed work, blocking issue, decision needed)
2. If no message provided, craft an appropriate opening based on recent conversation
3. Use the ` + "`initiate_call`" + ` tool to place the call
4. Wait for the user's response
5. Continue the conversation as needed using ` + "`continue_call`" + `
6. End the call politely using ` + "`end_call`" + ` when done
`

	cmd.AddExample(
		"Call with specific message",
		"/call I'm blocked on the API design. Can we discuss the endpoint structure?",
		"Initiates call and speaks the provided message",
	)
	cmd.AddExample(
		"Call without message",
		"/call",
		"Crafts an appropriate message based on conversation context",
	)

	return cmd
}

// createMessageCommand creates the /message command.
func createMessageCommand() *bundle.Command {
	cmd := bundle.NewCommand("message", "Send a message via chat (Discord/Telegram/WhatsApp)")

	cmd.Arguments = []bundle.Argument{
		{
			Name:        "provider",
			Type:        "string",
			Required:    true,
			Description: "Chat provider: discord, telegram, or whatsapp",
		},
		{
			Name:        "content",
			Type:        "string",
			Required:    true,
			Description: "Message content to send",
		},
	}

	cmd.Instructions = `Send a message to the user via a chat platform.

## Usage

` + "```\n/message <provider> <content>\n```" + `

## Arguments

- **provider**: The chat platform to use (discord, telegram, whatsapp)
- **content**: The message to send

## Behavior

1. Look up the user's chat ID for the specified provider
2. Use the ` + "`send_message`" + ` tool to send the message
3. Confirm message was sent
`

	cmd.AddExample(
		"Send Discord message",
		"/message discord The deployment is complete!",
		"Sends message to user's Discord",
	)

	return cmd
}

// createHooks creates the hooks configuration.
func createHooks() *bundle.Config {
	cfg := bundle.NewHooksConfig()

	// Add hook for OnStop event
	cfg.AddHook(
		core.OnStop,
		core.Hook{
			Type: "prompt",
			Prompt: `The user has stopped the current operation. Consider whether this is an appropriate time to communicate:

- If you completed significant work, offer to call or message them
- If you're blocked and need clarification, suggest calling to discuss
- If it's a minor pause, don't suggest contacting them

Choose the appropriate channel:
- **Phone call**: For urgent matters or complex discussions
- **Chat message**: For status updates or sharing links/code

If communication seems appropriate, use the relevant tool. Otherwise, continue working or wait for instructions.`,
		},
	)

	// Add hook for notification events
	cfg.AddHook(
		core.OnNotification,
		core.Hook{
			Type: "prompt",
			Prompt: `You received a notification that may indicate you're stuck or need user input.

If you've been working for a while without progress or need a decision that's blocking further work, consider:
- **Phone call**: For urgent decisions or complex discussions
- **Chat message**: For non-urgent updates or questions

Use the appropriate tool if communication is warranted.`,
		},
	)

	return cfg
}

// createVoiceAgent creates the voice calling agent for Kiro CLI.
func createVoiceAgent() *bundle.Agent {
	agent := bundle.NewAgent("voice-caller", "Voice calling agent for phone conversations")

	agent.Instructions = `You are a voice calling agent that can call users on the phone.

## Capabilities

You can use the following MCP tools:

- **initiate_call**: Start a new phone call to the user
- **continue_call**: Continue an active call with another message
- **speak_to_user**: Speak without waiting for a response
- **end_call**: End the call with an optional goodbye message

## When to Call

Call the user when:
- You've completed significant work and want to walk them through it
- You're blocked and need clarification
- A complex decision requires discussion
- The task needs iterative input

## Guidelines

1. Be conversational and natural
2. Be concise - phone time is valuable
3. Wait for responses after questions
4. Confirm important decisions before ending
5. End calls politely

## Cost

Calls cost approximately $0.02-0.04 per minute.
`

	agent.WithTools("Read", "Write", "Bash")

	return agent
}

// createChatAgent creates the chat messaging agent for Kiro CLI.
func createChatAgent() *bundle.Agent {
	agent := bundle.NewAgent("chat-messenger", "Chat messaging agent for Discord/Telegram/WhatsApp")

	agent.Instructions = `You are a chat messaging agent that can send messages via Discord, Telegram, and WhatsApp.

## Capabilities

You can use the following MCP tools:

- **send_message**: Send a message to a chat channel
- **list_channels**: List available chat channels
- **get_messages**: Get recent messages from a conversation

## When to Message

Send messages when:
- You want to share links, code, or formatted content
- The update is not urgent and can wait
- You're providing a summary or status update
- The user prefers asynchronous communication

## Guidelines

1. Choose the right platform for the content
2. Keep messages focused and clear
3. Use markdown formatting when appropriate
4. Include relevant context
`

	agent.WithTools("Read", "Write", "Bash")

	return agent
}

// Re-export types for convenience in this package.
type (
	Skill   = bundle.Skill
	Command = bundle.Command
	Agent   = bundle.Agent
	Config  = bundle.Config
)
