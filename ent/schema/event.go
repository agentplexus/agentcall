// Package schema contains the Ent schema definitions for AgentComms.
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Event holds the schema definition for the Event entity.
// Events represent all communication in the system: messages, interrupts, system events.
type Event struct {
	ent.Schema
}

// Fields of the Event.
func (Event) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Unique().
			Immutable().
			Comment("Unique event ID (evt_{ulid})"),
		field.String("tenant_id").
			Default("local").
			Comment("Tenant ID for multi-tenancy (local for single-tenant)"),
		field.String("agent_id").
			Comment("Target agent ID"),
		field.String("channel_id").
			Comment("Source channel (e.g., discord:123456)"),
		field.Enum("type").
			Values("human_message", "agent_message", "interrupt", "system").
			Comment("Event type"),
		field.Enum("role").
			Values("human", "agent", "system").
			Comment("Who initiated the event"),
		field.Time("timestamp").
			Default(time.Now).
			Comment("When the event occurred"),
		field.JSON("payload", map[string]any{}).
			Comment("Event payload (text, metadata, etc.)"),
		field.Enum("status").
			Values("new", "delivered", "failed").
			Default("new").
			Comment("Delivery status"),
	}
}

// Indexes of the Event.
func (Event) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("agent_id", "timestamp"),
		index.Fields("channel_id"),
		index.Fields("tenant_id"),
		index.Fields("status"),
	}
}

// Edges of the Event.
func (Event) Edges() []ent.Edge {
	return nil
}
