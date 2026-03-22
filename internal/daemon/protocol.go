package daemon

import (
	"encoding/json"
	"fmt"
	"time"
)

// Request represents a client request to the daemon.
type Request struct {
	// ID is a unique request identifier for correlation.
	ID string `json:"id"`

	// Method is the RPC method name.
	Method string `json:"method"`

	// TenantID is the tenant identifier for multi-tenant mode.
	// Optional, defaults to "local" for single-tenant mode.
	TenantID string `json:"tenant_id,omitempty"`

	// Params contains method-specific parameters.
	Params json.RawMessage `json:"params,omitempty"`
}

// Response represents a daemon response to a client.
type Response struct {
	// ID matches the request ID.
	ID string `json:"id"`

	// Result contains the method result (on success).
	Result json.RawMessage `json:"result,omitempty"`

	// Error contains error details (on failure).
	Error *ErrorInfo `json:"error,omitempty"`
}

// ErrorInfo contains error details.
type ErrorInfo struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error codes.
const (
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternal       = -32603
	ErrCodeNotFound       = -32604
)

// Method names.
const (
	MethodPing         = "ping"
	MethodStatus       = "status"
	MethodAgents       = "agents"
	MethodSend         = "send"
	MethodInterrupt    = "interrupt"
	MethodEvents       = "events"
	MethodReply        = "reply"
	MethodChannels     = "channels"
	MethodAgentMessage = "agent_message"
)

// PingResult is the response for the ping method.
type PingResult struct {
	Pong bool `json:"pong"`
}

// StatusResult is the response for the status method.
type StatusResult struct {
	Running   bool      `json:"running"`
	StartedAt time.Time `json:"started_at"`
	Agents    int       `json:"agents"`
	Providers []string  `json:"providers"`
}

// AgentsResult is the response for the agents method.
type AgentsResult struct {
	Agents []AgentInfo `json:"agents"`
}

// AgentInfo contains information about a registered agent.
type AgentInfo struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Target string `json:"target"` // e.g., "tmux:session:pane"
	Status string `json:"status"` // "online" or "offline"
}

// SendParams are the parameters for the send method.
type SendParams struct {
	AgentID string `json:"agent_id"`
	Message string `json:"message"`
}

// SendResult is the response for the send method.
type SendResult struct {
	EventID   string `json:"event_id"`
	Delivered bool   `json:"delivered"`
}

// InterruptParams are the parameters for the interrupt method.
type InterruptParams struct {
	AgentID string `json:"agent_id"`
	Reason  string `json:"reason,omitempty"`
}

// InterruptResult is the response for the interrupt method.
type InterruptResult struct {
	EventID   string `json:"event_id"`
	Delivered bool   `json:"delivered"`
}

// EventsParams are the parameters for the events method.
type EventsParams struct {
	AgentID string `json:"agent_id"`
	Limit   int    `json:"limit,omitempty"`
	Since   string `json:"since,omitempty"` // Event ID to start after
}

// EventsResult is the response for the events method.
type EventsResult struct {
	Events []EventInfo `json:"events"`
}

// EventInfo contains information about an event.
type EventInfo struct {
	ID        string         `json:"id"`
	AgentID   string         `json:"agent_id"`
	ChannelID string         `json:"channel_id"`
	Type      string         `json:"type"`
	Role      string         `json:"role"`
	Timestamp time.Time      `json:"timestamp"`
	Status    string         `json:"status"`
	Payload   map[string]any `json:"payload"`
}

// ReplyParams are the parameters for the reply method.
// This sends a message FROM an agent TO a chat channel.
type ReplyParams struct {
	// ChannelID is the target channel (format: "provider:chatid").
	ChannelID string `json:"channel_id"`

	// Message is the text to send.
	Message string `json:"message"`

	// AgentID is the agent sending the message (for event tracking).
	AgentID string `json:"agent_id,omitempty"`
}

// ReplyResult is the response for the reply method.
type ReplyResult struct {
	EventID   string `json:"event_id"`
	Delivered bool   `json:"delivered"`
}

// ChannelsResult is the response for the channels method.
type ChannelsResult struct {
	Channels []ChannelInfo `json:"channels"`
}

// ChannelInfo contains information about a mapped channel.
type ChannelInfo struct {
	ChannelID string `json:"channel_id"`
	AgentID   string `json:"agent_id"`
	Provider  string `json:"provider"`
}

// AgentMessageParams are the parameters for the agent_message method.
// This sends a message FROM one agent TO another agent.
type AgentMessageParams struct {
	// FromAgentID is the source agent sending the message.
	FromAgentID string `json:"from_agent_id"`

	// ToAgentID is the destination agent to receive the message.
	ToAgentID string `json:"to_agent_id"`

	// Message is the text content to send.
	Message string `json:"message"`
}

// AgentMessageResult is the response for the agent_message method.
type AgentMessageResult struct {
	EventID   string `json:"event_id"`
	Delivered bool   `json:"delivered"`
}

// NewRequest creates a new request with the given method and params.
func NewRequest(id, method string, params any) (*Request, error) {
	req := &Request{
		ID:     id,
		Method: method,
	}

	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		req.Params = data
	}

	return req, nil
}

// NewResponse creates a successful response.
func NewResponse(id string, result any) (*Response, error) {
	resp := &Response{ID: id}

	if result != nil {
		data, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result: %w", err)
		}
		resp.Result = data
	}

	return resp, nil
}

// NewErrorResponse creates an error response.
func NewErrorResponse(id string, code int, message string) *Response {
	return &Response{
		ID: id,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	}
}

// ParseParams unmarshals request params into the target struct.
func (r *Request) ParseParams(target any) error {
	if r.Params == nil {
		return nil
	}
	return json.Unmarshal(r.Params, target)
}

// ParseResult unmarshals response result into the target struct.
func (r *Response) ParseResult(target any) error {
	if r.Result == nil {
		return nil
	}
	return json.Unmarshal(r.Result, target)
}

// IsError returns true if the response contains an error.
func (r *Response) IsError() bool {
	return r.Error != nil
}

// Err returns an error if the response contains an error, nil otherwise.
func (r *Response) Err() error {
	if r.Error == nil {
		return nil
	}
	return fmt.Errorf("daemon error %d: %s", r.Error.Code, r.Error.Message)
}
