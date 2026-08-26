package service

import (
	"errors"
	"testing"

	"task268-bronzeform/internal/model"
)

func TestRejectRelationBlocksConfirmed(t *testing.T) {
	svc := newTestService(t)
	b, err := svc.CreateBatch("BX-RJC", "否决确认探针")
	if err != nil {
		t.Fatal(err)
	}
	g1, err := svc.ImportGlyph(b.ID, "A", "金", 900, 850, "毛公鼎", []model.Component{
		{Part: "金", Position: "center", Direction: model.DirNormal},
	})
	if err != nil {
		t.Fatal(err)
	}
	g2, err := svc.ImportGlyph(b.ID, "B", "金", 850, 800, "散氏盘", []model.Component{
		{Part: "金", Position: "center", Direction: model.DirNormal},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SubmitBatch(b.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GenerateCandidates(b.ID); err != nil {
		t.Fatal(err)
	}
	rels, err := svc.Rels.ListByBatch(b.ID)
	if err != nil || len(rels) == 0 {
		t.Fatalf("need relations, got %d err=%v", len(rels), err)
	}
	var relID int64
	for _, r := range rels {
		if r.SourceGlyph == g1.ID && r.TargetGlyph == g2.ID {
			relID = r.ID
			break
		}
	}
	if relID == 0 {
		t.Fatal("missing ab relation")
	}
	if err := svc.ConfirmRelation(relID); err != nil {
		t.Fatal(err)
	}
	err = svc.RejectRelation(relID)
	if !errors.Is(err, model.ErrInvalidState) {
		t.Fatalf("confirmed relation must not be rejectable, got %v", err)
	}
}
