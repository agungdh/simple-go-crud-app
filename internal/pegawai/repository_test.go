package pegawai_test

import (
	"context"
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
	testutil.RequireErrorContains(t, err, "insert pegawai", "foreign key constraint failed")
}
