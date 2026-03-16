package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/oklog/ulid/v2"
)

// Client provides access to the daemon via Unix socket.
type Client struct {
	socketPath string

	mu   sync.Mutex
	conn net.Conn
}

// NewClient creates a new daemon client.
func NewClient(socketPath string) *Client {
	return &Client{
		socketPath: socketPath,
	}
}

// DefaultClient creates a client using the default socket path.
func DefaultClient() *Client {
	homeDir, _ := os.UserHomeDir()
	socketPath := filepath.Join(homeDir, ".agentcomms", "daemon.sock")
	return NewClient(socketPath)
}

// Connect establishes a connection to the daemon.
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return nil // Already connected
	}

	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}

	c.conn = conn
	return nil
}

// Close closes the connection to the daemon.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil
	}

	err := c.conn.Close()
	c.conn = nil
	return err
}

// IsConnected returns true if connected to the daemon.
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}

// call sends a request and waits for a response.
func (c *Client) call(method string, params any, result any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return fmt.Errorf("not connected to daemon")
	}

	// Generate request ID
	id := ulid.Make().String()

	// Create request
	req, err := NewRequest(id, method, params)
	if err != nil {
		return err
	}

	// Send request
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	if _, err := c.conn.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	// Read response
	reader := bufio.NewReader(c.conn)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var resp Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for error
	if err := resp.Err(); err != nil {
		return err
	}

	// Parse result
	if result != nil {
		if err := resp.ParseResult(result); err != nil {
			return fmt.Errorf("failed to parse result: %w", err)
		}
	}

	return nil
}

// Ping checks if the daemon is responsive.
func (c *Client) Ping(_ context.Context) error {
	var result PingResult
	if err := c.call(MethodPing, nil, &result); err != nil {
		return err
	}
	if !result.Pong {
		return fmt.Errorf("unexpected ping response")
	}
	return nil
}

// Status returns the daemon status.
func (c *Client) Status(_ context.Context) (*StatusResult, error) {
	var result StatusResult
	if err := c.call(MethodStatus, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Agents returns the list of registered agents.
func (c *Client) Agents(_ context.Context) (*AgentsResult, error) {
	var result AgentsResult
	if err := c.call(MethodAgents, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Send sends a message to an agent.
func (c *Client) Send(_ context.Context, agentID, message string) (*SendResult, error) {
	params := SendParams{
		AgentID: agentID,
		Message: message,
	}

	var result SendResult
	if err := c.call(MethodSend, params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Interrupt sends an interrupt signal to an agent.
func (c *Client) Interrupt(_ context.Context, agentID, reason string) (*InterruptResult, error) {
	params := InterruptParams{
		AgentID: agentID,
		Reason:  reason,
	}

	var result InterruptResult
	if err := c.call(MethodInterrupt, params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Events returns recent events for an agent.
func (c *Client) Events(_ context.Context, agentID string, limit int) (*EventsResult, error) {
	params := EventsParams{
		AgentID: agentID,
		Limit:   limit,
	}

	var result EventsResult
	if err := c.call(MethodEvents, params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// EventsSince returns events after a specific event ID.
func (c *Client) EventsSince(_ context.Context, agentID, sinceID string, limit int) (*EventsResult, error) {
	params := EventsParams{
		AgentID: agentID,
		Limit:   limit,
		Since:   sinceID,
	}

	var result EventsResult
	if err := c.call(MethodEvents, params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Reply sends a message from an agent to a chat channel.
func (c *Client) Reply(_ context.Context, channelID, message, agentID string) (*ReplyResult, error) {
	params := ReplyParams{
		ChannelID: channelID,
		Message:   message,
		AgentID:   agentID,
	}

	var result ReplyResult
	if err := c.call(MethodReply, params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Channels returns the list of mapped channels.
func (c *Client) Channels(_ context.Context) (*ChannelsResult, error) {
	var result ChannelsResult
	if err := c.call(MethodChannels, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// IsDaemonRunning checks if the daemon is running by attempting to connect.
func IsDaemonRunning(socketPath string) bool {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// DefaultSocketPath returns the default socket path.
func DefaultSocketPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".agentcomms", "daemon.sock")
}
