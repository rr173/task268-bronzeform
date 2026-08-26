package store

import (
	"path/filepath"
	"testing"

	"task268-bronzeform/internal/model"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestBatchCreateAndGet(t *testing.T) {
	db := openTestDB(t)

	s := NewBatchStore(db)
	b := &model.Batch{Code: "BX-TEST", Name: "测试批次"}
	if err := s.Create(b); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := s.Get(b.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Code != "BX-TEST" || got.Status != model.BatchOrganizing {
		t.Fatalf("unexpected batch: %+v", got)
	}
}

func TestBatchUpdateStatus(t *testing.T) {
	db := openTestDB(t)

	s := NewBatchStore(db)
	b := &model.Batch{Code: "BX-FLOW", Name: "流转测试"}
	if err := s.Create(b); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateStatus(b.ID, model.BatchPendingCompare); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.BatchPendingCompare {
		t.Fatalf("want pending_compare, got %s", got.Status)
	}
}
