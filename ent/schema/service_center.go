package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// ServiceCenter è il centro servizi che gestisce i ticket.
type ServiceCenter struct{ ent.Schema }

func (ServiceCenter) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Immutable(),
		field.String("codice").MaxLen(20).Unique(),
		field.String("nome").MaxLen(200),
		field.Bool("active").Default(true),
	}
}

func (ServiceCenter) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("users", User.Type),
		edge.To("tickets", Ticket.Type),
	}
}
