package router

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/plexusone/agentcomms/ent"
	"github.com/plexusone/agentcomms/internal/bridge"
)

// Router dispatches events to agent actors.
type Router struct {
	client *ent.Client
	logger *slog.Logger

	mu     sync.RWMutex
	actors map[string]*AgentActor
}

// New creates a new router.
func New(client *ent.Client, logger *slog.Logger) *Router {
	if logger == nil {
		logger = slog.Default()
	}

	return &Router{
		client: client,
		logger: logger,
		actors: make(map[string]*AgentActor),
	}
}

// RegisterAgent registers an agent with its adapter.
func (r *Router) RegisterAgent(ctx context.Context, agentID string, adapter bridge.Adapter) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.actors[agentID]; exists {
		return fmt.Errorf("agent %s already registered", agentID)
	}

	actor := NewAgentActor(agentID, adapter, r.client, r.logger)
	actor.Start(ctx)

	r.actors[agentID] = actor

	r.logger.Info("registered agent", "agent_id", agentID)

	return nil
}

// UnregisterAgent removes an agent from the router.
func (r *Router) UnregisterAgent(agentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	actor, exists := r.actors[agentID]
	if !exists {
		return fmt.Errorf("agent %s not found", agentID)
	}

	actor.Stop()
	delete(r.actors, agentID)

	r.logger.Info("unregistered agent", "agent_id", agentID)

	return nil
}

// Dispatch sends an event to the appropriate agent actor.
func (r *Router) Dispatch(agentID string, evt *ent.Event) error {
	r.mu.RLock()
	actor, exists := r.actors[agentID]
	r.mu.RUnlock()

	if !exists {
		return fmt.Errorf("agent %s not found", agentID)
	}

	actor.Send(evt)
	return nil
}

// DispatchByChannel finds the agent for a channel and dispatches the event.
func (r *Router) DispatchByChannel(channelID string, evt *ent.Event) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Find agent by channel (linear search for now, could use index later)
	for _, actor := range r.actors {
		// For now, we rely on evt.AgentID being set correctly
		if actor.ID() == evt.AgentID {
			actor.Send(evt)
			return nil
		}
	}

	return fmt.Errorf("no agent found for channel %s", channelID)
}

// Agents returns a list of registered agent IDs.
func (r *Router) Agents() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agents := make([]string, 0, len(r.actors))
	for id := range r.actors {
		agents = append(agents, id)
	}
	return agents
}

// HasAgent checks if an agent is registered.
func (r *Router) HasAgent(agentID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.actors[agentID]
	return exists
}

// Stop stops all agent actors.
func (r *Router) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, actor := range r.actors {
		actor.Stop()
		r.logger.Info("stopped agent", "agent_id", id)
	}

	r.actors = make(map[string]*AgentActor)
}
