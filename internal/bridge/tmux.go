package bridge

import (
	"fmt"
	"os/exec"
	"strings"
)

// TmuxConfig holds configuration for the tmux adapter.
type TmuxConfig struct {
	// Session is the tmux session name.
	Session string `json:"session"`

	// Pane is the tmux pane identifier (e.g., "0", "1", or "0.1" for window.pane).
	Pane string `json:"pane"`
}

// TmuxAdapter sends messages to a tmux pane.
type TmuxAdapter struct {
	config TmuxConfig
	target string // Pre-computed target string (session:pane)
}

// NewTmuxAdapter creates a new tmux adapter.
func NewTmuxAdapter(cfg TmuxConfig) (*TmuxAdapter, error) {
	if cfg.Session == "" {
		return nil, fmt.Errorf("tmux session name is required")
	}
	if cfg.Pane == "" {
		cfg.Pane = "0" // Default to first pane
	}

	target := fmt.Sprintf("%s:%s", cfg.Session, cfg.Pane)

	// Verify the tmux session/pane exists
	if err := checkTmuxTarget(target); err != nil {
		return nil, fmt.Errorf("tmux target not found: %w", err)
	}

	return &TmuxAdapter{
		config: cfg,
		target: target,
	}, nil
}

// Send sends a message to the tmux pane.
func (t *TmuxAdapter) Send(message string) error {
	// Escape the message for tmux send-keys
	// We use literal mode (-l) to avoid key interpretation
	escaped := escapeForTmux(message)

	// G204: target is validated in NewTmuxAdapter, message is escaped
	cmd := exec.Command("tmux", "send-keys", "-t", t.target, "-l", escaped) //nolint:gosec
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	// Send Enter key separately
	cmd = exec.Command("tmux", "send-keys", "-t", t.target, "Enter") //nolint:gosec
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to send Enter: %w", err)
	}

	return nil
}

// Interrupt sends Ctrl-C to the tmux pane.
func (t *TmuxAdapter) Interrupt() error {
	// G204: target is validated in NewTmuxAdapter
	cmd := exec.Command("tmux", "send-keys", "-t", t.target, "C-c") //nolint:gosec
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to send interrupt: %w", err)
	}
	return nil
}

// Close releases resources (no-op for tmux).
func (t *TmuxAdapter) Close() error {
	return nil
}

// Target returns the tmux target string.
func (t *TmuxAdapter) Target() string {
	return t.target
}

// checkTmuxTarget verifies that a tmux target exists.
func checkTmuxTarget(target string) error {
	session := strings.Split(target, ":")[0]
	// G204: session name comes from config, validated at startup
	cmd := exec.Command("tmux", "has-session", "-t", session) //nolint:gosec
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("session does not exist")
	}
	return nil
}

// escapeForTmux escapes a string for safe use with tmux send-keys -l.
// The -l flag treats the string as literal, but we still need to handle
// some edge cases.
func escapeForTmux(s string) string {
	// With -l flag, most characters are safe.
	// However, we should handle newlines specially.
	// For now, replace newlines with spaces to avoid multi-line issues.
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}
