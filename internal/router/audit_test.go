package router_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/uptrace/bun"

	"simple-go-crud-app/internal/dto"
)

// auditSnapshot holds the internal audit columns, which are never exposed by
// the API. Tests read them directly from the database.
type auditSnapshot struct {
	CreatedAt *int64 `bun:"created_at"`
	CreatedBy *int64 `bun:"created_by"`
	UpdatedAt *int64 `bun:"updated_at"`
	UpdatedBy *int64 `bun:"updated_by"`
	DeletedAt *int64 `bun:"deleted_at"`
	DeletedBy *int64 `bun:"deleted_by"`
}

func loadAudit(t *testing.T, db *bun.DB, table string, id int64) auditSnapshot {
	t.Helper()

	var row auditSnapshot
	if err := db.NewSelect().
		Table(table).
		Column("created_at", "created_by", "updated_at", "updated_by", "deleted_at", "deleted_by").
		Where("id = ?", id).
		Scan(context.Background(), &row); err != nil {
		t.Fatalf("load audit %s %d: %v", table, id, err)
	}
	return row
}

func requireActorColumn(t *testing.T, label string, got *int64, want int64) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s is NULL, want %d", label, want)
	}
	if *got != want {
		t.Errorf("%s = %d, want %d", label, *got, want)
	}
}

func requireEmptyAudit(t *testing.T, label string, got *int64) {
	t.Helper()
	if got != nil {
		t.Errorf("%s = %d, want NULL", label, *got)
	}
}

func TestAuditColumnsFollowAuthenticatedUser(t *testing.T) {
	env := newTestEnv(t)

	// Create bidang as admin.
	rec := env.do(http.MethodPost, "/bidang", `{"bidang":"Keuangan"}`)
	requireStatus(t, rec, http.StatusCreated)

	rec = env.do(http.MethodGet, "/bidang", "")
	requireStatus(t, rec, http.StatusOK)
	bidangList := decodeJSON[dto.BidangListResponse](t, rec)
	if len(bidangList.Data) != 1 {
		t.Fatalf("bidang count = %d, want 1", len(bidangList.Data))
	}
	bidangID := bidangList.Data[0].ID

	audit := loadAudit(t, env.db, "bidang", bidangID)
	requireActorColumn(t, "bidang.created_by", audit.CreatedBy, env.adminID)
	if audit.CreatedAt == nil || *audit.CreatedAt <= 0 {
		t.Errorf("bidang.created_at = %v, want positive epoch milliseconds", audit.CreatedAt)
	}
	requireEmptyAudit(t, "bidang.updated_by", audit.UpdatedBy)
	requireEmptyAudit(t, "bidang.deleted_by", audit.DeletedBy)

	// Update bidang as admin.
	rec = env.do(http.MethodPut, fmt.Sprintf("/bidang/%d", bidangID), `{"bidang":"Keuangan Pusat"}`)
	requireStatus(t, rec, http.StatusOK)

	audit = loadAudit(t, env.db, "bidang", bidangID)
	requireActorColumn(t, "bidang.created_by after update", audit.CreatedBy, env.adminID)
	requireActorColumn(t, "bidang.updated_by", audit.UpdatedBy, env.adminID)

	// Delete bidang as admin.
	rec = env.do(http.MethodDelete, fmt.Sprintf("/bidang/%d", bidangID), "")
	requireStatus(t, rec, http.StatusNoContent)

	audit = loadAudit(t, env.db, "bidang", bidangID)
	requireActorColumn(t, "bidang.created_by after delete", audit.CreatedBy, env.adminID)
	requireActorColumn(t, "bidang.updated_by after delete", audit.UpdatedBy, env.adminID)
	requireActorColumn(t, "bidang.deleted_by", audit.DeletedBy, env.adminID)
	if audit.DeletedAt == nil || *audit.DeletedAt <= 0 {
		t.Errorf("bidang.deleted_at = %v, want positive epoch milliseconds", audit.DeletedAt)
	}

	// A second bidang is the parent for the pegawai audit test.
	rec = env.do(http.MethodPost, "/bidang", `{"bidang":"SDM"}`)
	requireStatus(t, rec, http.StatusCreated)

	rec = env.do(http.MethodGet, "/bidang", "")
	requireStatus(t, rec, http.StatusOK)
	bidangList = decodeJSON[dto.BidangListResponse](t, rec)
	if len(bidangList.Data) != 1 {
		t.Fatalf("active bidang count = %d, want 1", len(bidangList.Data))
	}
	bidangID = bidangList.Data[0].ID

	pegawaiBody := fmt.Sprintf(
		`{"bidang_id":%d,"nama":"Budi","nip":"123","tanggal_lahir":"01-01-1990","alamat":"Jl. A","jenis_kelamin":"pria"}`,
		bidangID,
	)
	rec = env.do(http.MethodPost, "/pegawai", pegawaiBody)
	requireStatus(t, rec, http.StatusCreated)

	rec = env.do(http.MethodGet, "/pegawai", "")
	requireStatus(t, rec, http.StatusOK)
	pegawaiList := decodeJSON[dto.PegawaiListResponse](t, rec)
	if len(pegawaiList.Data) != 1 {
		t.Fatalf("pegawai count = %d, want 1", len(pegawaiList.Data))
	}
	pegawaiID := pegawaiList.Data[0].ID

	audit = loadAudit(t, env.db, "pegawai", pegawaiID)
	requireActorColumn(t, "pegawai.created_by", audit.CreatedBy, env.adminID)
	requireEmptyAudit(t, "pegawai.updated_by", audit.UpdatedBy)
	requireEmptyAudit(t, "pegawai.deleted_by", audit.DeletedBy)

	rec = env.do(http.MethodPut, fmt.Sprintf("/pegawai/%d", pegawaiID), pegawaiBody)
	requireStatus(t, rec, http.StatusOK)

	audit = loadAudit(t, env.db, "pegawai", pegawaiID)
	requireActorColumn(t, "pegawai.updated_by", audit.UpdatedBy, env.adminID)

	rec = env.do(http.MethodDelete, fmt.Sprintf("/pegawai/%d", pegawaiID), "")
	requireStatus(t, rec, http.StatusNoContent)

	audit = loadAudit(t, env.db, "pegawai", pegawaiID)
	requireActorColumn(t, "pegawai.created_by after delete", audit.CreatedBy, env.adminID)
	requireActorColumn(t, "pegawai.updated_by after delete", audit.UpdatedBy, env.adminID)
	requireActorColumn(t, "pegawai.deleted_by", audit.DeletedBy, env.adminID)
}
