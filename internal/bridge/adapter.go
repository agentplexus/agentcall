// Package bridge provides adapters for connecting to agent runtimes.
package bridge

import "io"

// Adapter is the interface for agent runtime adapters.
// Adapters handle sending messages and interrupts to agents.
type Adapter interface {
	// Send sends a message to the agent.
	Send(message string) error

	// Interrupt sends an interrupt signal (Ctrl-C) to the agent.
	Interrupt() error

	// Close releases any resources held by the adapter.
	io.Closer
}

// Config holds common adapter configuration.
type Config struct {
	// Type is the adapter type (e.g., "tmux", "process").
	Type string `json:"type"`
}
