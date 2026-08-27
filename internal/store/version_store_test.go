package store

import (
	"testing"

	"task268-bronzeform/internal/model"
)

// seedVersionWithRels 建一个批次、4 个字形与对应 4 条关系，返回版本存储、版本 ID 与关系 ID 列表。
func seedVersionWithRels(t *testing.T) (*VersionStore, int64, []int64) {
	t.Helper()
	db := openTestDB(t)

	bs := NewBatchStore(db)
	b := &model.Batch{Code: "BX-VR", Name: "版本关系快照测试"}
	if err := bs.Create(b); err != nil {
		t.Fatalf("create batch: %v", err)
	}
	gs := NewGlyphStore(db)
	rs := NewRelationStore(db)
	rels := make([]int64, 0, 4)
	for i := 0; i < 4; i++ {
		src := &model.Glyph{
			BatchID: b.ID, Code: "S" + string(rune('A'+i)), Graph: "■", EraBegin: 900 - i, EraEnd: 860 - i,
			Source: "src", Fingerprint: "fp-src-" + string(rune('A'+i)),
		}
		tgt := &model.Glyph{
			BatchID: b.ID, Code: "T" + string(rune('A'+i)), Graph: "□", EraBegin: 850 - i, EraEnd: 810 - i,
			Source: "src", Fingerprint: "fp-tgt-" + string(rune('A'+i)),
		}
		if err := gs.Create(src); err != nil {
			t.Fatalf("create source glyph %d: %v", i, err)
		}
		if err := gs.Create(tgt); err != nil {
			t.Fatalf("create target glyph %d: %v", i, err)
		}
		r := &model.EvolutionRelation{
			BatchID:      b.ID,
			SourceGlyph:  src.ID,
			TargetGlyph:  tgt.ID,
			Kind:         model.KindEvolution,
			Status:       model.RelConfirmed,
			ChronoScore:  1,
		}
		if err := rs.Create(r); err != nil {
			t.Fatalf("create relation %d: %v", i, err)
		}
		rels = append(rels, r.ID)
	}
	vs := NewVersionStore(db)
	v := &model.EvolutionVersion{BatchID: b.ID, Name: "草稿 v1", ContentHash: "h"}
	if err := vs.Create(v); err != nil {
		t.Fatalf("create version: %v", err)
	}
	return vs, v.ID, rels
}

// TestSetRelationsOverwritesOldBindings 覆盖式写入：对同一版本再次写入时，
// 先前快照里残留的关系 ID 必须被清除，只保留新集合。
func TestSetRelationsOverwritesOldBindings(t *testing.T) {
	vs, verID, rels := seedVersionWithRels(t)

	// 第一轮：快照写入关系 0、1、2。
	if err := vs.SetRelations(verID, []int64{rels[0], rels[1], rels[2]}); err != nil {
		t.Fatalf("set relations first: %v", err)
	}
	got, err := vs.RelationIDs(verID)
	if err != nil {
		t.Fatalf("relation ids first: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 relation ids, got %v", got)
	}

	// 第二轮：研究者否决 rels[1]、确认 rels[3]，重新写入只有 0、2、3 的快照。
	if err := vs.SetRelations(verID, []int64{rels[0], rels[2], rels[3]}); err != nil {
		t.Fatalf("set relations second: %v", err)
	}
	got, err = vs.RelationIDs(verID)
	if err != nil {
		t.Fatalf("relation ids second: %v", err)
	}
	// 已否决的 rels[1] 不得残留。
	for _, id := range got {
		if id == rels[1] {
			t.Fatalf("rejected relation %d must not leak into updated snapshot, got %v", id, got)
		}
	}
	if len(got) != 3 || got[0] != rels[0] || got[1] != rels[2] || got[2] != rels[3] {
		t.Fatalf("expected [%d %d %d], got %v", rels[0], rels[2], rels[3], got)
	}
}

// TestSetRelationsClearsAllBindings 写入空集合应清空该版本的全部关系绑定。
func TestSetRelationsClearsAllBindings(t *testing.T) {
	vs, verID, rels := seedVersionWithRels(t)

	if err := vs.SetRelations(verID, []int64{rels[0], rels[1]}); err != nil {
		t.Fatalf("set relations: %v", err)
	}
	// 写入空集合，旧行必须被删除。
	if err := vs.SetRelations(verID, nil); err != nil {
		t.Fatalf("set relations empty: %v", err)
	}
	got, err := vs.RelationIDs(verID)
	if err != nil {
		t.Fatalf("relation ids: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no relation ids after clearing, got %v", got)
	}
}
