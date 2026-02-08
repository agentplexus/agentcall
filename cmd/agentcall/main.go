// Package main is the entry point for the agentcall MCP server.
//
// agentcall is a Claude Code MCP plugin that enables voice calls via phone.
// It showcases the agentplexus stack:
//   - omnivoice: Voice abstraction layer (TTS, STT, Transport, CallSystem interfaces)
//   - omnivoice-twilio: Twilio implementation of omnivoice interfaces
//   - mcpkit: MCP server runtime with ngrok integration
//
// Usage:
//
//	export AGENTCALL_PHONE_ACCOUNT_SID=your_twilio_sid
//	export AGENTCALL_PHONE_AUTH_TOKEN=your_twilio_token
//	export AGENTCALL_PHONE_NUMBER=+15551234567
//	export AGENTCALL_USER_PHONE_NUMBER=+15559876543
//	export NGROK_AUTHTOKEN=your_ngrok_token
//	./agentcall
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

	mcpkit "github.com/agentplexus/mcpkit/runtime"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/agentplexus/agentcall/pkg/callmanager"
	"github.com/agentplexus/agentcall/pkg/config"
	"github.com/agentplexus/agentcall/pkg/tools"
	twiliotransport "github.com/agentplexus/omnivoice-twilio/transport"
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

	logger.Info("starting agentcall MCP server")
	logger.Info("using agentplexus stack",
		"components", []string{"omnivoice", "omnivoice-twilio", "mcpkit"},
	)
	logger.Info("voice providers",
		"tts", cfg.TTSProvider,
		"stt", cfg.STTProvider,
	)

	// Create MCP runtime
	rt := mcpkit.New(&mcp.Implementation{
		Name:    "agentcall",
		Version: "v0.1.0",
	}, nil)

	// Create call manager
	manager, err := callmanager.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create call manager: %w", err)
	}
	defer func() { _ = manager.Close() }()

	// Register MCP tools
	tools.RegisterTools(rt, manager)

	// Start HTTP server with ngrok for webhooks
	httpOpts := &mcpkit.HTTPServerOptions{
		Addr: fmt.Sprintf(":%d", cfg.Port),
		Path: "/mcp",
		Ngrok: &mcpkit.NgrokOptions{
			Authtoken: cfg.NgrokAuthToken,
			Domain:    cfg.NgrokDomain,
		},
		OnReady: func(result *mcpkit.HTTPServerResult) {
			logger.Info("MCP server ready",
				"local_url", result.LocalURL,
				"public_url", result.PublicURL,
			)

			// Initialize call manager with public URL
			if err := manager.Initialize(result.PublicURL); err != nil {
				logger.Warn("failed to initialize call manager", "error", err)
			}

			// Set up webhook routes for Twilio
			setupTwilioWebhooks(manager, result.PublicURL)
		},
	}

	// Run the MCP server (blocks until context cancelled)
	_, err = rt.ServeHTTP(ctx, httpOpts)
	if err != nil && ctx.Err() == nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// setupTwilioWebhooks sets up HTTP handlers for Twilio webhooks.
func setupTwilioWebhooks(manager *callmanager.Manager, publicURL string) {
	transport := manager.Transport()
	if transport == nil {
		logger.Warn("transport not available for webhook setup")
		return
	}

	twilioTransport, ok := transport.(*twiliotransport.Provider)
	if !ok {
		logger.Warn("transport is not Twilio transport")
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
		// Parse status callback
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		// Log status update
		callSID := r.FormValue("CallSid")
		callSID = strings.ReplaceAll(callSID, "\n", "")
		callSID = strings.ReplaceAll(callSID, "\r", "")
		callStatus := r.FormValue("CallStatus")
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
