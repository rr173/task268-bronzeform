package version

import (
	"testing"

	"task268-bronzeform/internal/model"
)

func TestCollectDecided(t *testing.T) {
	rels := []model.EvolutionRelation{
		{ID: 1, BatchID: 1, SourceGlyph: 1, TargetGlyph: 2, Kind: model.KindEvolution, Status: model.RelConfirmed},
		{ID: 2, BatchID: 1, SourceGlyph: 2, TargetGlyph: 3, Kind: model.KindEvolution, Status: model.RelCandidate},
		{ID: 3, BatchID: 1, SourceGlyph: 3, TargetGlyph: 4, Kind: model.KindBorrowing, Status: model.RelBorrowed},
		{ID: 4, BatchID: 1, SourceGlyph: 4, TargetGlyph: 5, Kind: model.KindEvolution, Status: model.RelRejected},
	}
	sc, err := CollectDecided(rels, "v1")
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(sc.Relations) != 2 {
		t.Fatalf("expected 2 decided relations (confirmed+borrowed), got %d", len(sc.Relations))
	}
	ids := sc.RelationIDs()
	if ids[0] != 1 || ids[1] != 3 {
		t.Fatalf("expected relation ids [1 3], got %v", ids)
	}
	if sc.ContentHash == "" {
		t.Fatal("expected content hash")
	}
}

func TestCollectDecidedEmpty(t *testing.T) {
	if _, err := CollectDecided(nil, "v1"); err == nil {
		t.Fatal("expected error for no decided relations")
	}
}

func TestContentHashDeterministic(t *testing.T) {
	rels := []model.EvolutionRelation{
		{ID: 1, BatchID: 1, SourceGlyph: 1, TargetGlyph: 2, Kind: model.KindEvolution, Status: model.RelConfirmed},
	}
	sc1, _ := CollectDecided(rels, "v1")
	sc2, _ := CollectDecided(rels, "v1")
	if sc1.ContentHash != sc2.ContentHash {
		t.Fatalf("hash not deterministic: %s vs %s", sc1.ContentHash, sc2.ContentHash)
	}
}
