package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/plexusone/agentcomms/ent"
	"github.com/plexusone/agentcomms/ent/event"
	"github.com/plexusone/agentcomms/internal/events"
	"github.com/plexusone/agentcomms/internal/router"
	"github.com/plexusone/agentcomms/internal/tenant"
)

// ChatSender is the interface for sending outbound chat messages.
type ChatSender interface {
	SendMessage(ctx context.Context, channelID, content string) error
}

// Server handles IPC requests over a Unix socket.
type Server struct {
	socketPath  string
	listener    net.Listener
	logger      *slog.Logger
	multiTenant bool

	// Dependencies for handling requests
	client     *ent.Client
	router     *router.Router
	daemonCfg  *DaemonConfig
	chatSender ChatSender
	providers  []string
	startedAt  time.Time

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
}

// ServerConfig holds server configuration.
type ServerConfig struct {
	SocketPath  string
	Client      *ent.Client
	Router      *router.Router
	DaemonCfg   *DaemonConfig
	ChatSender  ChatSender
	Providers   []string
	Logger      *slog.Logger
	MultiTenant bool
}

// NewServer creates a new IPC server.
func NewServer(cfg ServerConfig) *Server {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return &Server{
		socketPath:  cfg.SocketPath,
		client:      cfg.Client,
		router:      cfg.Router,
		daemonCfg:   cfg.DaemonCfg,
		chatSender:  cfg.ChatSender,
		providers:   cfg.Providers,
		logger:      cfg.Logger.With("component", "server"),
		startedAt:   time.Now(),
		multiTenant: cfg.MultiTenant,
	}
}

// Start starts the server and begins accepting connections.
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}
	s.running = true
	s.mu.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	// Remove existing socket file
	if err := os.Remove(s.socketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing socket: %w", err)
	}

	// Create listener
	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		return fmt.Errorf("failed to listen on socket: %w", err)
	}
	s.listener = listener

	// Set socket permissions (readable/writable by owner only)
	if err := os.Chmod(s.socketPath, 0600); err != nil {
		s.logger.Warn("failed to set socket permissions", "error", err)
	}

	s.logger.Info("server started", "socket", s.socketPath)

	// Accept connections
	go s.acceptLoop(ctx)

	// Wait for context cancellation
	<-ctx.Done()

	return s.shutdown()
}

// Stop stops the server.
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	if s.cancel != nil {
		s.cancel()
	}

	return nil
}

// shutdown performs cleanup.
func (s *Server) shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info("shutting down server")

	if s.listener != nil {
		s.listener.Close()
	}

	// Remove socket file
	os.Remove(s.socketPath)

	s.running = false
	s.logger.Info("server stopped")

	return nil
}

// acceptLoop accepts incoming connections.
func (s *Server) acceptLoop(ctx context.Context) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				s.logger.Error("failed to accept connection", "error", err)
				continue
			}
		}

		go s.handleConnection(ctx, conn)
	}
}

// handleConnection handles a single client connection.
func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	s.logger.Debug("client connected")

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	for {
		// Read request (one JSON object per line)
		line, err := reader.ReadBytes('\n')
		if err != nil {
			s.logger.Debug("client disconnected", "error", err)
			return
		}

		// Parse request
		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			s.logger.Warn("invalid request", "error", err)
			resp := NewErrorResponse("", ErrCodeInvalidRequest, "invalid JSON")
			if err := s.writeResponse(writer, resp); err != nil {
				s.logger.Error("failed to write error response", "error", err)
				return
			}
			continue
		}

		// Handle request
		resp := s.handleRequest(ctx, &req)

		// Write response
		if err := s.writeResponse(writer, resp); err != nil {
			s.logger.Error("failed to write response", "error", err)
			return
		}
	}
}

// writeResponse writes a response to the connection.
func (s *Server) writeResponse(writer *bufio.Writer, resp *Response) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	if _, err := writer.Write(data); err != nil {
		return err
	}
	if err := writer.WriteByte('\n'); err != nil {
		return err
	}
	return writer.Flush()
}

// handleRequest dispatches a request to the appropriate handler.
func (s *Server) handleRequest(ctx context.Context, req *Request) *Response {
	s.logger.Debug("handling request", "method", req.Method, "id", req.ID, "tenant_id", req.TenantID)

	// In multi-tenant mode, require tenant_id
	if s.multiTenant && req.TenantID == "" {
		return NewErrorResponse(req.ID, ErrCodeInvalidRequest, "tenant_id is required in multi-tenant mode")
	}

	// Add tenant ID to context (defaults to "local" if not set)
	tenantID := req.TenantID
	if tenantID == "" {
		tenantID = tenant.DefaultTenantID
	}
	ctx = tenant.WithTenantID(ctx, tenantID)

	switch req.Method {
	case MethodPing:
		return s.handlePing(req)
	case MethodStatus:
		return s.handleStatus(req)
	case MethodAgents:
		return s.handleAgents(req)
	case MethodSend:
		return s.handleSend(ctx, req)
	case MethodInterrupt:
		return s.handleInterrupt(ctx, req)
	case MethodEvents:
		return s.handleEvents(ctx, req)
	case MethodReply:
		return s.handleReply(ctx, req)
	case MethodChannels:
		return s.handleChannels(req)
	case MethodAgentMessage:
		return s.handleAgentMessage(ctx, req)
	default:
		return NewErrorResponse(req.ID, ErrCodeMethodNotFound, "method not found: "+req.Method)
	}
}

// handlePing handles the ping method.
func (s *Server) handlePing(req *Request) *Response {
	resp, _ := NewResponse(req.ID, PingResult{Pong: true})
	return resp
}

// handleStatus handles the status method.
func (s *Server) handleStatus(req *Request) *Response {
	result := StatusResult{
		Running:   true,
		StartedAt: s.startedAt,
		Agents:    len(s.router.Agents()),
		Providers: s.providers,
	}

	resp, _ := NewResponse(req.ID, result)
	return resp
}

// handleAgents handles the agents method.
func (s *Server) handleAgents(req *Request) *Response {
	agentIDs := s.router.Agents()
	agentStatuses := s.router.AgentStatuses()
	agents := make([]AgentInfo, 0, len(agentIDs))

	for _, id := range agentIDs {
		agentCfg, ok := s.daemonCfg.GetAgent(id)
		if !ok {
			continue
		}

		info := AgentInfo{
			ID:     id,
			Type:   agentCfg.Type,
			Status: agentStatuses[id],
		}

		if agentCfg.Type == "tmux" {
			info.Target = fmt.Sprintf("tmux:%s:%s", agentCfg.TmuxSession, agentCfg.TmuxPane)
		}

		agents = append(agents, info)
	}

	resp, _ := NewResponse(req.ID, AgentsResult{Agents: agents})
	return resp
}

// handleSend handles the send method.
func (s *Server) handleSend(ctx context.Context, req *Request) *Response {
	var params SendParams
	if err := req.ParseParams(&params); err != nil {
		return NewErrorResponse(req.ID, ErrCodeInvalidParams, "invalid params: "+err.Error())
	}

	if params.AgentID == "" {
		return NewErrorResponse(req.ID, ErrCodeInvalidParams, "agent_id is required")
	}
	if params.Message == "" {
		return NewErrorResponse(req.ID, ErrCodeInvalidParams, "message is required")
	}

	// Check if agent exists
	if !s.router.HasAgent(params.AgentID) {
		return NewErrorResponse(req.ID, ErrCodeNotFound, "agent not found: "+params.AgentID)
	}

	// Create event
	evt, err := s.client.Event.Create().
		SetID(events.NewID()).
		SetTenantID(tenant.FromContext(ctx)).
		SetAgentID(params.AgentID).
		SetChannelID("cli:local").
		SetType(event.TypeHumanMessage).
		SetRole(event.RoleHuman).
		SetPayload(map[string]any{
			"text":   params.Message,
			"source": "cli",
		}).
		Save(ctx)

	if err != nil {
		s.logger.Error("failed to create event", "error", err)
		return NewErrorResponse(req.ID, ErrCodeInternal, "failed to create event")
	}

	// Dispatch to router
	if err := s.router.Dispatch(params.AgentID, evt); err != nil {
		s.logger.Error("failed to dispatch event", "error", err)
		return NewErrorResponse(req.ID, ErrCodeInternal, "failed to dispatch event")
	}

	resp, _ := NewResponse(req.ID, SendResult{
		EventID:   evt.ID,
		Delivered: true,
	})
	return resp
}

// handleInterrupt handles the interrupt method.
func (s *Server) handleInterrupt(ctx context.Context, req *Request) *Response {
	var params InterruptParams
	if err := req.ParseParams(&params); err != nil {
		return NewErrorResponse(req.ID, ErrCodeInvalidParams, "invalid params: "+err.Error())
	}

	if params.AgentID == "" {
		return NewErrorResponse(req.ID, ErrCodeInvalidParams, "agent_id is required")
	}

	// Check if agent exists
	if !s.router.HasAgent(params.AgentID) {
		return NewErrorResponse(req.ID, ErrCodeNotFound, "agent not found: "+params.AgentID)
	}

	// Create interrupt event
	evt, err := s.client.Event.Create().
		SetID(events.NewID()).
		SetTenantID(tenant.FromContext(ctx)).
		SetAgentID(params.AgentID).
		SetChannelID("cli:local").
		SetType(event.TypeInterrupt).
		SetRole(event.RoleHuman).
		SetPayload(map[string]any{
			"reason": params.Reason,
			"source": "cli",
		}).
		Save(ctx)

	if err != nil {
		s.logger.Error("failed to create event", "error", err)
		return NewErrorResponse(req.ID, ErrCodeInternal, "failed to create event")
	}

	// Dispatch to router
	if err := s.router.Dispatch(params.AgentID, evt); err != nil {
		s.logger.Error("failed to dispatch event", "error", err)
		return NewErrorResponse(req.ID, ErrCodeInternal, "failed to dispatch event")
	}

	resp, _ := NewResponse(req.ID, InterruptResult{
		EventID:   evt.ID,
		Delivered: true,
	})
	return resp
}

// handleEvents handles the events method.
func (s *Server) handleEvents(ctx context.Context, req *Request) *Response {
	var params EventsParams
	if err := req.ParseParams(&params); err != nil {
		return NewErrorResponse(req.ID, ErrCodeInvalidParams, "invalid params: "+err.Error())
	}

	if params.AgentID == "" {
		return NewErrorResponse(req.ID, ErrCodeInvalidParams, "agent_id is required")
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}

	// Query events
	query := s.client.Event.Query().
		Where(event.AgentID(params.AgentID)).
		Order(ent.Desc(event.FieldTimestamp)).
		Limit(limit)

	if params.Since != "" {
		// Get the timestamp of the "since" event
		sinceEvt, err := s.client.Event.Get(ctx, params.Since)
		if err == nil {
			query = query.Where(event.TimestampLT(sinceEvt.Timestamp))
		}
	}

	evts, err := query.All(ctx)
	if err != nil {
		s.logger.Error("failed to query events", "error", err)
		return NewErrorResponse(req.ID, ErrCodeInternal, "failed to query events")
	}

	// Convert to EventInfo
	eventInfos := make([]EventInfo, len(evts))
	for i, evt := range evts {
		eventInfos[i] = EventInfo{
			ID:        evt.ID,
			AgentID:   evt.AgentID,
			ChannelID: evt.ChannelID,
			Type:      string(evt.Type),
			Role:      string(evt.Role),
			Timestamp: evt.Timestamp,
			Status:    string(evt.Status),
			Payload:   evt.Payload,
		}
	}

	resp, _ := NewResponse(req.ID, EventsResult{Events: eventInfos})
	return resp
}

// handleReply handles the reply method (outbound message to chat channel).
func (s *Server) handleReply(ctx context.Context, req *Request) *Response {
	var params ReplyParams
	if err := req.ParseParams(&params); err != nil {
		return NewErrorResponse(req.ID, ErrCodeInvalidParams, "invalid params: "+err.Error())
	}

	if params.ChannelID == "" {
		return NewErrorResponse(req.ID, ErrCodeInvalidParams, "channel_id is required")
	}
	if params.Message == "" {
		return NewErrorResponse(req.ID, ErrCodeInvalidParams, "message is required")
	}

	// Check if chat sender is available
	if s.chatSender == nil {
		return NewErrorResponse(req.ID, ErrCodeInternal, "chat not configured")
	}

	// Determine agent ID (from params or from channel mapping)
	agentID := params.AgentID
	if agentID == "" {
		// Try to find agent from channel mapping
		if mappedAgent, ok := s.daemonCfg.FindAgentByChannel(params.ChannelID); ok {
			agentID = mappedAgent
		} else {
			agentID = "unknown"
		}
	}

	// Create event for tracking
	evt, err := s.client.Event.Create().
		SetID(events.NewID()).
		SetTenantID(tenant.FromContext(ctx)).
		SetAgentID(agentID).
		SetChannelID(params.ChannelID).
		SetType(event.TypeAgentMessage).
		SetRole(event.RoleAgent).
		SetPayload(map[string]any{
			"text":   params.Message,
			"source": "cli",
		}).
		Save(ctx)

	if err != nil {
		s.logger.Error("failed to create event", "error", err)
		return NewErrorResponse(req.ID, ErrCodeInternal, "failed to create event")
	}

	// Send message via chat transport
	if err := s.chatSender.SendMessage(ctx, params.ChannelID, params.Message); err != nil {
		s.logger.Error("failed to send message", "error", err, "channel_id", params.ChannelID)

		// Update event status to failed
		_, _ = s.client.Event.UpdateOneID(evt.ID).
			SetStatus(event.StatusFailed).
			Save(ctx)

		return NewErrorResponse(req.ID, ErrCodeInternal, "failed to send message: "+err.Error())
	}

	// Update event status to delivered
	_, _ = s.client.Event.UpdateOneID(evt.ID).
		SetStatus(event.StatusDelivered).
		Save(ctx)

	s.logger.Info("sent reply",
		"channel_id", params.ChannelID,
		"event_id", evt.ID,
	)

	resp, _ := NewResponse(req.ID, ReplyResult{
		EventID:   evt.ID,
		Delivered: true,
	})
	return resp
}

// handleChannels handles the channels method.
func (s *Server) handleChannels(req *Request) *Response {
	if s.daemonCfg.Chat == nil {
		resp, _ := NewResponse(req.ID, ChannelsResult{Channels: []ChannelInfo{}})
		return resp
	}

	channels := make([]ChannelInfo, 0, len(s.daemonCfg.Chat.Channels))
	for _, mapping := range s.daemonCfg.Chat.Channels {
		// Extract provider from channel ID (format: "provider:chatid")
		provider := ""
		if idx := strings.Index(mapping.ChannelID, ":"); idx > 0 {
			provider = mapping.ChannelID[:idx]
		}

		channels = append(channels, ChannelInfo{
			ChannelID: mapping.ChannelID,
			AgentID:   mapping.AgentID,
			Provider:  provider,
		})
	}

	resp, _ := NewResponse(req.ID, ChannelsResult{Channels: channels})
	return resp
}

// handleAgentMessage handles the agent_message method.
// This sends a message from one agent to another.
func (s *Server) handleAgentMessage(ctx context.Context, req *Request) *Response {
	var params AgentMessageParams
	if err := req.ParseParams(&params); err != nil {
		return NewErrorResponse(req.ID, ErrCodeInvalidParams, "invalid params: "+err.Error())
	}

	if params.FromAgentID == "" {
		return NewErrorResponse(req.ID, ErrCodeInvalidParams, "from_agent_id is required")
	}
	if params.ToAgentID == "" {
		return NewErrorResponse(req.ID, ErrCodeInvalidParams, "to_agent_id is required")
	}
	if params.Message == "" {
		return NewErrorResponse(req.ID, ErrCodeInvalidParams, "message is required")
	}

	// Check if destination agent exists
	if !s.router.HasAgent(params.ToAgentID) {
		return NewErrorResponse(req.ID, ErrCodeNotFound, "agent not found: "+params.ToAgentID)
	}

	// Create event with source_agent_id for agent-to-agent messaging
	evt, err := s.client.Event.Create().
		SetID(events.NewID()).
		SetTenantID(tenant.FromContext(ctx)).
		SetAgentID(params.ToAgentID).
		SetSourceAgentID(params.FromAgentID).
		SetChannelID("agent:" + params.FromAgentID).
		SetType(event.TypeAgentMessage).
		SetRole(event.RoleAgent).
		SetPayload(map[string]any{
			"text":   params.Message,
			"source": "agent",
		}).
		Save(ctx)

	if err != nil {
		s.logger.Error("failed to create agent message event", "error", err)
		return NewErrorResponse(req.ID, ErrCodeInternal, "failed to create event")
	}

	// Dispatch to router
	if err := s.router.Dispatch(params.ToAgentID, evt); err != nil {
		s.logger.Error("failed to dispatch agent message", "error", err)
		return NewErrorResponse(req.ID, ErrCodeInternal, "failed to dispatch event")
	}

	s.logger.Info("dispatched agent message",
		"from", params.FromAgentID,
		"to", params.ToAgentID,
		"event_id", evt.ID,
	)

	resp, _ := NewResponse(req.ID, AgentMessageResult{
		EventID:   evt.ID,
		Delivered: true,
	})
	return resp
}
