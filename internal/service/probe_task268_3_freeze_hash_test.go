package service

import (
	"errors"
	"testing"

	"task268-bronzeform/internal/model"
)

func TestFreezeVersionRejectsMutatedContent(t *testing.T) {
	svc := newTestService(t)
	b, err := svc.CreateBatch("BX-FRZ", "冻结哈希探针")
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
	if err != nil {
		t.Fatal(err)
	}
	var firstID int64
	for _, r := range rels {
		if r.SourceGlyph == g1.ID && r.TargetGlyph == g2.ID {
			firstID = r.ID
			break
		}
	}
	if firstID == 0 {
		t.Fatal("missing first relation")
	}
	if err := svc.ConfirmRelation(firstID); err != nil {
		t.Fatal(err)
	}
	v, err := svc.CreateVersion(b.ID, "v1")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rels {
		if r.SourceGlyph == g1.ID && r.TargetGlyph == g3.ID {
			if err := svc.ConfirmRelation(r.ID); err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	err = svc.FreezeVersion(v.ID)
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("freeze must reject mutated content hash, got %v", err)
	}
}
