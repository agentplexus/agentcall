// Package config provides configuration management for agentcomms.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// UnifiedConfig is the single JSON configuration file for agentcomms.
// It combines MCP server and daemon configuration into one file.
type UnifiedConfig struct {
	// Version is the config schema version (for future migrations).
	Version string `json:"version,omitempty"`

	// Server settings.
	Server ServerConfig `json:"server,omitempty"`

	// Agents defines the available AI agents.
	Agents []AgentConfig `json:"agents,omitempty"`

	// Voice settings for phone calls via Twilio.
	Voice *VoiceConfig `json:"voice,omitempty"`

	// Chat settings for Discord, Telegram, WhatsApp.
	Chat *ChatConfig `json:"chat,omitempty"`

	// Logging settings.
	Logging LoggingConfig `json:"logging,omitempty"`
}

// ServerConfig holds server settings.
type ServerConfig struct {
	// Port is the server port (default: 3333).
	Port int `json:"port,omitempty"`

	// DataDir overrides the default data directory (~/.agentcomms).
	DataDir string `json:"data_dir,omitempty"`
}

// AgentConfig defines an agent and its tmux target.
type AgentConfig struct {
	// ID is the unique agent identifier.
	ID string `json:"id"`

	// Type is the agent type (tmux, process).
	Type string `json:"type"`

	// TmuxSession is the tmux session name (for type=tmux).
	TmuxSession string `json:"tmux_session,omitempty"`

	// TmuxPane is the tmux pane identifier (for type=tmux).
	TmuxPane string `json:"tmux_pane,omitempty"`
}

// VoiceConfig holds voice calling configuration.
type VoiceConfig struct {
	// Phone provider settings (Twilio).
	Phone PhoneConfig `json:"phone"`

	// TTS (text-to-speech) settings.
	TTS TTSConfig `json:"tts"`

	// STT (speech-to-text) settings.
	STT STTConfig `json:"stt"`

	// Ngrok settings for webhook tunneling.
	Ngrok NgrokConfig `json:"ngrok"`

	// TranscriptTimeoutMS is the transcript timeout in milliseconds.
	TranscriptTimeoutMS int `json:"transcript_timeout_ms,omitempty"`
}

// PhoneConfig holds phone provider settings.
type PhoneConfig struct {
	// Provider is the phone provider ("twilio" or "telnyx").
	Provider string `json:"provider,omitempty"`

	// AccountSID is the Twilio account SID.
	AccountSID string `json:"account_sid"`

	// AuthToken is the Twilio auth token.
	AuthToken string `json:"auth_token"`

	// Number is the Twilio phone number (E.164 format).
	Number string `json:"number"`

	// UserNumber is the recipient phone number (E.164 format).
	UserNumber string `json:"user_number"`
}

// TTSConfig holds text-to-speech settings.
type TTSConfig struct {
	// Provider is the TTS provider ("elevenlabs", "deepgram", "openai").
	Provider string `json:"provider,omitempty"`

	// APIKey is the provider API key.
	APIKey string `json:"api_key"`

	// Voice is the voice ID (provider-specific).
	Voice string `json:"voice,omitempty"`

	// Model is the model ID (provider-specific).
	Model string `json:"model,omitempty"`
}

// STTConfig holds speech-to-text settings.
type STTConfig struct {
	// Provider is the STT provider ("elevenlabs", "deepgram", "openai").
	Provider string `json:"provider,omitempty"`

	// APIKey is the provider API key.
	APIKey string `json:"api_key"`

	// Model is the model ID (provider-specific).
	Model string `json:"model,omitempty"`

	// Language is the BCP-47 language code (e.g., "en-US").
	Language string `json:"language,omitempty"`

	// SilenceDurationMS is milliseconds of silence to detect end of speech.
	SilenceDurationMS int `json:"silence_duration_ms,omitempty"`
}

// NgrokConfig holds ngrok tunnel settings.
type NgrokConfig struct {
	// AuthToken is the ngrok auth token.
	AuthToken string `json:"auth_token"`

	// Domain is an optional custom ngrok domain.
	Domain string `json:"domain,omitempty"`
}

// ChatConfig holds chat provider configuration.
type ChatConfig struct {
	// Discord configuration (optional).
	Discord *DiscordConfig `json:"discord,omitempty"`

	// Telegram configuration (optional).
	Telegram *TelegramConfig `json:"telegram,omitempty"`

	// WhatsApp configuration (optional).
	WhatsApp *WhatsAppConfig `json:"whatsapp,omitempty"`

	// IRC configuration (optional).
	IRC *IRCConfig `json:"irc,omitempty"`

	// Channels maps chat channels to agents.
	Channels []ChannelMapping `json:"channels,omitempty"`
}

// DiscordConfig holds Discord-specific configuration.
type DiscordConfig struct {
	// Enabled controls whether Discord is active.
	Enabled bool `json:"enabled,omitempty"`

	// Token is the Discord bot token.
	Token string `json:"token"`

	// GuildID is the Discord guild (server) ID for filtering.
	GuildID string `json:"guild_id,omitempty"`
}

// TelegramConfig holds Telegram-specific configuration.
type TelegramConfig struct {
	// Enabled controls whether Telegram is active.
	Enabled bool `json:"enabled,omitempty"`

	// Token is the Telegram bot token.
	Token string `json:"token"`
}

// WhatsAppConfig holds WhatsApp-specific configuration.
type WhatsAppConfig struct {
	// Enabled controls whether WhatsApp is active.
	Enabled bool `json:"enabled,omitempty"`

	// DBPath is the SQLite database path for session storage.
	DBPath string `json:"db_path"`
}

// IRCConfig holds IRC-specific configuration.
type IRCConfig struct {
	// Enabled controls whether IRC is active.
	Enabled bool `json:"enabled,omitempty"`

	// Server is the IRC server address (e.g., "irc.libera.chat:6697").
	Server string `json:"server"`

	// Nick is the bot's nickname.
	Nick string `json:"nick"`

	// Password is the NickServ password (optional).
	Password string `json:"password,omitempty"`

	// Channels is the list of channels to join on connect.
	Channels []string `json:"channels"`

	// UseTLS enables TLS for the connection (default: true).
	UseTLS *bool `json:"use_tls,omitempty"`
}

// ChannelMapping maps a chat channel to an agent.
type ChannelMapping struct {
	// ChannelID is the full channel identifier (provider:chatid).
	ChannelID string `json:"channel_id"`

	// AgentID is the target agent ID.
	AgentID string `json:"agent_id"`
}

// LoggingConfig holds logging settings.
type LoggingConfig struct {
	// Level is the log level (debug, info, warn, error).
	Level string `json:"level,omitempty"`
}

// DefaultUnifiedConfig returns a UnifiedConfig with sensible defaults.
func DefaultUnifiedConfig() *UnifiedConfig {
	return &UnifiedConfig{
		Version: "1",
		Server: ServerConfig{
			Port: 3333,
		},
		Logging: LoggingConfig{
			Level: "info",
		},
	}
}

// LoadUnifiedConfig loads configuration from a JSON file.
// It expands environment variables in string values using ${VAR} syntax.
func LoadUnifiedConfig(path string) (*UnifiedConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables
	expanded := expandEnvVars(string(data))

	cfg := DefaultUnifiedConfig()
	if err := json.Unmarshal([]byte(expanded), cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

// LoadUnifiedConfigFromDir loads config.json from a directory.
func LoadUnifiedConfigFromDir(dir string) (*UnifiedConfig, error) {
	return LoadUnifiedConfig(filepath.Join(dir, "config.json"))
}

// envVarRegex matches ${VAR} or $VAR patterns.
var envVarRegex = regexp.MustCompile(`\$\{([^}]+)\}|\$([A-Za-z_][A-Za-z0-9_]*)`)

// expandEnvVars replaces ${VAR} and $VAR with environment variable values.
func expandEnvVars(s string) string {
	return envVarRegex.ReplaceAllStringFunc(s, func(match string) string {
		var varName string
		if strings.HasPrefix(match, "${") {
			// ${VAR} format
			varName = match[2 : len(match)-1]
		} else {
			// $VAR format
			varName = match[1:]
		}
		return os.Getenv(varName)
	})
}

// Save writes the configuration to a JSON file.
func (c *UnifiedConfig) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate checks the configuration for errors.
func (c *UnifiedConfig) Validate() error {
	var errors []string

	// Validate agents
	agentIDs := make(map[string]bool)
	for _, agent := range c.Agents {
		if agent.ID == "" {
			errors = append(errors, "agent ID is required")
			continue
		}
		if agentIDs[agent.ID] {
			errors = append(errors, fmt.Sprintf("duplicate agent ID: %s", agent.ID))
		}
		agentIDs[agent.ID] = true

		if agent.Type == "" {
			errors = append(errors, fmt.Sprintf("agent %s: type is required", agent.ID))
		}
		if agent.Type == "tmux" && agent.TmuxSession == "" {
			errors = append(errors, fmt.Sprintf("agent %s: tmux_session is required for tmux type", agent.ID))
		}
	}

	// Validate voice config
	if c.Voice != nil {
		if c.Voice.Phone.AccountSID == "" {
			errors = append(errors, "voice.phone.account_sid is required")
		}
		if c.Voice.Phone.AuthToken == "" {
			errors = append(errors, "voice.phone.auth_token is required")
		}
		if c.Voice.Phone.Number == "" {
			errors = append(errors, "voice.phone.number is required")
		}
		if c.Voice.Phone.UserNumber == "" {
			errors = append(errors, "voice.phone.user_number is required")
		}
		if c.Voice.TTS.APIKey == "" {
			errors = append(errors, "voice.tts.api_key is required")
		}
		if c.Voice.STT.APIKey == "" {
			errors = append(errors, "voice.stt.api_key is required")
		}
		if c.Voice.Ngrok.AuthToken == "" {
			errors = append(errors, "voice.ngrok.auth_token is required")
		}

		// Validate provider names
		validProviders := map[string]bool{"elevenlabs": true, "deepgram": true, "openai": true}
		if c.Voice.TTS.Provider != "" && !validProviders[c.Voice.TTS.Provider] {
			errors = append(errors, fmt.Sprintf("invalid TTS provider %q", c.Voice.TTS.Provider))
		}
		if c.Voice.STT.Provider != "" && !validProviders[c.Voice.STT.Provider] {
			errors = append(errors, fmt.Sprintf("invalid STT provider %q", c.Voice.STT.Provider))
		}
	}

	// Validate chat config
	if c.Chat != nil {
		if c.Chat.Discord != nil && c.Chat.Discord.Enabled && c.Chat.Discord.Token == "" {
			errors = append(errors, "chat.discord.token is required when enabled")
		}
		if c.Chat.Telegram != nil && c.Chat.Telegram.Enabled && c.Chat.Telegram.Token == "" {
			errors = append(errors, "chat.telegram.token is required when enabled")
		}
		if c.Chat.WhatsApp != nil && c.Chat.WhatsApp.Enabled && c.Chat.WhatsApp.DBPath == "" {
			errors = append(errors, "chat.whatsapp.db_path is required when enabled")
		}
		if c.Chat.IRC != nil && c.Chat.IRC.Enabled {
			if c.Chat.IRC.Server == "" {
				errors = append(errors, "chat.irc.server is required when enabled")
			}
			if c.Chat.IRC.Nick == "" {
				errors = append(errors, "chat.irc.nick is required when enabled")
			}
		}

		// Validate channel mappings
		for _, mapping := range c.Chat.Channels {
			if mapping.ChannelID == "" {
				errors = append(errors, "chat.channels[].channel_id is required")
			}
			if mapping.AgentID == "" {
				errors = append(errors, "chat.channels[].agent_id is required")
			}
			if mapping.AgentID != "" && !agentIDs[mapping.AgentID] {
				errors = append(errors, fmt.Sprintf("chat.channels: unknown agent_id %q", mapping.AgentID))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration errors:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// ToLegacyConfig converts UnifiedConfig to the legacy Config for backward compatibility.
func (c *UnifiedConfig) ToLegacyConfig() *Config {
	cfg := DefaultConfig()
	cfg.Port = c.Server.Port

	if c.Voice != nil {
		cfg.PhoneProvider = c.Voice.Phone.Provider
		if cfg.PhoneProvider == "" {
			cfg.PhoneProvider = "twilio"
		}
		cfg.PhoneAccountSID = c.Voice.Phone.AccountSID
		cfg.PhoneAuthToken = c.Voice.Phone.AuthToken
		cfg.PhoneNumber = c.Voice.Phone.Number
		cfg.UserPhoneNumber = c.Voice.Phone.UserNumber

		cfg.TTSProvider = c.Voice.TTS.Provider
		if cfg.TTSProvider == "" {
			cfg.TTSProvider = ProviderElevenLabs
		}
		cfg.TTSVoice = c.Voice.TTS.Voice
		cfg.TTSModel = c.Voice.TTS.Model

		cfg.STTProvider = c.Voice.STT.Provider
		if cfg.STTProvider == "" {
			cfg.STTProvider = ProviderDeepgram
		}
		cfg.STTModel = c.Voice.STT.Model
		cfg.STTLanguage = c.Voice.STT.Language
		cfg.STTSilenceDurationMS = c.Voice.STT.SilenceDurationMS

		cfg.NgrokAuthToken = c.Voice.Ngrok.AuthToken
		cfg.NgrokDomain = c.Voice.Ngrok.Domain
		cfg.TranscriptTimeoutMS = c.Voice.TranscriptTimeoutMS

		// Set API keys based on provider
		switch cfg.TTSProvider {
		case ProviderElevenLabs:
			cfg.ElevenLabsAPIKey = c.Voice.TTS.APIKey
		case ProviderDeepgram:
			cfg.DeepgramAPIKey = c.Voice.TTS.APIKey
		case ProviderOpenAI:
			cfg.OpenAIAPIKey = c.Voice.TTS.APIKey
		}
		switch cfg.STTProvider {
		case ProviderElevenLabs:
			if cfg.ElevenLabsAPIKey == "" {
				cfg.ElevenLabsAPIKey = c.Voice.STT.APIKey
			}
		case ProviderDeepgram:
			if cfg.DeepgramAPIKey == "" {
				cfg.DeepgramAPIKey = c.Voice.STT.APIKey
			}
		case ProviderOpenAI:
			if cfg.OpenAIAPIKey == "" {
				cfg.OpenAIAPIKey = c.Voice.STT.APIKey
			}
		}
	}

	if c.Chat != nil {
		if c.Chat.Discord != nil {
			cfg.DiscordEnabled = c.Chat.Discord.Enabled
			cfg.DiscordToken = c.Chat.Discord.Token
			cfg.DiscordGuildID = c.Chat.Discord.GuildID
		}
		if c.Chat.Telegram != nil {
			cfg.TelegramEnabled = c.Chat.Telegram.Enabled
			cfg.TelegramToken = c.Chat.Telegram.Token
		}
		if c.Chat.WhatsApp != nil {
			cfg.WhatsAppEnabled = c.Chat.WhatsApp.Enabled
			cfg.WhatsAppDBPath = c.Chat.WhatsApp.DBPath
		}
		if c.Chat.IRC != nil {
			cfg.IRCEnabled = c.Chat.IRC.Enabled
			cfg.IRCServer = c.Chat.IRC.Server
			cfg.IRCNick = c.Chat.IRC.Nick
			cfg.IRCPassword = c.Chat.IRC.Password
			cfg.IRCChannels = c.Chat.IRC.Channels
			if c.Chat.IRC.UseTLS != nil {
				cfg.IRCUseTLS = *c.Chat.IRC.UseTLS
			} else {
				cfg.IRCUseTLS = true // Default to TLS enabled
			}
		}
	}

	return cfg
}

// FindAgentByChannel returns the agent ID for a chat channel.
func (c *UnifiedConfig) FindAgentByChannel(channelID string) (string, bool) {
	if c.Chat == nil {
		return "", false
	}
	for _, mapping := range c.Chat.Channels {
		if mapping.ChannelID == channelID {
			return mapping.AgentID, true
		}
	}
	return "", false
}

// GetAgent returns the agent config by ID.
func (c *UnifiedConfig) GetAgent(id string) (*AgentConfig, bool) {
	for i := range c.Agents {
		if c.Agents[i].ID == id {
			return &c.Agents[i], true
		}
	}
	return nil, false
}

// HasChatProviders returns true if any chat provider is configured.
func (c *UnifiedConfig) HasChatProviders() bool {
	if c.Chat == nil {
		return false
	}
	return (c.Chat.Discord != nil && c.Chat.Discord.Enabled) ||
		(c.Chat.Telegram != nil && c.Chat.Telegram.Enabled) ||
		(c.Chat.WhatsApp != nil && c.Chat.WhatsApp.Enabled) ||
		(c.Chat.IRC != nil && c.Chat.IRC.Enabled)
}

// VoiceEnabled returns true if voice calling is configured.
func (c *UnifiedConfig) VoiceEnabled() bool {
	return c.Voice != nil && c.Voice.Phone.AccountSID != ""
}
