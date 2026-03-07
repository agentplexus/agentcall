// Package main publishes the agentcomms plugin to the Claude Code marketplace.
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
//	GITHUB_TOKEN=ghp_xxx go run ./cmd/publish --title "Add agentcomms voice and chat plugin"
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/plexusone/assistantkit/bundle"
	"github.com/plexusone/assistantkit/hooks/core"
	"github.com/plexusone/assistantkit/publish"
	"github.com/plexusone/assistantkit/publish/claude"
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
		tmpDir, err := os.MkdirTemp("", "agentcomms-publish-*")
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
		fmt.Printf("  Plugin: agentcomms\n")
		fmt.Printf("  Target: anthropics/claude-plugins-official\n")
		fmt.Printf("  Path:   external_plugins/agentcomms/\n")
		if *outputDir != "" {
			fmt.Printf("\nGenerated files kept at: %s\n", pluginDir)
		}
		return
	}

	// Publish
	fmt.Println("\nSubmitting to Claude Code marketplace...")
	opts := publish.PublishOptions{
		PluginDir:  pluginDir,
		PluginName: "agentcomms",
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

// createBundle builds the agentcomms bundle with all components.
// This is duplicated from generate-plugin to keep the commands independent.
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
			"AGENTCOMMS_PHONE_ACCOUNT_SID": "${AGENTCOMMS_PHONE_ACCOUNT_SID}",
			"AGENTCOMMS_PHONE_AUTH_TOKEN":  "${AGENTCOMMS_PHONE_AUTH_TOKEN}",
			"AGENTCOMMS_PHONE_NUMBER":      "${AGENTCOMMS_PHONE_NUMBER}",
			"AGENTCOMMS_USER_PHONE_NUMBER": "${AGENTCOMMS_USER_PHONE_NUMBER}",
			"NGROK_AUTHTOKEN":              "${NGROK_AUTHTOKEN}",
			"AGENTCOMMS_TTS_VOICE":         "${AGENTCOMMS_TTS_VOICE:-Rachel}",
			"AGENTCOMMS_DISCORD_ENABLED":   "${AGENTCOMMS_DISCORD_ENABLED}",
			"AGENTCOMMS_DISCORD_TOKEN":     "${AGENTCOMMS_DISCORD_TOKEN}",
			"AGENTCOMMS_TELEGRAM_ENABLED":  "${AGENTCOMMS_TELEGRAM_ENABLED}",
			"AGENTCOMMS_TELEGRAM_TOKEN":    "${AGENTCOMMS_TELEGRAM_TOKEN}",
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

## Available Tools

- **initiate_call**: Start a new phone call to the user
- **continue_call**: Continue an active call with another message
- **speak_to_user**: Speak without waiting for a response
- **end_call**: End the call with an optional goodbye message

## Best Practices

1. Be conversational and natural
2. Be concise - phone time is valuable
3. Wait for responses after questions
4. Confirm important decisions before ending

## Cost

Calls cost approximately $0.02-0.04 per minute.
`

	skill.AddTrigger("call")
	skill.AddTrigger("phone")
	skill.AddTrigger("voice")

	return skill
}

// createChatSkill creates the chat-messaging skill.
func createChatSkill() *bundle.Skill {
	skill := bundle.NewSkill("chat-messaging", "Chat messaging capability via Discord, Telegram, and WhatsApp")

	skill.Instructions = `# Chat Messaging Skill

Send messages to users via chat platforms (Discord, Telegram, WhatsApp).

## Available Tools

- **send_message**: Send a message to a chat channel
- **list_channels**: List available chat channels
- **get_messages**: Get recent messages from a conversation

## Supported Providers

- **discord**: Discord servers and DMs
- **telegram**: Telegram chats
- **whatsapp**: WhatsApp conversations
`

	skill.AddTrigger("message")
	skill.AddTrigger("chat")
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

Use the ` + "`initiate_call`" + ` tool to place the call.
`

	cmd.AddExample(
		"Call with message",
		"/call I'm blocked on the API design. Can we discuss?",
		"Initiates call and speaks the provided message",
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

Use the ` + "`send_message`" + ` tool to send the message.
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

	cfg.AddHook(
		core.OnStop,
		core.Hook{
			Type: "prompt",
			Prompt: `The user has stopped the current operation. Consider whether to communicate:
- Phone call for urgent matters or complex discussions
- Chat message for status updates or sharing links`,
		},
	)

	return cfg
}

// createVoiceAgent creates the voice calling agent for Kiro CLI.
func createVoiceAgent() *bundle.Agent {
	agent := bundle.NewAgent("voice-caller", "Voice calling agent for phone conversations")
	agent.Instructions = `You are a voice calling agent that can call users on the phone.`
	agent.WithTools("Read", "Write", "Bash")
	return agent
}

// createChatAgent creates the chat messaging agent for Kiro CLI.
func createChatAgent() *bundle.Agent {
	agent := bundle.NewAgent("chat-messenger", "Chat messaging agent for Discord/Telegram/WhatsApp")
	agent.Instructions = `You are a chat messaging agent that can send messages via Discord, Telegram, and WhatsApp.`
	agent.WithTools("Read", "Write", "Bash")
	return agent
}

// writeReadme creates a README.md for the marketplace submission.
func writeReadme(dir string) error {
	readme := `# agentcomms

An MCP plugin that enables voice calls and chat messaging for AI coding assistants. Start a task, walk away. Your phone rings when the AI is done, stuck, or needs a decision. Or get notified via Discord, Telegram, or WhatsApp.

## Features

- **Phone Calls**: Real voice calls to your phone via Twilio
- **Chat Messaging**: Send messages via Discord, Telegram, or WhatsApp
- **Multi-turn Conversations**: Back-and-forth discussions, not just one-way notifications
- **Smart Triggers**: Hooks that suggest calling/messaging when you're stuck or done with work

## Requirements

### Voice (optional)
- Twilio account with phone number
- ngrok account for webhook tunneling

### Chat (optional)
- Discord bot token
- Telegram bot token
- WhatsApp (via whatsmeow)

## Installation

1. Build the agentcomms binary:
   ` + "```bash\n   go build -o agentcomms ./cmd/agentcomms\n   ```" + `

2. Set environment variables:
   ` + "```bash\n   # Voice (optional)\n   export AGENTCOMMS_PHONE_ACCOUNT_SID=ACxxx\n   export AGENTCOMMS_PHONE_AUTH_TOKEN=xxx\n   export AGENTCOMMS_PHONE_NUMBER=+15551234567\n   export AGENTCOMMS_USER_PHONE_NUMBER=+15559876543\n   export NGROK_AUTHTOKEN=xxx\n   \n   # Chat (optional)\n   export AGENTCOMMS_DISCORD_ENABLED=true\n   export AGENTCOMMS_DISCORD_TOKEN=your_bot_token\n   ```" + `

## MCP Tools

### Voice
- **initiate_call**: Start a new call to the user
- **continue_call**: Continue an active call
- **speak_to_user**: Speak without waiting for response
- **end_call**: End the call

### Chat
- **send_message**: Send a message to a chat channel
- **list_channels**: List available chat channels
- **get_messages**: Get recent messages

## Links

- [Repository](https://github.com/plexusone/agentcomms)
- [Documentation](https://github.com/plexusone/agentcomms#readme)

## License

MIT
`
	return os.WriteFile(filepath.Join(dir, "README.md"), []byte(readme), 0600)
}
