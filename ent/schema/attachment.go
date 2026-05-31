package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Attachment è un allegato, collegabile sia a un Ticket che a un Comment.
type Attachment struct{ ent.Schema }

func (Attachment) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Immutable(),
		field.String("name").MaxLen(500),
		field.String("path").MaxLen(2000),
		field.String("mime_type").MaxLen(100).Optional(),
		field.Int64("size_bytes").Optional(),
		field.Time("uploaded_at").Default(time.Now).Immutable(),
	}
}

func (Attachment) Edges() []ent.Edge {
	return []ent.Edge{
		// Allegato a un ticket (opzionale: potrebbe essere solo su commento)
		edge.From("ticket", Ticket.Type).
			Ref("attachments").
			Unique(),
		// Allegato a un commento (opzionale)
		edge.From("comment", Comment.Type).
			Ref("attachments").
			Unique(),
		// Chi ha caricato l'allegato
		edge.From("uploaded_by", User.Type).
			Ref("attachments").
			Unique(),
	}
}
