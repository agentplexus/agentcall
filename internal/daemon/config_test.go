package daemon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDaemonConfig_NoFile(t *testing.T) {
	// Test loading config when file doesn't exist
	tmpDir := t.TempDir()

	cfg, err := LoadDaemonConfig(tmpDir)
	if err != nil {
		t.Errorf("LoadDaemonConfig() error = %v, expected nil", err)
	}

	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	if cfg.LogLevel != "info" {
		t.Errorf("expected default log_level 'info', got %q", cfg.LogLevel)
	}
}

func TestLoadDaemonConfig_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `
log_level: debug
agents:
  - id: agent1
    type: tmux
    tmux_session: claude
    tmux_pane: "0"
chat:
  discord:
    token: test-token
    guild_id: "123456"
  channels:
    - channel_id: "discord:111"
      agent_id: agent1
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadDaemonConfig(tmpDir)
	if err != nil {
		t.Errorf("LoadDaemonConfig() error = %v", err)
	}

	if cfg.LogLevel != "debug" {
		t.Errorf("expected log_level 'debug', got %q", cfg.LogLevel)
	}

	if len(cfg.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(cfg.Agents))
	}

	if cfg.Agents[0].ID != "agent1" {
		t.Errorf("expected agent ID 'agent1', got %q", cfg.Agents[0].ID)
	}

	if cfg.Chat == nil {
		t.Fatal("expected chat config")
	}

	if cfg.Chat.Discord == nil {
		t.Fatal("expected discord config")
	}

	if cfg.Chat.Discord.Token != "test-token" {
		t.Errorf("expected discord token 'test-token', got %q", cfg.Chat.Discord.Token)
	}
}

func TestDaemonConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *DaemonConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty config is valid",
			config:  &DaemonConfig{},
			wantErr: false,
		},
		{
			name: "valid agent config",
			config: &DaemonConfig{
				Agents: []AgentConfig{
					{ID: "agent1", Type: "tmux", TmuxSession: "claude"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing agent ID",
			config: &DaemonConfig{
				Agents: []AgentConfig{
					{Type: "tmux", TmuxSession: "claude"},
				},
			},
			wantErr: true,
			errMsg:  "agent ID is required",
		},
		{
			name: "duplicate agent ID",
			config: &DaemonConfig{
				Agents: []AgentConfig{
					{ID: "agent1", Type: "tmux", TmuxSession: "claude"},
					{ID: "agent1", Type: "tmux", TmuxSession: "other"},
				},
			},
			wantErr: true,
			errMsg:  "duplicate agent ID",
		},
		{
			name: "tmux agent missing session",
			config: &DaemonConfig{
				Agents: []AgentConfig{
					{ID: "agent1", Type: "tmux"},
				},
			},
			wantErr: true,
			errMsg:  "tmux_session is required",
		},
		{
			name: "discord config without token",
			config: &DaemonConfig{
				Chat: &ChatConfig{
					Discord: &DiscordConfig{},
				},
			},
			wantErr: true,
			errMsg:  "chat.discord.token is required",
		},
		{
			name: "telegram config without token",
			config: &DaemonConfig{
				Chat: &ChatConfig{
					Telegram: &TelegramConfig{},
				},
			},
			wantErr: true,
			errMsg:  "chat.telegram.token is required",
		},
		{
			name: "whatsapp config without db_path",
			config: &DaemonConfig{
				Chat: &ChatConfig{
					WhatsApp: &WhatsAppConfig{},
				},
			},
			wantErr: true,
			errMsg:  "chat.whatsapp.db_path is required",
		},
		{
			name: "chat channel references unknown agent",
			config: &DaemonConfig{
				Agents: []AgentConfig{
					{ID: "agent1", Type: "tmux", TmuxSession: "claude"},
				},
				Chat: &ChatConfig{
					Discord: &DiscordConfig{Token: "test-token"},
					Channels: []ChannelMapping{
						{ChannelID: "discord:111", AgentID: "unknown"},
					},
				},
			},
			wantErr: true,
			errMsg:  "references unknown agent",
		},
		{
			name: "valid full config",
			config: &DaemonConfig{
				Agents: []AgentConfig{
					{ID: "agent1", Type: "tmux", TmuxSession: "claude"},
				},
				Chat: &ChatConfig{
					Discord: &DiscordConfig{Token: "test-token"},
					Channels: []ChannelMapping{
						{ChannelID: "discord:111", AgentID: "agent1"},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %q, expected to contain %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("Validate() unexpected error = %v", err)
			}
		})
	}
}

func TestDaemonConfig_FindAgentByChannel(t *testing.T) {
	cfg := &DaemonConfig{
		Chat: &ChatConfig{
			Channels: []ChannelMapping{
				{ChannelID: "discord:111", AgentID: "agent1"},
				{ChannelID: "telegram:222", AgentID: "agent2"},
			},
		},
	}

	// Test found
	agentID, found := cfg.FindAgentByChannel("discord:111")
	if !found {
		t.Error("expected to find agent for channel discord:111")
	}
	if agentID != "agent1" {
		t.Errorf("expected agent1, got %q", agentID)
	}

	// Test not found
	_, found = cfg.FindAgentByChannel("discord:999")
	if found {
		t.Error("expected not to find agent for channel discord:999")
	}

	// Test nil chat config
	cfg2 := &DaemonConfig{}
	_, found = cfg2.FindAgentByChannel("discord:111")
	if found {
		t.Error("expected not to find agent with nil chat config")
	}
}

func TestDaemonConfig_GetAgent(t *testing.T) {
	cfg := &DaemonConfig{
		Agents: []AgentConfig{
			{ID: "agent1", Type: "tmux", TmuxSession: "claude"},
			{ID: "agent2", Type: "tmux", TmuxSession: "other"},
		},
	}

	// Test found
	agent, found := cfg.GetAgent("agent1")
	if !found {
		t.Error("expected to find agent1")
	}
	if agent.TmuxSession != "claude" {
		t.Errorf("expected session 'claude', got %q", agent.TmuxSession)
	}

	// Test not found
	_, found = cfg.GetAgent("unknown")
	if found {
		t.Error("expected not to find unknown agent")
	}
}

func TestDaemonConfig_HasChatProviders(t *testing.T) {
	tests := []struct {
		name   string
		config *DaemonConfig
		want   bool
	}{
		{
			name:   "nil chat config",
			config: &DaemonConfig{},
			want:   false,
		},
		{
			name: "empty chat config",
			config: &DaemonConfig{
				Chat: &ChatConfig{},
			},
			want: false,
		},
		{
			name: "discord configured",
			config: &DaemonConfig{
				Chat: &ChatConfig{
					Discord: &DiscordConfig{Token: "test"},
				},
			},
			want: true,
		},
		{
			name: "telegram configured",
			config: &DaemonConfig{
				Chat: &ChatConfig{
					Telegram: &TelegramConfig{Token: "test"},
				},
			},
			want: true,
		},
		{
			name: "whatsapp configured",
			config: &DaemonConfig{
				Chat: &ChatConfig{
					WhatsApp: &WhatsAppConfig{DBPath: "/tmp/wa.db"},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.HasChatProviders(); got != tt.want {
				t.Errorf("HasChatProviders() = %v, want %v", got, tt.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && searchString(s, substr)))
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
