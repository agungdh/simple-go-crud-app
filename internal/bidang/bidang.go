// Package bidang implements the bidang (department) domain end to end: model,
// persistence, business rules, DTOs, and HTTP handlers.
package bidang

import (
	"github.com/uptrace/bun"

	"simple-go-crud-app/internal/shared/entity"
)

// Entity is a department. The only business column is bidang; it is stored as
// text and indexed for trigram search.
type Entity struct {
	bun.BaseModel `bun:"table:bidang,alias:b"`

	entity.Base

	Bidang string `bun:"bidang,notnull"`
}
