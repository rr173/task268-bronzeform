package service

import (
	"errors"
	"testing"

	"task268-bronzeform/internal/model"
)

// newDecidedService 建一个带 1 条 candidate 演变关系的服务（G1→G2，西周早→晚）。
func newDecidedService(t *testing.T) (*Service, int64, int64) {
	t.Helper()
	svc := newTestService(t)
	b, err := svc.CreateBatch("BX-DEC", "decided batch")
	if err != nil {
		t.Fatal(err)
	}
	g1, err := svc.ImportGlyph(b.ID, "A", "旅", 900, 850, "毛公鼎", []model.Component{
		{Part: "𠂉", Position: "top", Direction: model.DirNormal},
		{Part: "从", Position: "bottom", Direction: model.DirNormal},
	})
	if err != nil {
		t.Fatal(err)
	}
	g2, err := svc.ImportGlyph(b.ID, "B", "旅", 850, 800, "散氏盘", []model.Component{
		{Part: "𠂉", Position: "top", Direction: model.DirNormal},
		{Part: "从", Position: "bottom", Direction: model.DirNormal},
	})
	if err != nil {
		t.Fatal(err)
	}
	if g1.Status != model.GlyphValid || g2.Status != model.GlyphValid {
		t.Fatalf("glyphs should be valid, got %s %s", g1.Status, g2.Status)
	}
	if _, err := svc.SubmitBatch(b.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GenerateCandidates(b.ID); err != nil {
		t.Fatal(err)
	}
	rels, err := svc.Rels.ListByBatch(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	var candID int64
	for _, r := range rels {
		if r.Status == model.RelCandidate {
			candID = r.ID
			break
		}
	}
	if candID == 0 {
		t.Fatal("expected a candidate relation")
	}
	return svc, candID, b.ID
}

func TestRejectCandidateSucceeds(t *testing.T) {
	svc, candID, _ := newDecidedService(t)
	if err := svc.RejectRelation(candID); err != nil {
		t.Fatalf("reject candidate: %v", err)
	}
	r, err := svc.Rels.Get(candID)
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != model.RelRejected {
		t.Fatalf("expected rejected, got %s", r.Status)
	}
}

// 关键回归：已确认（并可能被版本快照/冻结）的关系不得再被否决，
// 否则冻结证据与关系实时状态不一致。
func TestRejectConfirmedRelationForbidden(t *testing.T) {
	svc, candID, _ := newDecidedService(t)
	if err := svc.ConfirmRelation(candID); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	err := svc.RejectRelation(candID)
	if err == nil {
		t.Fatal("expected error rejecting confirmed relation, got nil")
	}
	if !errors.Is(err, model.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState, got %v", err)
	}
	r, err := svc.Rels.Get(candID)
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != model.RelConfirmed {
		t.Fatalf("confirmed relation must stay confirmed after rejected attempt, got %s", r.Status)
	}
}

// 已确认关系即便纳入版本冻结后仍不可否决（冻结内容哈希不可变）。
func TestRejectConfirmedRelationForbiddenAfterFreeze(t *testing.T) {
	svc, candID, batchID := newDecidedService(t)
	if err := svc.ConfirmRelation(candID); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	v, err := svc.CreateVersion(batchID, "v1")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ShareVersion(v.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.FreezeVersion(v.ID); err != nil {
		t.Fatal(err)
	}
	err = svc.RejectRelation(candID)
	if err == nil {
		t.Fatal("expected error rejecting frozen-version confirmed relation, got nil")
	}
	if !errors.Is(err, model.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState, got %v", err)
	}
	r, err := svc.Rels.Get(candID)
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != model.RelConfirmed {
		t.Fatalf("frozen-version relation must stay confirmed, got %s", r.Status)
	}
}

// chrono_conflict 关系非 candidate，不得被否决。
func TestRejectChronoConflictForbidden(t *testing.T) {
	svc, candID, batchID := newDecidedService(t)
	// 同批另起一对构造 chrono_conflict：源（战国，晚）晚于目标（西周，早）。
	g3, err := svc.ImportGlyph(batchID, "C", "旅", 400, 350, "中山王鼎", []model.Component{
		{Part: "𠂉", Position: "top", Direction: model.DirNormal},
		{Part: "从", Position: "bottom", Direction: model.DirNormal},
	})
	if err != nil {
		t.Fatal(err)
	}
	// 候选关系的源字形即西周早字形，作为年代逆序的目标。
	cand, err := svc.Rels.Get(candID)
	if err != nil {
		t.Fatal(err)
	}
	revRel, err := svc.CreateManualRelation(batchID, g3.ID, cand.SourceGlyph)
	if err != nil {
		t.Fatal(err)
	}
	if revRel.Status != model.RelChronoConflict {
		t.Fatalf("expected chrono_conflict, got %s", revRel.Status)
	}
	err = svc.RejectRelation(revRel.ID)
	if err == nil {
		t.Fatal("expected error rejecting chrono_conflict relation, got nil")
	}
	if !errors.Is(err, model.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState, got %v", err)
	}
}
