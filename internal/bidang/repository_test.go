package bidang_test

import (
	"context"
	"strings"
	"testing"

	"simple-go-crud-app/internal/bidang"
	"simple-go-crud-app/internal/shared/testutil"
)

func TestCreatePreservesDatabaseError(t *testing.T) {
	repo := bidang.NewRepository(testutil.OpenDB(t))
	ctx := context.Background()

	if err := repo.Create(ctx, newEntity("SDM")); err != nil {
		t.Fatalf("create first bidang: %v", err)
	}

	err := repo.Create(ctx, newEntity("SDM"))
	requireErrorContains(t, err, "insert bidang", "unique constraint failed")
}

func TestUpdatePreservesDatabaseError(t *testing.T) {
	repo := bidang.NewRepository(testutil.OpenDB(t))
	ctx := context.Background()

	first := newEntity("SDM")
	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("create first bidang: %v", err)
	}
	second := newEntity("Keuangan")
	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("create second bidang: %v", err)
	}

	second.Bidang = "SDM"
	second.MarkUpdated(0)
	err := repo.Update(ctx, second)
	requireErrorContains(t, err, "update bidang", "unique constraint failed")
}

func newEntity(name string) *bidang.Entity {
	entity := &bidang.Entity{Bidang: name}
	entity.MarkCreated(0)
	return entity
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
