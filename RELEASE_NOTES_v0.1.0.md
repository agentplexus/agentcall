# Release Notes v0.1.0

**Release Date:** 2026-01-19

## Why Should I Care About This Release?

**Your AI coding assistant can now call you on the phone.** Start a long-running task, walk away, and get a phone call when the AI is done, stuck, or needs a decision. No more staring at a terminal waiting for completion.

### Key Highlights

- **Phone calls from your AI assistant** - Claude Code, AWS Kiro CLI, and Gemini CLI can initiate real phone calls to discuss complex decisions, report completion, or ask clarifying questions

- **Premium voice quality** - Natural conversations using ElevenLabs streaming TTS and Deepgram streaming STT, not robotic text-to-speech

- **Single binary deployment** - 53 MB self-contained Go binary with no runtime dependencies. Copy one file and you're done

- **Provider-agnostic architecture** - Built on the omnivoice abstraction layer, making it easy to swap TTS/STT/phone providers

## What's New

### MCP Voice Call Plugin

The core plugin provides four MCP tools:

| Tool | Purpose |
|------|---------|
| `initiate_call` | Start a new call to the user with an initial message |
| `continue_call` | Continue conversation on an active call |
| `speak_to_user` | Speak without waiting for response (status updates) |
| `end_call` | End the call with optional goodbye message |

### Multi-Assistant Support

Generate configuration files for your preferred AI coding tool:

```bash
go run ./cmd/generate-plugin claude .   # Claude Code
go run ./cmd/generate-plugin kiro .     # AWS Kiro CLI
go run ./cmd/generate-plugin gemini .   # Gemini CLI
```

### The agentplexus Stack

This release showcases the complete agentplexus voice AI architecture:

- **omnivoice** - Provider-agnostic interfaces for TTS, STT, Transport, CallSystem
- **go-elevenlabs** - ElevenLabs streaming TTS with natural voices
- **omnivoice-deepgram** - Deepgram streaming STT with accurate transcription
- **omnivoice-twilio** - Twilio transport and call system
- **mcpkit** - MCP server runtime with built-in ngrok integration

## Use Cases

**Ideal for:**

- Reporting significant task completion
- Requesting clarification when blocked
- Discussing complex architectural decisions
- Walking through code changes verbally
- Multi-step processes needing back-and-forth

## Cost Estimate

| Service | Cost |
|---------|------|
| Twilio outbound calls | ~$0.014/min |
| ElevenLabs TTS | ~$0.03/min of speech |
| Deepgram STT | ~$0.0043/min |
| **Total per minute** | ~$0.05/min |

## Getting Started

See the [README](README.md) for installation and configuration instructions.

## Credits

Inspired by [ZeframLou/call-me](https://github.com/ZeframLou/call-me) (TypeScript).
