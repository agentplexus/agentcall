package bridge

import (
	"os/exec"
	"testing"
)

func TestTmuxAdapter(t *testing.T) {
	// Skip if tmux is not available
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}

	// Create a test session
	sessionName := "agentcomms-test"

	// Clean up any existing test session
	_ = exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	// Create test session
	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}

	// Clean up after test
	defer func() {
		_ = exec.Command("tmux", "kill-session", "-t", sessionName).Run()
	}()

	// Create adapter
	adapter, err := NewTmuxAdapter(TmuxConfig{
		Session: sessionName,
		Pane:    "0",
	})
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}
	defer adapter.Close()

	// Test send
	t.Run("Send", func(t *testing.T) {
		err := adapter.Send("echo hello from test")
		if err != nil {
			t.Errorf("Send failed: %v", err)
		}
	})

	// Test interrupt
	t.Run("Interrupt", func(t *testing.T) {
		err := adapter.Interrupt()
		if err != nil {
			t.Errorf("Interrupt failed: %v", err)
		}
	})

	// Test send with special characters
	t.Run("SendSpecialChars", func(t *testing.T) {
		err := adapter.Send("echo 'hello world' && echo \"test\"")
		if err != nil {
			t.Errorf("Send with special chars failed: %v", err)
		}
	})
}

func TestTmuxAdapterInvalidSession(t *testing.T) {
	// Skip if tmux is not available
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}

	_, err := NewTmuxAdapter(TmuxConfig{
		Session: "nonexistent-session-12345",
		Pane:    "0",
	})
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}
