package daemon

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"

	"github.com/plexusone/agentcomms/ent"
	"github.com/plexusone/agentcomms/internal/router"
)

// newTestEntClient creates a test Ent client with in-memory SQLite.
func newTestEntClient(t *testing.T) *ent.Client {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := ent.NewClient(ent.Driver(drv))

	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	t.Cleanup(func() {
		client.Close()
	})

	return client
}

func TestServerClientIntegration(t *testing.T) {
	// Create test dependencies
	client := newTestEntClient(t)
	r := router.New(client, nil)

	daemonCfg := &DaemonConfig{
		Agents: []AgentConfig{
			{ID: "test-agent", Type: "tmux", TmuxSession: "test"},
		},
	}

	// Create temporary socket path
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	// Create and start server
	server := NewServer(ServerConfig{
		SocketPath: socketPath,
		Client:     client,
		Router:     r,
		DaemonCfg:  daemonCfg,
		Providers:  []string{"discord"},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server in background
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start(ctx)
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Create client
	daemonClient := NewClient(socketPath)
	if err := daemonClient.Connect(); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer daemonClient.Close()

	// Test ping
	t.Run("Ping", func(t *testing.T) {
		if err := daemonClient.Ping(ctx); err != nil {
			t.Errorf("Ping() error = %v", err)
		}
	})

	// Test status
	t.Run("Status", func(t *testing.T) {
		status, err := daemonClient.Status(ctx)
		if err != nil {
			t.Fatalf("Status() error = %v", err)
		}

		if !status.Running {
			t.Error("expected Running=true")
		}
		if status.Agents != 0 { // No agents registered on router
			t.Errorf("expected 0 agents, got %d", status.Agents)
		}
		if len(status.Providers) != 1 || status.Providers[0] != "discord" {
			t.Errorf("expected providers [discord], got %v", status.Providers)
		}
	})

	// Test agents
	t.Run("Agents", func(t *testing.T) {
		agents, err := daemonClient.Agents(ctx)
		if err != nil {
			t.Fatalf("Agents() error = %v", err)
		}

		// No agents are registered on the router (only in config)
		if len(agents.Agents) != 0 {
			t.Errorf("expected 0 agents (none registered on router), got %d", len(agents.Agents))
		}
	})

	// Test send to non-existent agent
	t.Run("SendNotFound", func(t *testing.T) {
		_, err := daemonClient.Send(ctx, "nonexistent", "hello")
		if err == nil {
			t.Error("expected error for non-existent agent")
		}
	})

	// Test events for non-existent agent (should return empty)
	t.Run("EventsEmpty", func(t *testing.T) {
		events, err := daemonClient.Events(ctx, "test-agent", 10)
		if err != nil {
			t.Fatalf("Events() error = %v", err)
		}

		if len(events.Events) != 0 {
			t.Errorf("expected 0 events, got %d", len(events.Events))
		}
	})

	// Stop server
	cancel()
	select {
	case <-serverErr:
		// Server stopped
	case <-time.After(2 * time.Second):
		t.Error("server did not stop in time")
	}
}

func TestClientNotConnected(t *testing.T) {
	client := NewClient("/nonexistent/path.sock")

	// Operations should fail when not connected
	ctx := context.Background()

	if err := client.Ping(ctx); err == nil {
		t.Error("expected error when not connected")
	}

	_, err := client.Status(ctx)
	if err == nil {
		t.Error("expected error when not connected")
	}
}

func TestIsDaemonRunning(t *testing.T) {
	// Test with non-existent socket
	if IsDaemonRunning("/nonexistent/path.sock") {
		t.Error("expected false for non-existent socket")
	}
}

// mockChatSender implements ChatSender for testing.
type mockChatSender struct {
	messages []struct {
		channelID string
		content   string
	}
	err error
}

func (m *mockChatSender) SendMessage(_ context.Context, channelID, content string) error {
	if m.err != nil {
		return m.err
	}
	m.messages = append(m.messages, struct {
		channelID string
		content   string
	}{channelID, content})
	return nil
}

func TestServerReplyAndChannels(t *testing.T) {
	// Create test dependencies
	client := newTestEntClient(t)
	r := router.New(client, nil)
	mockChat := &mockChatSender{}

	daemonCfg := &DaemonConfig{
		Agents: []AgentConfig{
			{ID: "test-agent", Type: "tmux", TmuxSession: "test"},
		},
		Chat: &ChatConfig{
			Discord: &DiscordConfig{Token: "test"},
			Channels: []ChannelMapping{
				{ChannelID: "discord:123", AgentID: "test-agent"},
				{ChannelID: "telegram:456", AgentID: "test-agent"},
			},
		},
	}

	// Create temporary socket path
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	// Create and start server
	server := NewServer(ServerConfig{
		SocketPath: socketPath,
		Client:     client,
		Router:     r,
		DaemonCfg:  daemonCfg,
		ChatSender: mockChat,
		Providers:  []string{"discord", "telegram"},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server in background
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start(ctx)
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Create client
	daemonClient := NewClient(socketPath)
	if err := daemonClient.Connect(); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer daemonClient.Close()

	// Test channels
	t.Run("Channels", func(t *testing.T) {
		channels, err := daemonClient.Channels(ctx)
		if err != nil {
			t.Fatalf("Channels() error = %v", err)
		}

		if len(channels.Channels) != 2 {
			t.Errorf("expected 2 channels, got %d", len(channels.Channels))
		}

		// Check first channel
		if channels.Channels[0].ChannelID != "discord:123" {
			t.Errorf("expected channel_id 'discord:123', got %q", channels.Channels[0].ChannelID)
		}
		if channels.Channels[0].Provider != "discord" {
			t.Errorf("expected provider 'discord', got %q", channels.Channels[0].Provider)
		}
	})

	// Test reply
	t.Run("Reply", func(t *testing.T) {
		result, err := daemonClient.Reply(ctx, "discord:123", "Hello from agent!", "test-agent")
		if err != nil {
			t.Fatalf("Reply() error = %v", err)
		}

		if result.EventID == "" {
			t.Error("expected non-empty event ID")
		}
		if !result.Delivered {
			t.Error("expected Delivered=true")
		}

		// Verify message was sent
		if len(mockChat.messages) != 1 {
			t.Fatalf("expected 1 message sent, got %d", len(mockChat.messages))
		}
		if mockChat.messages[0].channelID != "discord:123" {
			t.Errorf("expected channel 'discord:123', got %q", mockChat.messages[0].channelID)
		}
		if mockChat.messages[0].content != "Hello from agent!" {
			t.Errorf("expected content 'Hello from agent!', got %q", mockChat.messages[0].content)
		}
	})

	// Test reply without chat sender
	t.Run("ReplyNoChatSender", func(t *testing.T) {
		// Create new server without chat sender
		socketPath2 := filepath.Join(tmpDir, "test2.sock")
		server2 := NewServer(ServerConfig{
			SocketPath: socketPath2,
			Client:     client,
			Router:     r,
			DaemonCfg:  daemonCfg,
			ChatSender: nil, // No chat sender
			Providers:  []string{},
		})

		go func() {
			_ = server2.Start(ctx)
		}()
		time.Sleep(100 * time.Millisecond)

		client2 := NewClient(socketPath2)
		if err := client2.Connect(); err != nil {
			t.Fatalf("failed to connect: %v", err)
		}
		defer client2.Close()

		_, err := client2.Reply(ctx, "discord:123", "test", "")
		if err == nil {
			t.Error("expected error when chat not configured")
		}
	})

	// Stop server
	cancel()
	select {
	case <-serverErr:
		// Server stopped
	case <-time.After(2 * time.Second):
		t.Error("server did not stop in time")
	}
}
