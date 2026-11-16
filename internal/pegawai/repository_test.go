package pegawai_test

import (
	"context"
	"strings"
	"testing"

	"simple-go-crud-app/internal/pegawai"
	"simple-go-crud-app/internal/shared/testutil"
)

func TestCreatePreservesForeignKeyError(t *testing.T) {
	repo := pegawai.NewRepository(testutil.OpenDB(t))
	ctx := context.Background()

	item := &pegawai.Entity{
		BidangID:     999999,
		Nama:         "Budi",
		NIP:          "1",
		TanggalLahir: "1990-08-17",
		Alamat:       "Jl. Merdeka",
		IsMale:       true,
	}
	item.MarkCreated(0)

	err := repo.Create(ctx, item)
	requireErrorContains(t, err, "insert pegawai", "foreign key constraint failed")
}

func requireErrorContains(t *testing.T, err error, parts ...string) {
	t.Helper()

	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	message := strings.ToLower(err.Error())
	for _, part := range parts {
		if !strings.Contains(message, strings.ToLower(part)) {
			t.Errorf("error %q does not contain %q", err, part)
		}
	}
}
