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

	// ElevenLabs TTS settings
	ElevenLabsAPIKey string
	TTSVoice         string // ElevenLabs voice ID (e.g., "Rachel")
	TTSModel         string // ElevenLabs model (e.g., "eleven_turbo_v2_5")

	// Deepgram STT settings
	DeepgramAPIKey       string
	STTModel             string // Deepgram model (e.g., "nova-2")
	STTLanguage          string // BCP-47 language code (e.g., "en-US")
	STTSilenceDurationMS int    // milliseconds of silence to detect end of speech

	// ngrok settings
	NgrokAuthToken string
	NgrokDomain    string // optional custom domain

	// Timeouts
	TranscriptTimeoutMS int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Port:                 3333,
		PhoneProvider:        "twilio",
		TTSVoice:             "Rachel",            // ElevenLabs voice
		TTSModel:             "eleven_turbo_v2_5", // Low-latency ElevenLabs model
		STTModel:             "nova-2",            // Deepgram model
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

	// ElevenLabs TTS
	cfg.ElevenLabsAPIKey = os.Getenv("AGENTCALL_ELEVENLABS_API_KEY")
	if cfg.ElevenLabsAPIKey == "" {
		cfg.ElevenLabsAPIKey = os.Getenv("ELEVENLABS_API_KEY") // fallback
	}
	if voice := os.Getenv("AGENTCALL_TTS_VOICE"); voice != "" {
		cfg.TTSVoice = voice
	}
	if model := os.Getenv("AGENTCALL_TTS_MODEL"); model != "" {
		cfg.TTSModel = model
	}

	// Deepgram STT
	cfg.DeepgramAPIKey = os.Getenv("AGENTCALL_DEEPGRAM_API_KEY")
	if cfg.DeepgramAPIKey == "" {
		cfg.DeepgramAPIKey = os.Getenv("DEEPGRAM_API_KEY") // fallback
	}
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
	if c.ElevenLabsAPIKey == "" {
		missing = append(missing, "AGENTCALL_ELEVENLABS_API_KEY or ELEVENLABS_API_KEY")
	}
	if c.DeepgramAPIKey == "" {
		missing = append(missing, "AGENTCALL_DEEPGRAM_API_KEY or DEEPGRAM_API_KEY")
	}
	if c.NgrokAuthToken == "" {
		missing = append(missing, "AGENTCALL_NGROK_AUTHTOKEN or NGROK_AUTHTOKEN")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %v", missing)
	}

	return nil
}
