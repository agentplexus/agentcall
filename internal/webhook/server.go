// Package webhook provides HTTP webhook handling for Twilio and Telnyx callbacks.
package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // SHA1 is required for Twilio signature validation
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

// SMSMessage represents an incoming SMS message from a webhook.
type SMSMessage struct {
	ID         string
	From       string
	To         string
	Body       string
	Provider   string // "twilio" or "telnyx"
	Timestamp  time.Time
	RawPayload map[string]string
}

// VoiceEvent represents a voice call status event.
type VoiceEvent struct {
	CallID       string
	Status       string // "initiated", "ringing", "answered", "completed", "busy", "no-answer", "failed"
	From         string
	To           string
	Direction    string
	Duration     int // seconds, for completed calls
	RecordingURL string
	Provider     string
	Timestamp    time.Time
	RawPayload   map[string]string
}

// SMSHandler is called when an SMS message is received.
type SMSHandler func(ctx context.Context, msg SMSMessage) error

// VoiceHandler is called when a voice event is received.
type VoiceHandler func(ctx context.Context, event VoiceEvent) error

// maxFormSize limits the size of webhook payloads (64KB is more than enough for Twilio/Telnyx).
const maxFormSize = 64 * 1024

// Server handles incoming webhooks from Twilio and Telnyx.
type Server struct {
	logger *slog.Logger
	mux    *http.ServeMux
	server *http.Server

	// Twilio credentials for signature validation
	twilioAuthToken string

	// Telnyx credentials for signature validation
	telnyxPublicKey string

	// Handlers
	smsHandler   SMSHandler
	voiceHandler VoiceHandler

	mu sync.RWMutex
}

// Config configures the webhook server.
type Config struct {
	// Port is the HTTP server port.
	Port int

	// TwilioAuthToken is used to validate Twilio webhook signatures.
	TwilioAuthToken string

	// TelnyxPublicKey is used to validate Telnyx webhook signatures (optional).
	TelnyxPublicKey string

	// Logger for server logging.
	Logger *slog.Logger
}

// New creates a new webhook server.
func New(cfg Config) *Server {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	s := &Server{
		logger:          cfg.Logger,
		mux:             http.NewServeMux(),
		twilioAuthToken: cfg.TwilioAuthToken,
		telnyxPublicKey: cfg.TelnyxPublicKey,
	}

	// Register routes
	s.mux.HandleFunc("/webhook/twilio/sms", s.handleTwilioSMS)
	s.mux.HandleFunc("/webhook/twilio/voice", s.handleTwilioVoice)
	s.mux.HandleFunc("/webhook/telnyx/sms", s.handleTelnyxSMS)
	s.mux.HandleFunc("/webhook/telnyx/voice", s.handleTelnyxVoice)
	s.mux.HandleFunc("/health", s.handleHealth)

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      s.mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return s
}

// OnSMS registers a handler for incoming SMS messages.
func (s *Server) OnSMS(handler SMSHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.smsHandler = handler
}

// OnVoice registers a handler for voice events.
func (s *Server) OnVoice(handler VoiceHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.voiceHandler = handler
}

// Start starts the webhook server.
func (s *Server) Start() error {
	s.logger.Info("starting webhook server", "addr", s.server.Addr)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// Handler returns the HTTP handler for use with external servers.
func (s *Server) Handler() http.Handler {
	return s.mux
}

// handleHealth is a simple health check endpoint.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// handleTwilioSMS handles incoming SMS from Twilio.
func (s *Server) handleTwilioSMS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit request body size to prevent memory exhaustion
	r.Body = http.MaxBytesReader(w, r.Body, maxFormSize)

	// Parse form data
	if err := r.ParseForm(); err != nil {
		s.logger.Error("failed to parse form", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Validate Twilio signature if auth token is configured
	if s.twilioAuthToken != "" {
		if !s.validateTwilioSignature(r) {
			s.logger.Warn("invalid Twilio signature")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// Extract message data
	msg := SMSMessage{
		ID:         r.FormValue("MessageSid"),
		From:       r.FormValue("From"),
		To:         r.FormValue("To"),
		Body:       r.FormValue("Body"),
		Provider:   "twilio",
		Timestamp:  time.Now(),
		RawPayload: formToMap(r.Form),
	}

	s.logger.Info("received Twilio SMS",
		"from", msg.From,
		"to", msg.To,
		"message_id", msg.ID,
	)

	// Call handler
	s.mu.RLock()
	handler := s.smsHandler
	s.mu.RUnlock()

	if handler != nil {
		if err := handler(r.Context(), msg); err != nil {
			s.logger.Error("SMS handler error", "error", err)
		}
	}

	// Return empty TwiML response
	w.Header().Set("Content-Type", "application/xml")
	_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Response></Response>`))
}

// handleTwilioVoice handles voice status callbacks from Twilio.
func (s *Server) handleTwilioVoice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit request body size to prevent memory exhaustion
	r.Body = http.MaxBytesReader(w, r.Body, maxFormSize)

	if err := r.ParseForm(); err != nil {
		s.logger.Error("failed to parse form", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Validate Twilio signature if auth token is configured
	if s.twilioAuthToken != "" {
		if !s.validateTwilioSignature(r) {
			s.logger.Warn("invalid Twilio signature")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// Extract event data
	event := VoiceEvent{
		CallID:       r.FormValue("CallSid"),
		Status:       r.FormValue("CallStatus"),
		From:         r.FormValue("From"),
		To:           r.FormValue("To"),
		Direction:    r.FormValue("Direction"),
		RecordingURL: r.FormValue("RecordingUrl"),
		Provider:     "twilio",
		Timestamp:    time.Now(),
		RawPayload:   formToMap(r.Form),
	}

	// Parse duration if present (ignore parse errors, non-critical)
	if duration := r.FormValue("CallDuration"); duration != "" {
		_, _ = fmt.Sscanf(duration, "%d", &event.Duration)
	}

	s.logger.Info("received Twilio voice event",
		"call_id", event.CallID,
		"status", event.Status,
		"from", event.From,
		"to", event.To,
	)

	// Call handler
	s.mu.RLock()
	handler := s.voiceHandler
	s.mu.RUnlock()

	if handler != nil {
		if err := handler(r.Context(), event); err != nil {
			s.logger.Error("voice handler error", "error", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handleTelnyxSMS handles incoming SMS from Telnyx.
func (s *Server) handleTelnyxSMS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Error("failed to read body", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Parse JSON payload
	payload, err := parseTelnyxWebhook(body)
	if err != nil {
		s.logger.Error("failed to parse Telnyx webhook", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Only handle message.received events
	if payload.EventType != "message.received" {
		w.WriteHeader(http.StatusOK)
		return
	}

	msg := SMSMessage{
		ID:         payload.ID,
		From:       payload.From,
		To:         payload.To,
		Body:       payload.Text,
		Provider:   "telnyx",
		Timestamp:  time.Now(),
		RawPayload: payload.Raw,
	}

	s.logger.Info("received Telnyx SMS",
		"from", msg.From,
		"to", msg.To,
		"message_id", msg.ID,
	)

	// Call handler
	s.mu.RLock()
	handler := s.smsHandler
	s.mu.RUnlock()

	if handler != nil {
		if err := handler(r.Context(), msg); err != nil {
			s.logger.Error("SMS handler error", "error", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handleTelnyxVoice handles voice events from Telnyx.
func (s *Server) handleTelnyxVoice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Error("failed to read body", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Parse JSON payload
	payload, err := parseTelnyxWebhook(body)
	if err != nil {
		s.logger.Error("failed to parse Telnyx webhook", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Map Telnyx event types to our status
	status := mapTelnyxVoiceEvent(payload.EventType)
	if status == "" {
		// Not a voice event we care about
		w.WriteHeader(http.StatusOK)
		return
	}

	event := VoiceEvent{
		CallID:     payload.CallControlID,
		Status:     status,
		From:       payload.From,
		To:         payload.To,
		Provider:   "telnyx",
		Timestamp:  time.Now(),
		RawPayload: payload.Raw,
	}

	s.logger.Info("received Telnyx voice event",
		"call_id", event.CallID,
		"status", event.Status,
		"event_type", payload.EventType,
	)

	// Call handler
	s.mu.RLock()
	handler := s.voiceHandler
	s.mu.RUnlock()

	if handler != nil {
		if err := handler(r.Context(), event); err != nil {
			s.logger.Error("voice handler error", "error", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// validateTwilioSignature validates the X-Twilio-Signature header.
func (s *Server) validateTwilioSignature(r *http.Request) bool {
	signature := r.Header.Get("X-Twilio-Signature")
	if signature == "" {
		return false
	}

	// Build the URL that Twilio used
	scheme := "https"
	if r.TLS == nil {
		// Check X-Forwarded-Proto for proxied requests
		if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
			scheme = proto
		}
	}
	fullURL := fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL.RequestURI())

	// Get sorted form parameters
	params := make([]string, 0, len(r.Form))
	for k := range r.Form {
		params = append(params, k)
	}
	sort.Strings(params)

	// Build string to sign: URL + sorted params
	var builder strings.Builder
	builder.WriteString(fullURL)
	for _, k := range params {
		builder.WriteString(k)
		builder.WriteString(r.FormValue(k))
	}

	// Compute HMAC-SHA1
	mac := hmac.New(sha1.New, []byte(s.twilioAuthToken))
	mac.Write([]byte(builder.String()))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expected))
}

// formToMap converts url.Values to map[string]string.
func formToMap(form url.Values) map[string]string {
	m := make(map[string]string, len(form))
	for k, v := range form {
		if len(v) > 0 {
			m[k] = v[0]
		}
	}
	return m
}

// mapTelnyxVoiceEvent maps Telnyx event types to our status.
func mapTelnyxVoiceEvent(eventType string) string {
	switch eventType {
	case "call.initiated":
		return "initiated"
	case "call.ringing":
		return "ringing"
	case "call.answered":
		return "answered"
	case "call.hangup":
		return "completed"
	case "call.machine.detection.ended":
		return "machine_detected"
	default:
		return ""
	}
}
