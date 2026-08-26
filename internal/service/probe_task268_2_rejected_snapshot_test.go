package service

import (
	"testing"

	"task268-bronzeform/internal/model"
)

func TestCollectDecidedOmitsRejectedRelations(t *testing.T) {
	svc := newTestService(t)
	b, err := svc.CreateBatch("BX-REJ", "否决快照探针")
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
	g3, err := svc.ImportGlyph(b.ID, "C", "金", 800, 770, "虢季子白盘", []model.Component{
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
	if err != nil || len(rels) < 2 {
		t.Fatalf("need at least 2 relations, got %d err=%v", len(rels), err)
	}
	var confirmID, rejectID int64
	for _, r := range rels {
		if r.SourceGlyph == g1.ID && r.TargetGlyph == g2.ID {
			confirmID = r.ID
		}
		if r.SourceGlyph == g2.ID && r.TargetGlyph == g3.ID {
			rejectID = r.ID
		}
	}
	if confirmID == 0 || rejectID == 0 {
		t.Fatalf("missing expected relations among %+v", rels)
	}
	if err := svc.ConfirmRelation(confirmID); err != nil {
		t.Fatal(err)
	}
	if err := svc.RejectRelation(rejectID); err != nil {
		t.Fatal(err)
	}
	v, err := svc.CreateVersion(b.ID, "v1")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range v.RelationIDs {
		if id == rejectID {
			t.Fatalf("rejected relation %d must not be in version snapshot", rejectID)
		}
	}
}
