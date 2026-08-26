package service

import (
	"path/filepath"
	"testing"

	"task268-bronzeform/internal/store"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	path := filepath.Join(t.TempDir(), "svc.db")
	st, err := store.Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return New(
		store.NewBatchStore(st),
		store.NewGlyphStore(st),
		store.NewRelationStore(st),
		store.NewVersionStore(st),
	)
}
