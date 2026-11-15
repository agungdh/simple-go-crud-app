package model

import "github.com/uptrace/bun"

// Bidang is a department. The only business column is bidang; it is stored as
// text and indexed for trigram search.
type Bidang struct {
	bun.BaseModel `bun:"table:bidang,alias:b"`

	Base

	Bidang string `bun:"bidang,notnull"`
}
