package database

import (
	"database/sql"
	"fmt"

	"simple-go-crud-app/internal/shared/apperr"
)

// AffectedOrNotFound inspects the result of an update or delete. No matching
// active row is reported as apperr.ErrNotFound so handlers can answer 404
// instead of leaking a database detail. operation is the caller's context
// label, for example "update bidang 42".
func AffectedOrNotFound(res sql.Result, operation string) error {
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s rows affected: %w", operation, err)
	}
	if affected == 0 {
		return apperr.ErrNotFound
	}
	return nil
}
