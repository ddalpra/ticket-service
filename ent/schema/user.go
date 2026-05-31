package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// User è la specializzazione locale dell'utente Keycloak.
type User struct{ ent.Schema }

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Immutable(),
		// sub del JWT emesso da Keycloak
		field.String("keycloak_id").Unique().Immutable(),
		field.String("username").MaxLen(150),
		field.String("email").MaxLen(255),
		field.String("first_name").MaxLen(100).Optional(),
		field.String("last_name").MaxLen(100).Optional(),
		field.Enum("role").
			Values("customer", "support_l1", "support_l2", "supervisor"),
		field.Bool("active").Default(true),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		// Un customer appartiene a un'azienda
		edge.From("company", Company.Type).
			Ref("users").
			Unique(),
		// Un operatore appartiene a un centro servizi
		edge.From("service_center", ServiceCenter.Type).
			Ref("users").
			Unique(),
		// Ticket assegnati a questo utente
		edge.To("assigned_tickets", Ticket.Type).
			StorageKey(edge.Column("assigned_to_id")),
		// Ticket creati da questo utente
		edge.To("created_tickets", Ticket.Type).
			StorageKey(edge.Column("created_by_id")),
		// Ticket aggiornati da questo utente
		edge.To("updated_tickets", Ticket.Type).
			StorageKey(edge.Column("updated_by_id")),
		// Aggiungi questa riga per completare la relazione:
        	edge.To("attachments", Attachment.Type),
        	// Commenti scritti da questo utente (Risolve l'errore Comment che mancava)
		edge.To("comments", Comment.Type),
	}
}
