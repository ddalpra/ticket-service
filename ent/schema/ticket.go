package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Ticket rappresenta una richiesta di supporto.
type Ticket struct{ ent.Schema }

func (Ticket) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Immutable(),
		field.String("title").MaxLen(150),
		field.Text("question"),

		field.Enum("state").
			Values("OPEN", "CLOSED", "TODO", "WAIT").
			Default("OPEN"),

		field.Enum("priority").
			Values("LOW", "MEDIUM", "HIGH").
			Default("LOW"),

		field.Enum("state_job").
			Values(
				"APERTA",
				"IN_ATTESA_CLIENTE",
				"PRESA_IN_CARICO",
				"IN_ATTESA_CENTRO_SERVIZI",
				"CHIUSA",
			).
			Default("APERTA"),

		field.Time("data_assigned").Optional().Nillable(),
		field.Time("data_creation").Default(time.Now).Immutable(),
		field.Time("data_updating").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Ticket) Edges() []ent.Edge {
	return []ent.Edge{
		// Azienda proprietaria del ticket
		edge.From("company", Company.Type).
			Ref("tickets").
			Unique().
			Required(),

		// Centro servizi a cui è destinato il ticket
		edge.From("service_center", ServiceCenter.Type).
			Ref("tickets").
			Unique().
			Required(),

		// Operatore assegnato (opzionale: non assegnato alla creazione)
		edge.From("assigned_to", User.Type).
			Ref("assigned_tickets").
			Unique(),

		// Chi ha creato il ticket
		edge.From("created_by", User.Type).
			Ref("created_tickets").
			Unique().
			Required(),

		// Chi ha effettuato l'ultima modifica
		edge.From("updated_by", User.Type).
			Ref("updated_tickets").
			Unique(),

		edge.To("comments", Comment.Type),
		edge.To("attachments", Attachment.Type),
	}
}
