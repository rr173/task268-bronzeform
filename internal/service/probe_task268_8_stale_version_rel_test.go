package service

import (
	"testing"

	"task268-bronzeform/internal/model"
)

func TestSetRelationsReplacesStaleIDs(t *testing.T) {
	svc := newTestService(t)
	b, err := svc.CreateBatch("BX-VR", "版本关系探针")
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
		t.Fatalf("need relations, got %d err=%v", len(rels), err)
	}
	var relAB, relBC int64
	for _, r := range rels {
		if r.SourceGlyph == g1.ID && r.TargetGlyph == g2.ID {
			relAB = r.ID
		}
		if r.SourceGlyph == g2.ID && r.TargetGlyph == g3.ID {
			relBC = r.ID
		}
	}
	if relAB == 0 || relBC == 0 {
		t.Fatalf("missing ab/bc relations: %+v", rels)
	}
	if err := svc.ConfirmRelation(relAB); err != nil {
		t.Fatal(err)
	}
	v1, err := svc.CreateVersion(b.ID, "v1")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ConfirmRelation(relBC); err != nil {
		t.Fatal(err)
	}
	if err := svc.Versions.SetRelations(v1.ID, []int64{relBC}); err != nil {
		t.Fatal(err)
	}
	ids, err := svc.Versions.RelationIDs(v1.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		if id == relAB {
			t.Fatalf("stale relation id %d must not remain after SetRelations replace", relAB)
		}
	}
	foundBC := false
	for _, id := range ids {
		if id == relBC {
			foundBC = true
		}
	}
	if !foundBC {
		t.Fatalf("SetRelations must bind new relation %d, got %v", relBC, ids)
	}
}
