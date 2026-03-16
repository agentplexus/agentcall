package tools

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpkit "github.com/plexusone/mcpkit/runtime"

	"github.com/plexusone/agentcomms/internal/daemon"
)

// InboundConfig holds configuration for inbound message tools.
type InboundConfig struct {
	// AgentID is the agent identifier for this session.
	// If empty, uses AGENTCOMMS_AGENT_ID env var or "default".
	AgentID string

	// SocketPath is the daemon socket path.
	// If empty, uses the default path.
	SocketPath string
}

// InboundManager manages inbound message polling from the daemon.
type InboundManager struct {
	agentID    string
	client     *daemon.Client
	lastSeenID string // Track last seen event for pagination
}

// NewInboundManager creates a new inbound message manager.
func NewInboundManager(cfg InboundConfig) *InboundManager {
	agentID := cfg.AgentID
	if agentID == "" {
		agentID = os.Getenv("AGENTCOMMS_AGENT_ID")
	}
	if agentID == "" {
		agentID = "default"
	}

	var client *daemon.Client
	if cfg.SocketPath != "" {
		client = daemon.NewClient(cfg.SocketPath)
	} else {
		client = daemon.DefaultClient()
	}

	return &InboundManager{
		agentID: agentID,
		client:  client,
	}
}

// AgentID returns the configured agent ID.
func (m *InboundManager) AgentID() string {
	return m.agentID
}

// Connect establishes connection to the daemon.
func (m *InboundManager) Connect() error {
	return m.client.Connect()
}

// Close closes the daemon connection.
func (m *InboundManager) Close() error {
	return m.client.Close()
}

// IsConnected returns true if connected to daemon.
func (m *InboundManager) IsConnected() bool {
	return m.client.IsConnected()
}

// ensureConnected connects if not already connected.
func (m *InboundManager) ensureConnected() error {
	if m.client.IsConnected() {
		return nil
	}
	return m.client.Connect()
}

// CheckMessagesInput is the input for the check_messages tool.
type CheckMessagesInput struct {
	AgentID string `json:"agent_id,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

// CheckMessagesOutput is the output of the check_messages tool.
type CheckMessagesOutput struct {
	Messages   []InboundMessage `json:"messages"`
	AgentID    string           `json:"agent_id"`
	HasMore    bool             `json:"has_more"`
	LastSeenID string           `json:"last_seen_id,omitempty"`
}

// InboundMessage represents a message from a human.
type InboundMessage struct {
	ID        string    `json:"id"`
	ChannelID string    `json:"channel_id"`
	Provider  string    `json:"provider"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
}

// GetAgentEventsInput is the input for the get_agent_events tool.
type GetAgentEventsInput struct {
	AgentID string `json:"agent_id,omitempty"`
	SinceID string `json:"since_id,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

// GetAgentEventsOutput is the output of the get_agent_events tool.
type GetAgentEventsOutput struct {
	Events  []daemon.EventInfo `json:"events"`
	AgentID string             `json:"agent_id"`
}

// DaemonStatusInput is the input for the daemon_status tool.
type DaemonStatusInput struct{}

// DaemonStatusOutput is the output of the daemon_status tool.
type DaemonStatusOutput struct {
	Running   bool      `json:"running"`
	StartedAt time.Time `json:"started_at,omitempty"`
	Agents    int       `json:"agents,omitempty"`
	Providers []string  `json:"providers,omitempty"`
}

// RegisterInboundTools registers inbound message MCP tools with the runtime.
func RegisterInboundTools(rt *mcpkit.Runtime, manager *InboundManager) {
	// check_messages - Check for new messages from humans
	mcpkit.AddTool(rt, &mcp.Tool{
		Name:        "check_messages",
		Description: "Check for new messages sent to this agent from humans via chat (Discord, Telegram, WhatsApp). Use this periodically during long tasks to see if the user has sent any instructions or feedback. Returns only human messages (not agent responses).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"agent_id": map[string]any{
					"type":        "string",
					"description": "The agent ID to check messages for. Defaults to the current agent.",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Maximum number of messages to return (default: 10).",
					"default":     10,
				},
			},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in CheckMessagesInput) (*mcp.CallToolResult, CheckMessagesOutput, error) {
		if err := manager.ensureConnected(); err != nil {
			return nil, CheckMessagesOutput{}, fmt.Errorf("daemon not running: %w", err)
		}

		agentID := in.AgentID
		if agentID == "" {
			agentID = manager.agentID
		}

		limit := in.Limit
		if limit <= 0 {
			limit = 10
		}

		// Get events from daemon
		result, err := manager.client.Events(ctx, agentID, limit)
		if err != nil {
			return nil, CheckMessagesOutput{}, fmt.Errorf("failed to get events: %w", err)
		}

		// Filter to only human messages
		var messages []InboundMessage
		for _, evt := range result.Events {
			if evt.Role != "human" {
				continue
			}
			if evt.Type != "human_message" {
				continue
			}

			// Extract text from payload
			text := ""
			if payload, ok := evt.Payload["text"].(string); ok {
				text = payload
			}

			// Extract provider from channel ID
			provider := ""
			if len(evt.ChannelID) > 0 {
				for i, c := range evt.ChannelID {
					if c == ':' {
						provider = evt.ChannelID[:i]
						break
					}
				}
			}

			messages = append(messages, InboundMessage{
				ID:        evt.ID,
				ChannelID: evt.ChannelID,
				Provider:  provider,
				Text:      text,
				Timestamp: evt.Timestamp,
				Type:      evt.Type,
			})
		}

		// Update last seen ID
		if len(result.Events) > 0 {
			manager.lastSeenID = result.Events[0].ID
		}

		return nil, CheckMessagesOutput{
			Messages:   messages,
			AgentID:    agentID,
			HasMore:    len(result.Events) >= limit,
			LastSeenID: manager.lastSeenID,
		}, nil
	})

	// get_agent_events - Get all events (messages, interrupts, etc.)
	mcpkit.AddTool(rt, &mcp.Tool{
		Name:        "get_agent_events",
		Description: "Get recent events for an agent including all message types, interrupts, and status changes. Use this for a complete view of agent activity. For just human messages, use check_messages instead.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"agent_id": map[string]any{
					"type":        "string",
					"description": "The agent ID to get events for. Defaults to the current agent.",
				},
				"since_id": map[string]any{
					"type":        "string",
					"description": "Only return events after this event ID (for pagination).",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Maximum number of events to return (default: 20).",
					"default":     20,
				},
			},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in GetAgentEventsInput) (*mcp.CallToolResult, GetAgentEventsOutput, error) {
		if err := manager.ensureConnected(); err != nil {
			return nil, GetAgentEventsOutput{}, fmt.Errorf("daemon not running: %w", err)
		}

		agentID := in.AgentID
		if agentID == "" {
			agentID = manager.agentID
		}

		limit := in.Limit
		if limit <= 0 {
			limit = 20
		}

		var result *daemon.EventsResult
		var err error

		if in.SinceID != "" {
			result, err = manager.client.EventsSince(ctx, agentID, in.SinceID, limit)
		} else {
			result, err = manager.client.Events(ctx, agentID, limit)
		}

		if err != nil {
			return nil, GetAgentEventsOutput{}, fmt.Errorf("failed to get events: %w", err)
		}

		return nil, GetAgentEventsOutput{
			Events:  result.Events,
			AgentID: agentID,
		}, nil
	})

	// daemon_status - Check if the daemon is running
	mcpkit.AddTool(rt, &mcp.Tool{
		Name:        "daemon_status",
		Description: "Check if the agentcomms daemon is running and get its status. The daemon handles inbound messages from chat platforms (Discord, Telegram, WhatsApp) and routes them to agents.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in DaemonStatusInput) (*mcp.CallToolResult, DaemonStatusOutput, error) {
		if err := manager.ensureConnected(); err != nil {
			// Daemon not running - return status indicating this
			return nil, DaemonStatusOutput{
				Running: false,
			}, nil
		}

		status, err := manager.client.Status(ctx)
		if err != nil {
			return nil, DaemonStatusOutput{Running: false}, nil
		}

		return nil, DaemonStatusOutput{
			Running:   status.Running,
			StartedAt: status.StartedAt,
			Agents:    status.Agents,
			Providers: status.Providers,
		}, nil
	})
}
