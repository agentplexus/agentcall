// Package callmanager orchestrates voice calls using the omnivoice stack.
package callmanager

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	elevenlabs "github.com/agentplexus/go-elevenlabs"
	elevenlabstts "github.com/agentplexus/go-elevenlabs/omnivoice/tts"
	deepgramstt "github.com/agentplexus/omnivoice-deepgram/omnivoice/stt"
	twilliocallsystem "github.com/agentplexus/omnivoice-twilio/callsystem"
	"github.com/agentplexus/omnivoice/callsystem"
	"github.com/agentplexus/omnivoice/stt"
	"github.com/agentplexus/omnivoice/tts"

	"github.com/agentplexus/agentcall/pkg/config"
)

// CallState represents the state of an active call.
type CallState struct {
	ID              string
	Call            callsystem.Call
	StartTime       time.Time
	Conversation    []ConversationTurn
	LastUserMessage string
	mu              sync.RWMutex
}

// ConversationTurn represents a single turn in the conversation.
type ConversationTurn struct {
	Role      string // "assistant" or "user"
	Content   string
	Timestamp time.Time
}

// AddTurn adds a conversation turn.
func (cs *CallState) AddTurn(role, content string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.Conversation = append(cs.Conversation, ConversationTurn{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
	if role == "user" {
		cs.LastUserMessage = content
	}
}

// Duration returns the call duration.
func (cs *CallState) Duration() time.Duration {
	return time.Since(cs.StartTime)
}

// Manager orchestrates voice calls using the omnivoice stack.
type Manager struct {
	config *config.Config

	// omnivoice providers
	callSystem  callsystem.CallSystem
	ttsProvider tts.StreamingProvider
	sttProvider stt.StreamingProvider

	// Active calls
	calls   map[string]*CallState
	callsMu sync.RWMutex

	// Call counter for generating IDs
	callCounter int
	counterMu   sync.Mutex

	// Public URL for webhooks (set after ngrok starts)
	publicURL string
}

// New creates a new call manager.
func New(cfg *config.Config) (*Manager, error) {
	m := &Manager{
		config: cfg,
		calls:  make(map[string]*CallState),
	}

	return m, nil
}

// Initialize sets up the omnivoice providers.
// Call this after ngrok is started and publicURL is known.
func (m *Manager) Initialize(publicURL string) error {
	m.publicURL = publicURL

	// Create Twilio CallSystem provider
	cs, err := twilliocallsystem.New(
		twilliocallsystem.WithAccountSID(m.config.PhoneAccountSID),
		twilliocallsystem.WithAuthToken(m.config.PhoneAuthToken),
		twilliocallsystem.WithPhoneNumber(m.config.PhoneNumber),
		twilliocallsystem.WithWebhookURL(publicURL+"/media-stream"),
	)
	if err != nil {
		return fmt.Errorf("failed to create callsystem: %w", err)
	}
	m.callSystem = cs

	// Create ElevenLabs TTS provider
	elevenClient, err := elevenlabs.NewClient(elevenlabs.WithAPIKey(m.config.ElevenLabsAPIKey))
	if err != nil {
		return fmt.Errorf("failed to create ElevenLabs client: %w", err)
	}
	m.ttsProvider = elevenlabstts.NewWithClient(elevenClient)

	// Create Deepgram STT provider
	sttProvider, err := deepgramstt.New(deepgramstt.WithAPIKey(m.config.DeepgramAPIKey))
	if err != nil {
		return fmt.Errorf("failed to create Deepgram provider: %w", err)
	}
	m.sttProvider = sttProvider

	return nil
}

// generateCallID generates a unique call ID.
func (m *Manager) generateCallID() string {
	m.counterMu.Lock()
	defer m.counterMu.Unlock()
	m.callCounter++
	return fmt.Sprintf("call-%d-%d", m.callCounter, time.Now().Unix())
}

// InitiateCall starts a new call to the user and speaks a message.
func (m *Manager) InitiateCall(ctx context.Context, message string) (*CallState, string, error) {
	if m.callSystem == nil {
		return nil, "", fmt.Errorf("call manager not initialized; call Initialize() first")
	}

	// Make the call
	call, err := m.callSystem.MakeCall(ctx, m.config.UserPhoneNumber)
	if err != nil {
		return nil, "", fmt.Errorf("failed to make call: %w", err)
	}

	// Create call state
	callID := m.generateCallID()
	state := &CallState{
		ID:        callID,
		Call:      call,
		StartTime: time.Now(),
	}

	// Store call state
	m.callsMu.Lock()
	m.calls[callID] = state
	m.callsMu.Unlock()

	// Wait for call to be answered (with timeout)
	answered := m.waitForAnswer(ctx, call, 30*time.Second)
	if !answered {
		_ = call.Hangup(ctx)
		m.removeCall(callID)
		return nil, "", fmt.Errorf("call not answered")
	}

	// Speak the initial message
	response, err := m.speakAndListen(ctx, state, message)
	if err != nil {
		return state, "", fmt.Errorf("failed to speak: %w", err)
	}

	return state, response, nil
}

// ContinueCall continues an existing call with a new message.
func (m *Manager) ContinueCall(ctx context.Context, callID, message string) (string, error) {
	state := m.getCall(callID)
	if state == nil {
		return "", fmt.Errorf("call not found: %s", callID)
	}

	response, err := m.speakAndListen(ctx, state, message)
	if err != nil {
		return "", fmt.Errorf("failed to continue call: %w", err)
	}

	return response, nil
}

// SpeakToUser speaks to the user without waiting for a response.
func (m *Manager) SpeakToUser(ctx context.Context, callID, message string) error {
	state := m.getCall(callID)
	if state == nil {
		return fmt.Errorf("call not found: %s", callID)
	}

	if err := m.speak(ctx, state, message); err != nil {
		return fmt.Errorf("failed to speak: %w", err)
	}

	return nil
}

// EndCall ends an existing call with a final message.
func (m *Manager) EndCall(ctx context.Context, callID, message string) (time.Duration, error) {
	state := m.getCall(callID)
	if state == nil {
		return 0, fmt.Errorf("call not found: %s", callID)
	}

	// Speak final message
	if message != "" {
		// Best effort - ignore errors and continue with hangup
		_ = m.speak(ctx, state, message)
		// Wait for audio to play
		time.Sleep(2 * time.Second)
	}

	duration := state.Duration()

	// Hangup
	if err := state.Call.Hangup(ctx); err != nil {
		return duration, fmt.Errorf("failed to hangup: %w", err)
	}

	m.removeCall(callID)

	return duration, nil
}

// GetCall returns the state of a call.
func (m *Manager) GetCall(callID string) *CallState {
	return m.getCall(callID)
}

// getCall retrieves a call state by ID.
func (m *Manager) getCall(callID string) *CallState {
	m.callsMu.RLock()
	defer m.callsMu.RUnlock()
	return m.calls[callID]
}

// removeCall removes a call from the active calls map.
func (m *Manager) removeCall(callID string) {
	m.callsMu.Lock()
	defer m.callsMu.Unlock()
	delete(m.calls, callID)
}

// waitForAnswer waits for the call to be answered.
func (m *Manager) waitForAnswer(ctx context.Context, call callsystem.Call, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return false
		default:
		}

		status := call.Status()
		if status == callsystem.StatusAnswered {
			return true
		}
		if status == callsystem.StatusEnded || status == callsystem.StatusFailed ||
			status == callsystem.StatusBusy || status == callsystem.StatusNoAnswer {
			return false
		}

		time.Sleep(500 * time.Millisecond)
	}
	return false
}

// speak generates TTS and streams it to the call.
func (m *Manager) speak(ctx context.Context, state *CallState, message string) error {
	// Record the assistant turn
	state.AddTurn("assistant", message)

	// Get the transport connection from the call
	transport := state.Call.Transport()
	if transport == nil {
		return fmt.Errorf("no transport connection available")
	}

	// Synthesize using streaming TTS with native ulaw output for Twilio
	stream, err := m.ttsProvider.SynthesizeStream(ctx, message, tts.SynthesisConfig{
		VoiceID:      m.config.TTSVoice,
		Model:        m.config.TTSModel,
		OutputFormat: "ulaw", // Native mu-law for Twilio
		SampleRate:   8000,   // Telephony sample rate
	})
	if err != nil {
		return fmt.Errorf("TTS synthesis failed: %w", err)
	}

	// Stream audio to the transport
	audioIn := transport.AudioIn()
	for chunk := range stream {
		if chunk.Error != nil {
			return fmt.Errorf("TTS stream error: %w", chunk.Error)
		}
		if len(chunk.Audio) > 0 {
			if _, err := audioIn.Write(chunk.Audio); err != nil {
				return fmt.Errorf("failed to write audio: %w", err)
			}
		}
		if chunk.IsFinal {
			break
		}
	}

	return nil
}

// speakAndListen speaks a message and waits for user response.
func (m *Manager) speakAndListen(ctx context.Context, state *CallState, message string) (string, error) {
	// Speak the message
	if err := m.speak(ctx, state, message); err != nil {
		return "", err
	}

	// Listen for response using STT
	response, err := m.listen(ctx, state)
	if err != nil {
		return "", fmt.Errorf("failed to listen: %w", err)
	}

	return response, nil
}

// listen waits for and transcribes user speech.
func (m *Manager) listen(ctx context.Context, state *CallState) (string, error) {
	// Get the transport connection from the call
	transport := state.Call.Transport()
	if transport == nil {
		return "", fmt.Errorf("no transport connection available")
	}

	// Create a streaming transcription session
	writer, events, err := m.sttProvider.TranscribeStream(ctx, stt.TranscriptionConfig{
		Language:          m.config.STTLanguage,
		Model:             m.config.STTModel,
		Encoding:          "mulaw",
		SampleRate:        8000,
		Channels:          1,
		EnablePunctuation: true,
	})
	if err != nil {
		return "", fmt.Errorf("failed to start transcription: %w", err)
	}
	defer func() { _ = writer.Close() }()

	// Start goroutine to stream audio from transport to STT
	audioCtx, audioCancel := context.WithCancel(ctx)
	defer audioCancel()

	go func() {
		audioOut := transport.AudioOut()
		buf := make([]byte, 1024)
		for {
			select {
			case <-audioCtx.Done():
				return
			default:
				n, err := audioOut.Read(buf)
				if err != nil {
					if err == io.EOF {
						return
					}
					return
				}
				if n > 0 {
					_, _ = writer.Write(buf[:n])
				}
			}
		}
	}()

	// Set up timeout
	timeout := time.Duration(m.config.TranscriptTimeoutMS) * time.Millisecond
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	var transcript string
	for {
		select {
		case <-ctx.Done():
			return transcript, ctx.Err()
		case <-timer.C:
			if transcript != "" {
				state.AddTurn("user", transcript)
			}
			return transcript, nil
		case event, ok := <-events:
			if !ok {
				if transcript != "" {
					state.AddTurn("user", transcript)
				}
				return transcript, nil
			}

			if event.Error != nil {
				return transcript, event.Error
			}

			if event.IsFinal && event.Transcript != "" {
				transcript = event.Transcript
				state.AddTurn("user", transcript)
				return transcript, nil
			}

			// Update partial transcript
			if event.Transcript != "" {
				transcript = event.Transcript
			}
		}
	}
}

// Close shuts down the call manager.
func (m *Manager) Close() error {
	m.callsMu.Lock()
	defer m.callsMu.Unlock()

	// Hangup all active calls
	ctx := context.Background()
	for _, state := range m.calls {
		_ = state.Call.Hangup(ctx)
	}

	m.calls = make(map[string]*CallState)

	if cs, ok := m.callSystem.(interface{ Close() error }); ok {
		return cs.Close()
	}

	return nil
}

// Transport returns the Twilio transport provider for WebSocket handling.
func (m *Manager) Transport() interface{} {
	if cs, ok := m.callSystem.(*twilliocallsystem.Provider); ok {
		return cs.Transport()
	}
	return nil
}
