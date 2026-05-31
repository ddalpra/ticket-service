package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Company è l'azienda cliente.
type Company struct{ ent.Schema }

func (Company) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Immutable(),
		field.String("codice").MaxLen(15).Unique(),
		field.String("ragione_sociale").MaxLen(200),
		field.String("piva").MaxLen(16),
		field.String("cf").MaxLen(16),
		field.Bool("active").Default(true),
	}
}

func (Company) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("users", User.Type),
		edge.To("tickets", Ticket.Type),
	}
}
