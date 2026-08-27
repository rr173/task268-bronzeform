package service

import (
	"testing"

	"task268-bronzeform/internal/model"
)

func TestExcludedGlyphsOmittedFromPairwise(t *testing.T) {
	svc := newTestService(t)
	b, err := svc.CreateBatch("BX-EXCL", "排除字形探针")
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
	if err := svc.ExcludeGlyph(g3.ID); err != nil {
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
	for _, r := range rels {
		if r.SourceGlyph == g3.ID || r.TargetGlyph == g3.ID {
			t.Fatalf("excluded glyph %d must not appear in relations, got rel %d", g3.ID, r.ID)
		}
	}
	_ = g1
	_ = g2
}
