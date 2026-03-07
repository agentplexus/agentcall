// Package config provides configuration management for agentcomms.
package config

import (
	"fmt"
	"os"
)

// Config holds all configuration for the agentcomms server.
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
	TTSProvider string // "elevenlabs", "deepgram", or "openai"
	STTProvider string // "elevenlabs", "deepgram", or "openai"

	// ElevenLabs settings
	ElevenLabsAPIKey string

	// Deepgram settings
	DeepgramAPIKey string

	// OpenAI settings
	OpenAIAPIKey string

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

	// Chat provider settings
	WhatsAppEnabled bool
	WhatsAppDBPath  string

	DiscordEnabled bool
	DiscordToken   string
	DiscordGuildID string

	TelegramEnabled bool
	TelegramToken   string
}

// Provider constants.
const (
	ProviderElevenLabs = "elevenlabs"
	ProviderDeepgram   = "deepgram"
	ProviderOpenAI     = "openai"
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
		WhatsAppDBPath:       "./whatsapp.db",
	}
}

// LoadFromEnv loads configuration from environment variables.
// Supports both AGENTCOMMS_ and legacy AGENTCALL_ prefixes with AGENTCOMMS_ taking precedence.
func LoadFromEnv() (*Config, error) {
	cfg := DefaultConfig()

	// Server port
	if port := getEnvWithFallback("AGENTCOMMS_PORT", "AGENTCALL_PORT"); port != "" {
		var p int
		if _, err := fmt.Sscanf(port, "%d", &p); err == nil {
			cfg.Port = p
		}
	}

	// Phone provider
	if provider := getEnvWithFallback("AGENTCOMMS_PHONE_PROVIDER", "AGENTCALL_PHONE_PROVIDER"); provider != "" {
		cfg.PhoneProvider = provider
	}
	cfg.PhoneAccountSID = getEnvWithFallback("AGENTCOMMS_PHONE_ACCOUNT_SID", "AGENTCALL_PHONE_ACCOUNT_SID")
	cfg.PhoneAuthToken = getEnvWithFallback("AGENTCOMMS_PHONE_AUTH_TOKEN", "AGENTCALL_PHONE_AUTH_TOKEN")
	cfg.PhoneNumber = getEnvWithFallback("AGENTCOMMS_PHONE_NUMBER", "AGENTCALL_PHONE_NUMBER")
	cfg.UserPhoneNumber = getEnvWithFallback("AGENTCOMMS_USER_PHONE_NUMBER", "AGENTCALL_USER_PHONE_NUMBER")

	// Voice provider selection
	if ttsProvider := getEnvWithFallback("AGENTCOMMS_TTS_PROVIDER", "AGENTCALL_TTS_PROVIDER"); ttsProvider != "" {
		cfg.TTSProvider = ttsProvider
	}
	if sttProvider := getEnvWithFallback("AGENTCOMMS_STT_PROVIDER", "AGENTCALL_STT_PROVIDER"); sttProvider != "" {
		cfg.STTProvider = sttProvider
	}

	// ElevenLabs API key
	cfg.ElevenLabsAPIKey = getEnvWithFallback("AGENTCOMMS_ELEVENLABS_API_KEY", "AGENTCALL_ELEVENLABS_API_KEY")
	if cfg.ElevenLabsAPIKey == "" {
		cfg.ElevenLabsAPIKey = os.Getenv("ELEVENLABS_API_KEY") // fallback
	}

	// Deepgram API key
	cfg.DeepgramAPIKey = getEnvWithFallback("AGENTCOMMS_DEEPGRAM_API_KEY", "AGENTCALL_DEEPGRAM_API_KEY")
	if cfg.DeepgramAPIKey == "" {
		cfg.DeepgramAPIKey = os.Getenv("DEEPGRAM_API_KEY") // fallback
	}

	// OpenAI API key
	cfg.OpenAIAPIKey = getEnvWithFallback("AGENTCOMMS_OPENAI_API_KEY", "")
	if cfg.OpenAIAPIKey == "" {
		cfg.OpenAIAPIKey = os.Getenv("OPENAI_API_KEY") // fallback
	}

	// TTS settings
	if voice := getEnvWithFallback("AGENTCOMMS_TTS_VOICE", "AGENTCALL_TTS_VOICE"); voice != "" {
		cfg.TTSVoice = voice
	}
	if model := getEnvWithFallback("AGENTCOMMS_TTS_MODEL", "AGENTCALL_TTS_MODEL"); model != "" {
		cfg.TTSModel = model
	}

	// STT settings
	if model := getEnvWithFallback("AGENTCOMMS_STT_MODEL", "AGENTCALL_STT_MODEL"); model != "" {
		cfg.STTModel = model
	}
	if lang := getEnvWithFallback("AGENTCOMMS_STT_LANGUAGE", "AGENTCALL_STT_LANGUAGE"); lang != "" {
		cfg.STTLanguage = lang
	}
	if silence := getEnvWithFallback("AGENTCOMMS_STT_SILENCE_DURATION_MS", "AGENTCALL_STT_SILENCE_DURATION_MS"); silence != "" {
		var s int
		if _, err := fmt.Sscanf(silence, "%d", &s); err == nil {
			cfg.STTSilenceDurationMS = s
		}
	}

	// ngrok
	cfg.NgrokAuthToken = getEnvWithFallback("AGENTCOMMS_NGROK_AUTHTOKEN", "AGENTCALL_NGROK_AUTHTOKEN")
	if cfg.NgrokAuthToken == "" {
		cfg.NgrokAuthToken = os.Getenv("NGROK_AUTHTOKEN") // fallback
	}
	cfg.NgrokDomain = getEnvWithFallback("AGENTCOMMS_NGROK_DOMAIN", "AGENTCALL_NGROK_DOMAIN")

	// Transcript timeout
	if timeout := getEnvWithFallback("AGENTCOMMS_TRANSCRIPT_TIMEOUT_MS", "AGENTCALL_TRANSCRIPT_TIMEOUT_MS"); timeout != "" {
		var t int
		if _, err := fmt.Sscanf(timeout, "%d", &t); err == nil {
			cfg.TranscriptTimeoutMS = t
		}
	}

	// Chat providers - WhatsApp
	if enabled := os.Getenv("AGENTCOMMS_WHATSAPP_ENABLED"); enabled == "true" || enabled == "1" {
		cfg.WhatsAppEnabled = true
	}
	if dbPath := os.Getenv("AGENTCOMMS_WHATSAPP_DB_PATH"); dbPath != "" {
		cfg.WhatsAppDBPath = dbPath
	}

	// Chat providers - Discord
	if enabled := os.Getenv("AGENTCOMMS_DISCORD_ENABLED"); enabled == "true" || enabled == "1" {
		cfg.DiscordEnabled = true
	}
	cfg.DiscordToken = os.Getenv("AGENTCOMMS_DISCORD_TOKEN")
	if cfg.DiscordToken == "" {
		cfg.DiscordToken = os.Getenv("DISCORD_TOKEN") // fallback
	}
	cfg.DiscordGuildID = os.Getenv("AGENTCOMMS_DISCORD_GUILD_ID")

	// Chat providers - Telegram
	if enabled := os.Getenv("AGENTCOMMS_TELEGRAM_ENABLED"); enabled == "true" || enabled == "1" {
		cfg.TelegramEnabled = true
	}
	cfg.TelegramToken = os.Getenv("AGENTCOMMS_TELEGRAM_TOKEN")
	if cfg.TelegramToken == "" {
		cfg.TelegramToken = os.Getenv("TELEGRAM_BOT_TOKEN") // fallback
	}

	return cfg, cfg.Validate()
}

// getEnvWithFallback returns the value of the primary env var, or falls back to secondary.
func getEnvWithFallback(primary, secondary string) string {
	if val := os.Getenv(primary); val != "" {
		return val
	}
	if secondary != "" {
		return os.Getenv(secondary)
	}
	return ""
}

// Validate checks that required configuration is present.
func (c *Config) Validate() error {
	var missing []string
	var errors []string

	// Phone provider settings - only required if voice is enabled
	if c.VoiceEnabled() {
		if c.PhoneAccountSID == "" {
			missing = append(missing, "AGENTCOMMS_PHONE_ACCOUNT_SID")
		}
		if c.PhoneAuthToken == "" {
			missing = append(missing, "AGENTCOMMS_PHONE_AUTH_TOKEN")
		}
		if c.PhoneNumber == "" {
			missing = append(missing, "AGENTCOMMS_PHONE_NUMBER")
		}
		if c.UserPhoneNumber == "" {
			missing = append(missing, "AGENTCOMMS_USER_PHONE_NUMBER")
		}

		// Validate provider selection
		validProviders := map[string]bool{ProviderElevenLabs: true, ProviderDeepgram: true, ProviderOpenAI: true}
		if !validProviders[c.TTSProvider] {
			errors = append(errors, fmt.Sprintf("invalid TTS provider %q (must be %q, %q, or %q)", c.TTSProvider, ProviderElevenLabs, ProviderDeepgram, ProviderOpenAI))
		}
		if !validProviders[c.STTProvider] {
			errors = append(errors, fmt.Sprintf("invalid STT provider %q (must be %q, %q, or %q)", c.STTProvider, ProviderElevenLabs, ProviderDeepgram, ProviderOpenAI))
		}

		// Check API keys based on selected providers
		if c.NeedsElevenLabs() && c.ElevenLabsAPIKey == "" {
			missing = append(missing, "AGENTCOMMS_ELEVENLABS_API_KEY or ELEVENLABS_API_KEY")
		}
		if c.NeedsDeepgram() && c.DeepgramAPIKey == "" {
			missing = append(missing, "AGENTCOMMS_DEEPGRAM_API_KEY or DEEPGRAM_API_KEY")
		}
		if c.NeedsOpenAI() && c.OpenAIAPIKey == "" {
			missing = append(missing, "AGENTCOMMS_OPENAI_API_KEY or OPENAI_API_KEY")
		}

		// ngrok required for voice
		if c.NgrokAuthToken == "" {
			missing = append(missing, "AGENTCOMMS_NGROK_AUTHTOKEN or NGROK_AUTHTOKEN")
		}
	}

	// Chat provider validation
	if c.DiscordEnabled && c.DiscordToken == "" {
		missing = append(missing, "AGENTCOMMS_DISCORD_TOKEN or DISCORD_TOKEN")
	}
	if c.TelegramEnabled && c.TelegramToken == "" {
		missing = append(missing, "AGENTCOMMS_TELEGRAM_TOKEN or TELEGRAM_BOT_TOKEN")
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration errors: %v", errors)
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %v", missing)
	}

	return nil
}

// VoiceEnabled returns true if voice calling is configured.
func (c *Config) VoiceEnabled() bool {
	return c.PhoneAccountSID != "" || c.PhoneAuthToken != "" || c.PhoneNumber != ""
}

// ChatEnabled returns true if any chat provider is enabled.
func (c *Config) ChatEnabled() bool {
	return c.WhatsAppEnabled || c.DiscordEnabled || c.TelegramEnabled
}

// NeedsElevenLabs returns true if any provider uses ElevenLabs.
func (c *Config) NeedsElevenLabs() bool {
	return c.TTSProvider == ProviderElevenLabs || c.STTProvider == ProviderElevenLabs
}

// NeedsDeepgram returns true if any provider uses Deepgram.
func (c *Config) NeedsDeepgram() bool {
	return c.TTSProvider == ProviderDeepgram || c.STTProvider == ProviderDeepgram
}

// NeedsOpenAI returns true if any provider uses OpenAI.
func (c *Config) NeedsOpenAI() bool {
	return c.TTSProvider == ProviderOpenAI || c.STTProvider == ProviderOpenAI
}

// TTSAPIKey returns the API key for the configured TTS provider.
func (c *Config) TTSAPIKey() string {
	switch c.TTSProvider {
	case ProviderElevenLabs:
		return c.ElevenLabsAPIKey
	case ProviderDeepgram:
		return c.DeepgramAPIKey
	case ProviderOpenAI:
		return c.OpenAIAPIKey
	default:
		return ""
	}
}

// STTAPIKey returns the API key for the configured STT provider.
func (c *Config) STTAPIKey() string {
	switch c.STTProvider {
	case ProviderElevenLabs:
		return c.ElevenLabsAPIKey
	case ProviderDeepgram:
		return c.DeepgramAPIKey
	case ProviderOpenAI:
		return c.OpenAIAPIKey
	default:
		return ""
	}
}
