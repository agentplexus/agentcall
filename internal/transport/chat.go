// Package transport provides inbound message transports for AgentComms.
package transport

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/plexusone/omnichat/provider"

	"github.com/plexusone/agentcomms/ent"
	"github.com/plexusone/agentcomms/ent/event"
	"github.com/plexusone/agentcomms/internal/events"
	"github.com/plexusone/agentcomms/internal/router"
)

// ChannelResolver resolves chat channel IDs to agent IDs.
type ChannelResolver interface {
	FindAgentByChannel(channelID string) (agentID string, found bool)
}

// ChatTransport handles inbound messages from chat providers via omnichat.
type ChatTransport struct {
	chatRouter *provider.Router
	client     *ent.Client
	router     *router.Router
	resolver   ChannelResolver
	logger     *slog.Logger

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
}

// NewChatTransport creates a new chat transport using omnichat.
func NewChatTransport(
	chatRouter *provider.Router,
	client *ent.Client,
	router *router.Router,
	resolver ChannelResolver,
	logger *slog.Logger,
) *ChatTransport {
	if logger == nil {
		logger = slog.Default()
	}

	return &ChatTransport{
		chatRouter: chatRouter,
		client:     client,
		router:     router,
		resolver:   resolver,
		logger:     logger.With("transport", "chat"),
	}
}

// Start begins listening for messages from all registered providers.
func (t *ChatTransport) Start(ctx context.Context) error {
	t.mu.Lock()
	if t.running {
		t.mu.Unlock()
		return fmt.Errorf("chat transport already running")
	}
	t.running = true
	t.mu.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	t.cancel = cancel

	// Register message handler for all providers
	t.chatRouter.OnMessage(provider.All(), t.handleMessage)

	// Connect all providers
	if err := t.chatRouter.ConnectAll(ctx); err != nil {
		t.mu.Lock()
		t.running = false
		t.mu.Unlock()
		return fmt.Errorf("failed to connect providers: %w", err)
	}

	t.logger.Info("chat transport started",
		"providers", t.chatRouter.ListProviders(),
	)

	// Wait for context cancellation
	<-ctx.Done()

	return t.shutdown(ctx)
}

// Stop stops the chat transport.
func (t *ChatTransport) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running {
		return nil
	}

	if t.cancel != nil {
		t.cancel()
	}

	return nil
}

// shutdown performs cleanup.
func (t *ChatTransport) shutdown(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.logger.Info("shutting down chat transport")

	if err := t.chatRouter.DisconnectAll(ctx); err != nil {
		t.logger.Error("failed to disconnect providers", "error", err)
	}

	t.running = false
	t.logger.Info("chat transport stopped")

	return nil
}

// handleMessage processes incoming chat messages from any provider.
func (t *ChatTransport) handleMessage(ctx context.Context, msg provider.IncomingMessage) error {
	// Build channel ID in format "provider:chatid"
	channelID := fmt.Sprintf("%s:%s", msg.ProviderName, msg.ChatID)

	// Find the agent for this channel
	agentID, found := t.resolver.FindAgentByChannel(channelID)
	if !found {
		// Channel not mapped to any agent, ignore
		t.logger.Debug("ignoring message from unmapped channel",
			"channel_id", channelID,
			"provider", msg.ProviderName,
		)
		return nil
	}

	// Check if agent is registered
	if !t.router.HasAgent(agentID) {
		t.logger.Warn("agent not registered, cannot deliver message",
			"agent_id", agentID,
			"channel_id", channelID,
		)
		return nil
	}

	t.logger.Info("received chat message",
		"channel_id", channelID,
		"provider", msg.ProviderName,
		"sender", msg.SenderName,
		"agent_id", agentID,
		"content_length", len(msg.Content),
	)

	// Create event
	evt, err := t.client.Event.Create().
		SetID(events.NewID()).
		SetAgentID(agentID).
		SetChannelID(channelID).
		SetType(event.TypeHumanMessage).
		SetRole(event.RoleHuman).
		SetPayload(map[string]any{
			"text":        msg.Content,
			"sender_id":   msg.SenderID,
			"sender_name": msg.SenderName,
			"message_id":  msg.ID,
			"provider":    msg.ProviderName,
			"chat_id":     msg.ChatID,
			"chat_type":   string(msg.ChatType),
			"reply_to":    msg.ReplyTo,
		}).
		Save(ctx)

	if err != nil {
		t.logger.Error("failed to create event",
			"error", err,
			"channel_id", channelID,
		)
		return err
	}

	t.logger.Debug("created event",
		"event_id", evt.ID,
		"agent_id", agentID,
	)

	// Dispatch to router
	if err := t.router.Dispatch(agentID, evt); err != nil {
		t.logger.Error("failed to dispatch event",
			"error", err,
			"event_id", evt.ID,
			"agent_id", agentID,
		)
		return err
	}

	t.logger.Debug("dispatched event to router",
		"event_id", evt.ID,
		"agent_id", agentID,
	)

	return nil
}

// Router returns the underlying omnichat router for outbound messages.
func (t *ChatTransport) Router() *provider.Router {
	return t.chatRouter
}

// SendMessage sends a message to a chat channel.
// channelID should be in format "provider:chatid".
func (t *ChatTransport) SendMessage(ctx context.Context, channelID, content string) error {
	providerName, chatID, err := parseChannelID(channelID)
	if err != nil {
		return err
	}

	return t.chatRouter.Send(ctx, providerName, chatID, provider.OutgoingMessage{
		Content: content,
	})
}

// parseChannelID parses a channel ID in format "provider:chatid".
func parseChannelID(channelID string) (providerName, chatID string, err error) {
	for i, c := range channelID {
		if c == ':' {
			return channelID[:i], channelID[i+1:], nil
		}
	}
	return "", "", fmt.Errorf("invalid channel ID format: %s (expected provider:chatid)", channelID)
}
