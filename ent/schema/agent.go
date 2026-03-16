// Package schema contains the Ent schema definitions for AgentComms.
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Agent holds the schema definition for the Agent entity.
// Agents represent coding assistants that can receive messages.
type Agent struct {
	ent.Schema
}

// Fields of the Agent.
func (Agent) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Unique().
			Immutable().
			Comment("Agent ID (e.g., backend, frontend)"),
		field.String("tenant_id").
			Default("local").
			Comment("Tenant ID for multi-tenancy"),
		field.Enum("type").
			Values("tmux", "process").
			Comment("Agent adapter type"),
		field.JSON("config", map[string]any{}).
			Comment("Adapter configuration (session, pane, etc.)"),
		field.String("channel_id").
			Comment("Bound communication channel"),
		field.Enum("status").
			Values("online", "offline").
			Default("offline").
			Comment("Agent availability status"),
	}
}

// Indexes of the Agent.
func (Agent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("channel_id").Unique(),
		index.Fields("tenant_id"),
	}
}

// Edges of the Agent.
func (Agent) Edges() []ent.Edge {
	return nil
}
