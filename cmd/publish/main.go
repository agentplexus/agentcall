// Package main publishes the agentcall plugin to the Claude Code marketplace.
//
// This tool generates the plugin files and submits a PR to anthropics/claude-plugins-official.
//
// Usage:
//
//	# Validate locally without creating a PR (dry run)
//	go run ./cmd/publish --dry-run
//
//	# Validate and show what would be submitted
//	go run ./cmd/publish --dry-run --verbose
//
//	# Submit to marketplace (requires GITHUB_TOKEN)
//	GITHUB_TOKEN=ghp_xxx go run ./cmd/publish
//
//	# Submit with custom PR title
//	GITHUB_TOKEN=ghp_xxx go run ./cmd/publish --title "Add agentcall voice calling plugin"
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/agentplexus/assistantkit/bundle"
	"github.com/agentplexus/assistantkit/hooks/core"
	"github.com/agentplexus/assistantkit/publish"
	"github.com/agentplexus/assistantkit/publish/claude"
)

func main() {
	// Parse flags
	dryRun := flag.Bool("dry-run", false, "Validate without creating PR")
	verbose := flag.Bool("verbose", false, "Show detailed output")
	title := flag.String("title", "", "Custom PR title")
	body := flag.String("body", "", "Custom PR body")
	outputDir := flag.String("output", "", "Keep generated files in this directory (otherwise uses temp)")
	flag.Parse()

	// Get GitHub token from environment
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" && !*dryRun {
		log.Fatal("GITHUB_TOKEN environment variable required (or use --dry-run)")
	}

	// Create temp directory for generated files (or use specified output)
	var pluginDir string
	var cleanup func()

	if *outputDir != "" {
		pluginDir = *outputDir
		cleanup = func() {} // No cleanup needed
		if *verbose {
			fmt.Printf("Using output directory: %s\n", pluginDir)
		}
	} else {
		tmpDir, err := os.MkdirTemp("", "agentcall-publish-*")
		if err != nil {
			log.Fatalf("Failed to create temp directory: %v", err)
		}
		pluginDir = tmpDir
		cleanup = func() { _ = os.RemoveAll(tmpDir) }
		if *verbose {
			fmt.Printf("Using temp directory: %s\n", pluginDir)
		}
	}
	defer cleanup()

	// Generate plugin files
	fmt.Println("Generating plugin files...")
	b := createBundle()
	if err := b.Generate("claude", pluginDir); err != nil {
		log.Fatalf("Failed to generate plugin: %v", err)
	}

	// Add README.md (required for marketplace)
	if err := writeReadme(pluginDir); err != nil {
		log.Fatalf("Failed to write README: %v", err)
	}

	// List generated files
	if *verbose {
		fmt.Println("\nGenerated files:")
		err := filepath.Walk(pluginDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				relPath, _ := filepath.Rel(pluginDir, path)
				fmt.Printf("  %s\n", relPath)
			}
			return nil
		})
		if err != nil {
			log.Printf("Warning: could not list files: %v", err)
		}
	}

	// Create publisher
	publisher := claude.NewPublisher(token)

	// Validate
	fmt.Println("\nValidating plugin...")
	if err := publisher.Validate(pluginDir); err != nil {
		log.Fatalf("Validation failed: %v", err)
	}
	fmt.Println("Validation passed!")

	if *dryRun {
		fmt.Println("\n[Dry run] Would submit to Claude Code marketplace:")
		fmt.Printf("  Plugin: agentcall\n")
		fmt.Printf("  Target: anthropics/claude-plugins-official\n")
		fmt.Printf("  Path:   external_plugins/agentcall/\n")
		if *outputDir != "" {
			fmt.Printf("\nGenerated files kept at: %s\n", pluginDir)
		}
		return
	}

	// Publish
	fmt.Println("\nSubmitting to Claude Code marketplace...")
	opts := publish.PublishOptions{
		PluginDir:  pluginDir,
		PluginName: "agentcall",
		DryRun:     false,
		Verbose:    *verbose,
		Title:      *title,
		Body:       *body,
	}

	result, err := publisher.Publish(context.Background(), opts)
	if err != nil {
		log.Fatalf("Publish failed: %v", err)
	}

	fmt.Printf("\nSuccess! %s\n", result.Status)
	fmt.Printf("PR URL: %s\n", result.PRURL)
	fmt.Printf("Fork:   %s\n", result.ForkURL)
	fmt.Printf("Branch: %s\n", result.Branch)
}

// createBundle builds the agentcall bundle with all components.
// This is duplicated from generate-plugin to keep the commands independent.
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

// writeReadme creates a README.md for the marketplace submission.
func writeReadme(dir string) error {
	readme := `# agentcall

An MCP plugin that enables voice calls via phone for AI coding assistants. Start a task, walk away. Your phone rings when the AI is done, stuck, or needs a decision.

## Features

- **Phone Calls**: Real voice calls to your phone via Twilio
- **Multi-turn Conversations**: Back-and-forth discussions, not just one-way notifications
- **Smart Triggers**: Hooks that suggest calling when you're stuck or done with work
- **Cost Effective**: ~$0.02-0.04 per minute

## Requirements

- Twilio account with phone number
- ngrok account for webhook tunneling

## Installation

1. Build the agentcall binary:
   ` + "```bash\n   go build -o agentcall ./cmd/agentcall\n   ```" + `

2. Set environment variables:
   ` + "```bash\n   export AGENTCALL_PHONE_ACCOUNT_SID=ACxxx\n   export AGENTCALL_PHONE_AUTH_TOKEN=xxx\n   export AGENTCALL_PHONE_NUMBER=+15551234567\n   export AGENTCALL_USER_PHONE_NUMBER=+15559876543\n   export NGROK_AUTHTOKEN=xxx\n   ```" + `

## MCP Tools

- **initiate_call**: Start a new call to the user
- **continue_call**: Continue an active call
- **speak_to_user**: Speak without waiting for response
- **end_call**: End the call

## Links

- [Repository](https://github.com/agentplexus/agentcall)
- [Documentation](https://github.com/agentplexus/agentcall#readme)

## License

MIT
`
	return os.WriteFile(filepath.Join(dir, "README.md"), []byte(readme), 0600)
}
