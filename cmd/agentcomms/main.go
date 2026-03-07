// Package main is the entry point for the agentcomms MCP server.
//
// agentcomms is a Claude Code MCP plugin that enables voice calls and chat messaging.
// It showcases the plexusone stack:
//   - omnivoice: Voice abstraction layer (TTS, STT, Transport, CallSystem interfaces)
//   - omnichat: Chat messaging abstraction (Discord, Telegram, WhatsApp)
//   - omnivoice-twilio: Twilio implementation of omnivoice interfaces
//   - mcpkit: MCP server runtime with ngrok integration
//
// Usage:
//
//	# Voice calling
//	export AGENTCOMMS_PHONE_ACCOUNT_SID=your_twilio_sid
//	export AGENTCOMMS_PHONE_AUTH_TOKEN=your_twilio_token
//	export AGENTCOMMS_PHONE_NUMBER=+15551234567
//	export AGENTCOMMS_USER_PHONE_NUMBER=+15559876543
//	export NGROK_AUTHTOKEN=your_ngrok_token
//
//	# Chat (optional)
//	export AGENTCOMMS_DISCORD_ENABLED=true
//	export AGENTCOMMS_DISCORD_TOKEN=your_discord_token
//
//	./agentcomms
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpkit "github.com/plexusone/mcpkit/runtime"

	"github.com/plexusone/agentcomms/pkg/chat"
	"github.com/plexusone/agentcomms/pkg/config"
	"github.com/plexusone/agentcomms/pkg/tools"
	"github.com/plexusone/agentcomms/pkg/voice"
)

// logger is the package-level logger.
var logger = slog.Default()

func main() {
	if err := run(); err != nil {
		logger.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Info("shutting down")
		cancel()
	}()

	// Load configuration
	cfg, err := config.LoadFromEnv()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	logger.Info("starting agentcomms MCP server")
	logger.Info("using plexusone stack",
		"components", []string{"omnivoice", "omnichat", "mcpkit"},
	)

	// Log enabled features
	if cfg.VoiceEnabled() {
		logger.Info("voice providers",
			"tts", cfg.TTSProvider,
			"stt", cfg.STTProvider,
		)
	}
	if cfg.ChatEnabled() {
		var chatProviders []string
		if cfg.DiscordEnabled {
			chatProviders = append(chatProviders, "discord")
		}
		if cfg.TelegramEnabled {
			chatProviders = append(chatProviders, "telegram")
		}
		if cfg.WhatsAppEnabled {
			chatProviders = append(chatProviders, "whatsapp")
		}
		logger.Info("chat providers", "providers", chatProviders)
	}

	// Create MCP runtime
	rt := mcpkit.New(&mcp.Implementation{
		Name:    "agentcomms",
		Version: "v0.2.0",
	}, nil)

	// Create voice manager if voice is enabled
	var voiceManager *voice.Manager
	if cfg.VoiceEnabled() {
		voiceManager, err = voice.New(cfg)
		if err != nil {
			return fmt.Errorf("failed to create voice manager: %w", err)
		}
		defer func() { _ = voiceManager.Close() }()
	}

	// Create chat manager if chat is enabled
	var chatManager *chat.Manager
	if cfg.ChatEnabled() {
		chatManager, err = chat.New(cfg, logger)
		if err != nil {
			return fmt.Errorf("failed to create chat manager: %w", err)
		}
		defer func() { _ = chatManager.Close() }()

		// Initialize chat providers
		if err := chatManager.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize chat manager: %w", err)
		}
	}

	// Register MCP tools
	tools.RegisterTools(rt, voiceManager, chatManager)

	// Start HTTP server with ngrok for webhooks (required for voice)
	httpOpts := &mcpkit.HTTPServerOptions{
		Addr: fmt.Sprintf(":%d", cfg.Port),
		Path: "/mcp",
	}

	// Only set up ngrok if voice is enabled (needs webhooks)
	if cfg.VoiceEnabled() && cfg.NgrokAuthToken != "" {
		httpOpts.Ngrok = &mcpkit.NgrokOptions{
			Authtoken: cfg.NgrokAuthToken,
			Domain:    cfg.NgrokDomain,
		}
		httpOpts.OnReady = func(result *mcpkit.HTTPServerResult) {
			logger.Info("MCP server ready",
				"local_url", result.LocalURL,
				"public_url", result.PublicURL,
			)

			// Initialize voice manager with public URL
			if voiceManager != nil {
				if err := voiceManager.Initialize(result.PublicURL); err != nil {
					logger.Warn("failed to initialize voice manager", "error", err)
				}

				// Set up webhook routes for Twilio
				setupTwilioWebhooks(voiceManager, result.PublicURL)
			}
		}
	} else {
		httpOpts.OnReady = func(result *mcpkit.HTTPServerResult) {
			logger.Info("MCP server ready (chat only)",
				"local_url", result.LocalURL,
			)
		}
	}

	// Run the MCP server (blocks until context cancelled)
	_, err = rt.ServeHTTP(ctx, httpOpts)
	if err != nil && ctx.Err() == nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// setupTwilioWebhooks sets up HTTP handlers for Twilio webhooks.
func setupTwilioWebhooks(manager *voice.Manager, publicURL string) {
	twilioTransport := manager.Transport()
	if twilioTransport == nil {
		logger.Warn("transport not available for webhook setup")
		return
	}

	// Handle Twilio Media Streams WebSocket connections
	http.HandleFunc("/media-stream", func(w http.ResponseWriter, r *http.Request) {
		if err := twilioTransport.HandleWebSocket(w, r, "/media-stream"); err != nil {
			logger.Error("WebSocket error", "error", err)
			http.Error(w, "WebSocket error", http.StatusInternalServerError)
		}
	})

	// Handle Twilio voice webhook (for incoming calls)
	http.HandleFunc("/voice", func(w http.ResponseWriter, r *http.Request) {
		// Return TwiML to connect to Media Streams
		w.Header().Set("Content-Type", "application/xml")
		_, _ = fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<Response>
    <Connect>
        <Stream url="%s/media-stream">
            <Parameter name="direction" value="both"/>
        </Stream>
    </Connect>
</Response>`, publicURL)
	})

	// Handle Twilio status callbacks
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		// Limit body and parse status callback (G120)
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		// Log status update (use Form.Get after ParseForm)
		callSID := r.Form.Get("CallSid")
		callSID = strings.ReplaceAll(callSID, "\n", "")
		callSID = strings.ReplaceAll(callSID, "\r", "")
		callStatus := r.Form.Get("CallStatus")
		callStatus = strings.ReplaceAll(callStatus, "\n", "")
		callStatus = strings.ReplaceAll(callStatus, "\r", "")
		logger.Info("call status update", "call_sid", callSID, "status", callStatus)
		w.WriteHeader(http.StatusOK)
	})

	logger.Info("Twilio webhooks configured",
		"voice_url", publicURL+"/voice",
		"stream_url", publicURL+"/media-stream",
		"status_url", publicURL+"/status",
	)
}
