// Package router provides the actor-style event router for agent communication.
package router

import (
	"context"
	"log/slog"

	"github.com/plexusone/agentcomms/ent"
	"github.com/plexusone/agentcomms/ent/event"
	"github.com/plexusone/agentcomms/internal/bridge"
)

// AgentActor handles events for a single agent.
// Each actor runs in its own goroutine and processes events sequentially.
type AgentActor struct {
	id      string
	adapter bridge.Adapter
	client  *ent.Client
	logger  *slog.Logger

	inbox  chan *ent.Event
	done   chan struct{}
	cancel context.CancelFunc
}

// NewAgentActor creates a new agent actor.
func NewAgentActor(id string, adapter bridge.Adapter, client *ent.Client, logger *slog.Logger) *AgentActor {
	if logger == nil {
		logger = slog.Default()
	}

	return &AgentActor{
		id:      id,
		adapter: adapter,
		client:  client,
		logger:  logger.With("agent", id),
		inbox:   make(chan *ent.Event, 100), // Buffered to prevent blocking
		done:    make(chan struct{}),
	}
}

// Start starts the actor's event processing loop.
func (a *AgentActor) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	a.cancel = cancel

	go a.run(ctx)
}

// Stop stops the actor.
func (a *AgentActor) Stop() {
	if a.cancel != nil {
		a.cancel()
	}
	<-a.done // Wait for goroutine to finish
}

// Send sends an event to the actor's inbox.
func (a *AgentActor) Send(evt *ent.Event) {
	select {
	case a.inbox <- evt:
	default:
		a.logger.Warn("inbox full, dropping event", "event_id", evt.ID)
	}
}

// run is the main event processing loop.
func (a *AgentActor) run(ctx context.Context) {
	defer close(a.done)

	a.logger.Info("actor started")

	for {
		select {
		case evt := <-a.inbox:
			a.handle(ctx, evt)
		case <-ctx.Done():
			a.logger.Info("actor stopping")
			return
		}
	}
}

// handle processes a single event.
func (a *AgentActor) handle(ctx context.Context, evt *ent.Event) {
	a.logger.Debug("handling event",
		"event_id", evt.ID,
		"type", evt.Type,
	)

	var err error

	switch evt.Type {
	case event.TypeHumanMessage:
		err = a.handleHumanMessage(evt)
	case event.TypeInterrupt:
		err = a.handleInterrupt(evt)
	case event.TypeAgentMessage:
		// Check if this is an agent-to-agent message (has source_agent_id)
		if evt.SourceAgentID != "" {
			err = a.handleAgentMessage(evt)
		} else {
			// Outbound agent messages (to chat) are handled elsewhere
			a.logger.Debug("ignoring outbound agent message")
			return
		}
	case event.TypeSystem:
		a.logger.Debug("ignoring system event")
		return
	default:
		a.logger.Warn("unknown event type", "type", evt.Type)
		return
	}

	// Update event status
	status := event.StatusDelivered
	if err != nil {
		status = event.StatusFailed
		a.logger.Error("failed to handle event",
			"event_id", evt.ID,
			"error", err,
		)
	}

	if updateErr := a.updateEventStatus(ctx, evt.ID, status); updateErr != nil {
		a.logger.Error("failed to update event status",
			"event_id", evt.ID,
			"error", updateErr,
		)
	}
}

// handleHumanMessage sends a human message to the agent.
func (a *AgentActor) handleHumanMessage(evt *ent.Event) error {
	// Extract text from payload
	text, ok := evt.Payload["text"].(string)
	if !ok {
		a.logger.Warn("event has no text payload", "event_id", evt.ID)
		return nil
	}

	a.logger.Info("sending message to agent", "text_length", len(text))

	return a.adapter.Send(text)
}

// handleInterrupt sends an interrupt signal to the agent.
func (a *AgentActor) handleInterrupt(evt *ent.Event) error {
	reason := ""
	if r, ok := evt.Payload["reason"].(string); ok {
		reason = r
	}
	a.logger.Info("interrupting agent", "reason", reason)
	return a.adapter.Interrupt()
}

// handleAgentMessage sends an agent-to-agent message to this agent.
// The message is prefixed with the source agent ID for context.
func (a *AgentActor) handleAgentMessage(evt *ent.Event) error {
	// Extract text from payload
	text, ok := evt.Payload["text"].(string)
	if !ok {
		a.logger.Warn("agent message has no text payload", "event_id", evt.ID)
		return nil
	}

	// Format message with source agent prefix
	formattedText := "[from: " + evt.SourceAgentID + "] " + text

	a.logger.Info("sending agent message",
		"from_agent", evt.SourceAgentID,
		"text_length", len(text),
	)

	return a.adapter.Send(formattedText)
}

// updateEventStatus updates the status of an event in the database.
func (a *AgentActor) updateEventStatus(ctx context.Context, eventID string, status event.Status) error {
	return a.client.Event.
		UpdateOneID(eventID).
		SetStatus(status).
		Exec(ctx)
}

// ID returns the agent ID.
func (a *AgentActor) ID() string {
	return a.id
}
