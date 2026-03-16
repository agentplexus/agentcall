package router

import (
	"context"
	"database/sql"
	"os/exec"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"

	"github.com/plexusone/agentcomms/ent"
	"github.com/plexusone/agentcomms/ent/event"
	"github.com/plexusone/agentcomms/internal/bridge"
	"github.com/plexusone/agentcomms/internal/events"
)

// newTestClient creates a test Ent client with an in-memory SQLite database.
func newTestClient(t *testing.T) *ent.Client {
	t.Helper()

	// Use DSN with foreign keys enabled
	db, err := sql.Open("sqlite", "file::memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := ent.NewClient(ent.Driver(drv))

	// Run migrations
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	t.Cleanup(func() {
		client.Close()
	})

	return client
}

func TestRouterDispatch(t *testing.T) {
	// Skip if tmux is not available
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}

	// Create test session
	sessionName := "agentcomms-router-test"
	_ = exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	defer func() {
		_ = exec.Command("tmux", "kill-session", "-t", sessionName).Run()
	}()

	// Create test database
	client := newTestClient(t)

	// Create router
	r := New(client, nil)

	// Create tmux adapter
	adapter, err := bridge.NewTmuxAdapter(bridge.TmuxConfig{
		Session: sessionName,
		Pane:    "0",
	})
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	ctx := context.Background()

	// Register agent
	if err := r.RegisterAgent(ctx, "test-agent", adapter); err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	// Create test event
	evt, err := client.Event.Create().
		SetID(events.NewID()).
		SetAgentID("test-agent").
		SetChannelID("test:channel").
		SetType(event.TypeHumanMessage).
		SetRole(event.RoleHuman).
		SetPayload(map[string]any{"text": "echo hello from router test"}).
		Save(ctx)
	if err != nil {
		t.Fatalf("failed to create event: %v", err)
	}

	// Dispatch event
	if err := r.Dispatch("test-agent", evt); err != nil {
		t.Errorf("Dispatch failed: %v", err)
	}

	// Wait for async processing
	time.Sleep(500 * time.Millisecond)

	// Verify event status was updated
	updated, err := client.Event.Get(ctx, evt.ID)
	if err != nil {
		t.Fatalf("failed to get event: %v", err)
	}

	if updated.Status != event.StatusDelivered {
		t.Errorf("expected status %s, got %s", event.StatusDelivered, updated.Status)
	}

	// Stop router
	r.Stop(ctx)
}

func TestRouterRegisterUnregister(t *testing.T) {
	client := newTestClient(t)

	r := New(client, nil)
	ctx := context.Background()

	// Create mock adapter
	adapter := &mockAdapter{}

	// Register
	if err := r.RegisterAgent(ctx, "test", adapter); err != nil {
		t.Errorf("RegisterAgent failed: %v", err)
	}

	if !r.HasAgent("test") {
		t.Error("expected agent to be registered")
	}

	// Duplicate registration should fail
	if err := r.RegisterAgent(ctx, "test", adapter); err == nil {
		t.Error("expected error for duplicate registration")
	}

	// Unregister
	if err := r.UnregisterAgent(ctx, "test"); err != nil {
		t.Errorf("UnregisterAgent failed: %v", err)
	}

	if r.HasAgent("test") {
		t.Error("expected agent to be unregistered")
	}

	// Unregister non-existent should fail
	if err := r.UnregisterAgent(ctx, "nonexistent"); err == nil {
		t.Error("expected error for non-existent agent")
	}

	r.Stop(ctx)
}

// mockAdapter is a simple mock adapter for testing.
type mockAdapter struct {
	sent      []string
	interrupt int
}

func (m *mockAdapter) Send(msg string) error {
	m.sent = append(m.sent, msg)
	return nil
}

func (m *mockAdapter) Interrupt() error {
	m.interrupt++
	return nil
}

func (m *mockAdapter) Close() error {
	return nil
}

func TestRouterAgentStatuses(t *testing.T) {
	client := newTestClient(t)
	r := New(client, nil)
	ctx := context.Background()

	// Initially no agents
	statuses := r.AgentStatuses()
	if len(statuses) != 0 {
		t.Errorf("expected 0 statuses, got %d", len(statuses))
	}

	// Register agent
	adapter := &mockAdapter{}
	if err := r.RegisterAgent(ctx, "test-agent", adapter); err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}

	// Should have one online agent
	statuses = r.AgentStatuses()
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if statuses["test-agent"] != "online" {
		t.Errorf("expected status 'online', got %q", statuses["test-agent"])
	}

	// Register another agent
	adapter2 := &mockAdapter{}
	if err := r.RegisterAgent(ctx, "test-agent-2", adapter2); err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}

	// Should have two online agents
	statuses = r.AgentStatuses()
	if len(statuses) != 2 {
		t.Fatalf("expected 2 statuses, got %d", len(statuses))
	}
	if statuses["test-agent"] != "online" || statuses["test-agent-2"] != "online" {
		t.Errorf("expected both agents online, got %v", statuses)
	}

	// Unregister one
	if err := r.UnregisterAgent(ctx, "test-agent"); err != nil {
		t.Fatalf("UnregisterAgent failed: %v", err)
	}

	// Should have one agent remaining
	statuses = r.AgentStatuses()
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if _, ok := statuses["test-agent"]; ok {
		t.Error("expected test-agent to be removed")
	}

	r.Stop(ctx)
}
