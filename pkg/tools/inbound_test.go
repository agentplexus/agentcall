package tools

import (
	"os"
	"testing"
)

func TestNewInboundManager(t *testing.T) {
	tests := []struct {
		name      string
		cfg       InboundConfig
		envVar    string
		wantAgent string
	}{
		{
			name:      "default agent ID when nothing set",
			cfg:       InboundConfig{},
			wantAgent: "default",
		},
		{
			name:      "use config agent ID",
			cfg:       InboundConfig{AgentID: "my-agent"},
			wantAgent: "my-agent",
		},
		{
			name:      "use env var when config empty",
			cfg:       InboundConfig{},
			envVar:    "env-agent",
			wantAgent: "env-agent",
		},
		{
			name:      "config takes precedence over env",
			cfg:       InboundConfig{AgentID: "config-agent"},
			envVar:    "env-agent",
			wantAgent: "config-agent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set env var if specified
			if tt.envVar != "" {
				os.Setenv("AGENTCOMMS_AGENT_ID", tt.envVar)
				defer os.Unsetenv("AGENTCOMMS_AGENT_ID")
			} else {
				os.Unsetenv("AGENTCOMMS_AGENT_ID")
			}

			manager := NewInboundManager(tt.cfg)
			if manager.AgentID() != tt.wantAgent {
				t.Errorf("AgentID() = %q, want %q", manager.AgentID(), tt.wantAgent)
			}
		})
	}
}

func TestInboundManager_NotConnected(t *testing.T) {
	manager := NewInboundManager(InboundConfig{
		SocketPath: "/nonexistent/path.sock",
	})

	if manager.IsConnected() {
		t.Error("expected IsConnected() = false for new manager")
	}

	// Connect should fail for non-existent socket
	err := manager.Connect()
	if err == nil {
		t.Error("expected error when connecting to non-existent socket")
		manager.Close()
	}
}
