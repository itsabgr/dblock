package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type DLock struct {
	ent.Schema
}

func (DLock) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable().MaxLen(64).NotEmpty(),
		field.UUID("holder", uuid.UUID{}).Unique().Immutable().Default(uuid.New),
		field.Int64("deadline").Min(1577840461).Max(4102444800).Positive(),
	}
}
