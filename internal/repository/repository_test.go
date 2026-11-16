package repository_test

import (
	"context"
	"strings"
	"testing"

	"simple-go-crud-app/internal/model"
	"simple-go-crud-app/internal/repository"
	"simple-go-crud-app/internal/testutil"
)

func TestBidangCreatePreservesDatabaseError(t *testing.T) {
	repo := repository.NewBidangRepository(testutil.OpenDB(t))
	ctx := context.Background()

	if err := repo.Create(ctx, newBidang("SDM")); err != nil {
		t.Fatalf("create first bidang: %v", err)
	}

	err := repo.Create(ctx, newBidang("SDM"))
	requireErrorContains(t, err, "insert bidang", "unique constraint failed")
}

func TestBidangUpdatePreservesDatabaseError(t *testing.T) {
	repo := repository.NewBidangRepository(testutil.OpenDB(t))
	ctx := context.Background()

	first := newBidang("SDM")
	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("create first bidang: %v", err)
	}
	second := newBidang("Keuangan")
	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("create second bidang: %v", err)
	}

	second.Bidang = "SDM"
	second.MarkUpdated(0)
	err := repo.Update(ctx, second)
	requireErrorContains(t, err, "update bidang", "unique constraint failed")
}

func TestPegawaiCreatePreservesForeignKeyError(t *testing.T) {
	repo := repository.NewPegawaiRepository(testutil.OpenDB(t))
	ctx := context.Background()

	pegawai := &model.Pegawai{
		BidangID:     999999,
		Nama:         "Budi",
		NIP:          "1",
		TanggalLahir: "1990-08-17",
		Alamat:       "Jl. Merdeka",
		IsMale:       true,
	}
	pegawai.MarkCreated(0)

	err := repo.Create(ctx, pegawai)
	requireErrorContains(t, err, "insert pegawai", "foreign key constraint failed")
}

func newBidang(name string) *model.Bidang {
	bidang := &model.Bidang{Bidang: name}
	bidang.MarkCreated(0)
	return bidang
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
