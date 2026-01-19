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

	"github.com/agentplexus/assistantkit/bundle"
	"github.com/agentplexus/assistantkit/hooks/core"
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
	log.Printf("Generating %s integration files to %s...\n", tool, outputDir)

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

// createBundle builds the agentcall bundle with all components.
func createBundle() *bundle.Bundle {
	b := bundle.New("agentcall", "0.1.0", "Voice calling for AI assistants via phone")
	b.Plugin.Author = "agentplexus"
	b.Plugin.License = "MIT"
	b.Plugin.Repository = "https://github.com/agentplexus/agentcall"
	b.Plugin.Homepage = "https://github.com/agentplexus/agentcall"

	// Add MCP server
	b.AddMCPServer("agentcall", bundle.MCPServer{
		Command: "./agentcall",
		Env: map[string]string{
			"AGENTCALL_PHONE_ACCOUNT_SID": "${AGENTCALL_PHONE_ACCOUNT_SID}",
			"AGENTCALL_PHONE_AUTH_TOKEN":  "${AGENTCALL_PHONE_AUTH_TOKEN}",
			"AGENTCALL_PHONE_NUMBER":      "${AGENTCALL_PHONE_NUMBER}",
			"AGENTCALL_USER_PHONE_NUMBER": "${AGENTCALL_USER_PHONE_NUMBER}",
			"NGROK_AUTHTOKEN":             "${NGROK_AUTHTOKEN}",
			"AGENTCALL_TTS_VOICE":         "${AGENTCALL_TTS_VOICE:-Polly.Matthew}",
		},
	})

	// Add dependencies
	b.Plugin.AddOptionalDependency("ngrok", "ngrok")

	// Add skill
	b.AddSkill(createPhoneSkill())

	// Add command
	b.AddCommand(createCallCommand())

	// Add hooks
	b.SetHooks(createHooks())

	// Add agent (for Kiro CLI)
	b.AddAgent(createVoiceAgent())

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

## Examples

**With message:**
` + "```\n/call Hey, I finished the feature! Want me to walk you through it?\n```" + `

**Without message (crafts based on context):**
` + "```\n/call\n```" + `

## Notes

- The call will ring the user's configured phone number
- Calls cost approximately $0.02-0.04 per minute
- Use for meaningful interactions, not simple questions
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

// createHooks creates the hooks configuration.
func createHooks() *bundle.Config {
	cfg := bundle.NewHooksConfig()

	// Add hook for OnStop event
	cfg.AddHook(
		core.OnStop,
		core.Hook{
			Type: "prompt",
			Prompt: `The user has stopped the current operation. Consider whether this is an appropriate time to call them:

- If you completed significant work, offer to call and walk them through it
- If you're blocked and need clarification, suggest calling to discuss
- If it's a minor pause, don't suggest calling

If calling seems appropriate, use the initiate_call tool. Otherwise, continue working or wait for instructions.`,
		},
	)

	// Add hook for notification events
	cfg.AddHook(
		core.OnNotification,
		core.Hook{
			Type: "prompt",
			Prompt: `You received a notification that may indicate you're stuck or need user input.

If you've been working for a while without progress or need a decision that's blocking further work, consider calling the user to discuss. Use the initiate_call tool if appropriate.`,
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

// Re-export types for convenience in this package
type (
	Skill   = bundle.Skill
	Command = bundle.Command
	Agent   = bundle.Agent
	Config  = bundle.Config
)
