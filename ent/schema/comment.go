package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Comment è un commento su un ticket.
type Comment struct{ ent.Schema }

func (Comment) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Immutable(),
		field.Int("order_comment").NonNegative(),
		field.Text("comment_detail"),
		field.Time("data_creation").Default(time.Now).Immutable(),
	}
}

func (Comment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("ticket", Ticket.Type).
			Ref("comments").
			Unique().
			Required(),
		edge.From("author", User.Type).
			Ref("comments").
			Unique().
			Required(),
		edge.To("attachments", Attachment.Type),
	}
}
