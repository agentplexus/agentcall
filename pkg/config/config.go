// Package config provides configuration management for agentcall.
package config

import (
	"fmt"
	"os"
)

// Config holds all configuration for the agentcall server.
type Config struct {
	// Server settings
	Port int

	// Phone provider settings (Twilio)
	PhoneProvider   string // "twilio" or "telnyx"
	PhoneAccountSID string
	PhoneAuthToken  string
	PhoneNumber     string // E.164 format, e.g., +15551234567
	UserPhoneNumber string // E.164 format

	// Voice provider selection
	TTSProvider string // "elevenlabs" or "deepgram"
	STTProvider string // "elevenlabs" or "deepgram"

	// ElevenLabs settings
	ElevenLabsAPIKey string

	// Deepgram settings
	DeepgramAPIKey string

	// TTS settings (provider-agnostic)
	TTSVoice string // Voice ID (provider-specific)
	TTSModel string // Model ID (provider-specific)

	// STT settings (provider-agnostic)
	STTModel             string // Model ID (provider-specific)
	STTLanguage          string // BCP-47 language code (e.g., "en-US")
	STTSilenceDurationMS int    // milliseconds of silence to detect end of speech

	// ngrok settings
	NgrokAuthToken string
	NgrokDomain    string // optional custom domain

	// Timeouts
	TranscriptTimeoutMS int
}

// Provider constants.
const (
	ProviderElevenLabs = "elevenlabs"
	ProviderDeepgram   = "deepgram"
)

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Port:                 3333,
		PhoneProvider:        "twilio",
		TTSProvider:          ProviderElevenLabs, // Default to ElevenLabs for TTS
		STTProvider:          ProviderDeepgram,   // Default to Deepgram for STT
		TTSVoice:             "Rachel",           // ElevenLabs default voice
		TTSModel:             "eleven_turbo_v2_5",
		STTModel:             "nova-2",
		STTLanguage:          "en-US",
		STTSilenceDurationMS: 800,
		TranscriptTimeoutMS:  180000, // 3 minutes
	}
}

// LoadFromEnv loads configuration from environment variables.
func LoadFromEnv() (*Config, error) {
	cfg := DefaultConfig()

	// Server port
	if port := os.Getenv("AGENTCALL_PORT"); port != "" {
		var p int
		if _, err := fmt.Sscanf(port, "%d", &p); err == nil {
			cfg.Port = p
		}
	}

	// Phone provider
	if provider := os.Getenv("AGENTCALL_PHONE_PROVIDER"); provider != "" {
		cfg.PhoneProvider = provider
	}
	cfg.PhoneAccountSID = os.Getenv("AGENTCALL_PHONE_ACCOUNT_SID")
	cfg.PhoneAuthToken = os.Getenv("AGENTCALL_PHONE_AUTH_TOKEN")
	cfg.PhoneNumber = os.Getenv("AGENTCALL_PHONE_NUMBER")
	cfg.UserPhoneNumber = os.Getenv("AGENTCALL_USER_PHONE_NUMBER")

	// Voice provider selection
	if ttsProvider := os.Getenv("AGENTCALL_TTS_PROVIDER"); ttsProvider != "" {
		cfg.TTSProvider = ttsProvider
	}
	if sttProvider := os.Getenv("AGENTCALL_STT_PROVIDER"); sttProvider != "" {
		cfg.STTProvider = sttProvider
	}

	// ElevenLabs API key
	cfg.ElevenLabsAPIKey = os.Getenv("AGENTCALL_ELEVENLABS_API_KEY")
	if cfg.ElevenLabsAPIKey == "" {
		cfg.ElevenLabsAPIKey = os.Getenv("ELEVENLABS_API_KEY") // fallback
	}

	// Deepgram API key
	cfg.DeepgramAPIKey = os.Getenv("AGENTCALL_DEEPGRAM_API_KEY")
	if cfg.DeepgramAPIKey == "" {
		cfg.DeepgramAPIKey = os.Getenv("DEEPGRAM_API_KEY") // fallback
	}

	// TTS settings
	if voice := os.Getenv("AGENTCALL_TTS_VOICE"); voice != "" {
		cfg.TTSVoice = voice
	}
	if model := os.Getenv("AGENTCALL_TTS_MODEL"); model != "" {
		cfg.TTSModel = model
	}

	// STT settings
	if model := os.Getenv("AGENTCALL_STT_MODEL"); model != "" {
		cfg.STTModel = model
	}
	if lang := os.Getenv("AGENTCALL_STT_LANGUAGE"); lang != "" {
		cfg.STTLanguage = lang
	}
	if silence := os.Getenv("AGENTCALL_STT_SILENCE_DURATION_MS"); silence != "" {
		var s int
		if _, err := fmt.Sscanf(silence, "%d", &s); err == nil {
			cfg.STTSilenceDurationMS = s
		}
	}

	// ngrok
	cfg.NgrokAuthToken = os.Getenv("AGENTCALL_NGROK_AUTHTOKEN")
	if cfg.NgrokAuthToken == "" {
		cfg.NgrokAuthToken = os.Getenv("NGROK_AUTHTOKEN") // fallback
	}
	cfg.NgrokDomain = os.Getenv("AGENTCALL_NGROK_DOMAIN")

	// Transcript timeout
	if timeout := os.Getenv("AGENTCALL_TRANSCRIPT_TIMEOUT_MS"); timeout != "" {
		var t int
		if _, err := fmt.Sscanf(timeout, "%d", &t); err == nil {
			cfg.TranscriptTimeoutMS = t
		}
	}

	return cfg, cfg.Validate()
}

// Validate checks that required configuration is present.
func (c *Config) Validate() error {
	var missing []string
	var errors []string

	// Phone provider settings
	if c.PhoneAccountSID == "" {
		missing = append(missing, "AGENTCALL_PHONE_ACCOUNT_SID")
	}
	if c.PhoneAuthToken == "" {
		missing = append(missing, "AGENTCALL_PHONE_AUTH_TOKEN")
	}
	if c.PhoneNumber == "" {
		missing = append(missing, "AGENTCALL_PHONE_NUMBER")
	}
	if c.UserPhoneNumber == "" {
		missing = append(missing, "AGENTCALL_USER_PHONE_NUMBER")
	}

	// Validate provider selection
	if c.TTSProvider != ProviderElevenLabs && c.TTSProvider != ProviderDeepgram {
		errors = append(errors, fmt.Sprintf("invalid TTS provider %q (must be %q or %q)", c.TTSProvider, ProviderElevenLabs, ProviderDeepgram))
	}
	if c.STTProvider != ProviderElevenLabs && c.STTProvider != ProviderDeepgram {
		errors = append(errors, fmt.Sprintf("invalid STT provider %q (must be %q or %q)", c.STTProvider, ProviderElevenLabs, ProviderDeepgram))
	}

	// Check API keys based on selected providers
	needsElevenLabs := c.TTSProvider == ProviderElevenLabs || c.STTProvider == ProviderElevenLabs
	needsDeepgram := c.TTSProvider == ProviderDeepgram || c.STTProvider == ProviderDeepgram

	if needsElevenLabs && c.ElevenLabsAPIKey == "" {
		missing = append(missing, "AGENTCALL_ELEVENLABS_API_KEY or ELEVENLABS_API_KEY")
	}
	if needsDeepgram && c.DeepgramAPIKey == "" {
		missing = append(missing, "AGENTCALL_DEEPGRAM_API_KEY or DEEPGRAM_API_KEY")
	}

	// ngrok
	if c.NgrokAuthToken == "" {
		missing = append(missing, "AGENTCALL_NGROK_AUTHTOKEN or NGROK_AUTHTOKEN")
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration errors: %v", errors)
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %v", missing)
	}

	return nil
}

// NeedsElevenLabs returns true if any provider uses ElevenLabs.
func (c *Config) NeedsElevenLabs() bool {
	return c.TTSProvider == ProviderElevenLabs || c.STTProvider == ProviderElevenLabs
}

// NeedsDeepgram returns true if any provider uses Deepgram.
func (c *Config) NeedsDeepgram() bool {
	return c.TTSProvider == ProviderDeepgram || c.STTProvider == ProviderDeepgram
}
