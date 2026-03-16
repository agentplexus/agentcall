// Package daemon provides the AgentComms daemon configuration.
package daemon

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// AgentConfig defines an agent and its tmux target.
type AgentConfig struct {
	// ID is the unique agent identifier.
	ID string `yaml:"id"`

	// Type is the agent type (tmux, process).
	Type string `yaml:"type"`

	// TmuxSession is the tmux session name (for type=tmux).
	TmuxSession string `yaml:"tmux_session"`

	// TmuxPane is the tmux pane identifier (for type=tmux).
	TmuxPane string `yaml:"tmux_pane"`
}

// ChannelMapping maps a chat channel to an agent.
// ChannelID format: "provider:chatid" (e.g., "discord:123456789")
type ChannelMapping struct {
	// ChannelID is the full channel identifier (provider:chatid).
	ChannelID string `yaml:"channel_id"`

	// AgentID is the target agent ID.
	AgentID string `yaml:"agent_id"`
}

// DiscordConfig holds Discord-specific configuration.
type DiscordConfig struct {
	// Token is the Discord bot token.
	Token string `yaml:"token"`

	// GuildID is the Discord guild (server) ID for filtering.
	GuildID string `yaml:"guild_id"`
}

// TelegramConfig holds Telegram-specific configuration.
type TelegramConfig struct {
	// Token is the Telegram bot token.
	Token string `yaml:"token"`
}

// WhatsAppConfig holds WhatsApp-specific configuration.
type WhatsAppConfig struct {
	// DBPath is the SQLite database path for session storage.
	DBPath string `yaml:"db_path"`
}

// ChatConfig holds configuration for all chat providers.
type ChatConfig struct {
	// Discord configuration (optional).
	Discord *DiscordConfig `yaml:"discord"`

	// Telegram configuration (optional).
	Telegram *TelegramConfig `yaml:"telegram"`

	// WhatsApp configuration (optional).
	WhatsApp *WhatsAppConfig `yaml:"whatsapp"`

	// Channels maps chat channels to agents.
	Channels []ChannelMapping `yaml:"channels"`
}

// DaemonConfig holds the daemon configuration loaded from config.yaml.
type DaemonConfig struct {
	// DataDir overrides the default data directory.
	DataDir string `yaml:"data_dir"`

	// LogLevel sets the logging level (debug, info, warn, error).
	LogLevel string `yaml:"log_level"`

	// Agents defines the available agents.
	Agents []AgentConfig `yaml:"agents"`

	// Chat holds chat provider configuration (omnichat).
	Chat *ChatConfig `yaml:"chat"`
}

// DefaultDaemonConfig returns a DaemonConfig with sensible defaults.
func DefaultDaemonConfig() *DaemonConfig {
	return &DaemonConfig{
		LogLevel: "info",
		Agents:   []AgentConfig{},
	}
}

// LoadDaemonConfig loads configuration from the config file.
// It looks for config.yaml in the data directory.
func LoadDaemonConfig(dataDir string) (*DaemonConfig, error) {
	cfg := DefaultDaemonConfig()

	configPath := filepath.Join(dataDir, "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No config file, return defaults
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

// FindAgentByChannel returns the agent ID for a chat channel.
// channelID format: "provider:chatid" (e.g., "discord:123456789")
func (c *DaemonConfig) FindAgentByChannel(channelID string) (string, bool) {
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
func (c *DaemonConfig) GetAgent(id string) (*AgentConfig, bool) {
	for i := range c.Agents {
		if c.Agents[i].ID == id {
			return &c.Agents[i], true
		}
	}
	return nil, false
}

// HasChatProviders returns true if any chat provider is configured.
func (c *DaemonConfig) HasChatProviders() bool {
	if c.Chat == nil {
		return false
	}
	return c.Chat.Discord != nil || c.Chat.Telegram != nil || c.Chat.WhatsApp != nil
}

// Validate checks the configuration for errors.
func (c *DaemonConfig) Validate() error {
	// Check for duplicate agent IDs
	agentIDs := make(map[string]bool)
	for _, agent := range c.Agents {
		if agent.ID == "" {
			return fmt.Errorf("agent ID is required")
		}
		if agentIDs[agent.ID] {
			return fmt.Errorf("duplicate agent ID: %s", agent.ID)
		}
		agentIDs[agent.ID] = true

		if agent.Type == "" {
			return fmt.Errorf("agent %s: type is required", agent.ID)
		}
		if agent.Type == "tmux" && agent.TmuxSession == "" {
			return fmt.Errorf("agent %s: tmux_session is required for tmux type", agent.ID)
		}
	}

	// Validate chat config
	if c.Chat != nil {
		// Validate Discord config
		if c.Chat.Discord != nil && c.Chat.Discord.Token == "" {
			return fmt.Errorf("chat.discord.token is required")
		}

		// Validate Telegram config
		if c.Chat.Telegram != nil && c.Chat.Telegram.Token == "" {
			return fmt.Errorf("chat.telegram.token is required")
		}

		// Validate WhatsApp config
		if c.Chat.WhatsApp != nil && c.Chat.WhatsApp.DBPath == "" {
			return fmt.Errorf("chat.whatsapp.db_path is required")
		}

		// Check channel mappings reference valid agents
		for _, mapping := range c.Chat.Channels {
			if mapping.ChannelID == "" {
				return fmt.Errorf("chat channel mapping: channel_id is required")
			}
			if mapping.AgentID == "" {
				return fmt.Errorf("chat channel mapping: agent_id is required")
			}
			if !agentIDs[mapping.AgentID] {
				return fmt.Errorf("chat channel %s references unknown agent: %s",
					mapping.ChannelID, mapping.AgentID)
			}
		}
	}

	return nil
}
