package service

import (
	"testing"

	"task268-bronzeform/internal/model"
)

func TestBatchSubmitAndGenerate(t *testing.T) {
	svc := newTestService(t)
	b, err := svc.CreateBatch("BX-LOOP", "集成测试批次")
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
	if g1.Status != model.GlyphValid || g2.Status != model.GlyphValid {
		t.Fatalf("glyphs should be valid, got %s %s", g1.Status, g2.Status)
	}
	if _, err := svc.SubmitBatch(b.ID); err != nil {
		t.Fatal(err)
	}
	n, err := svc.GenerateCandidates(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("expected candidates, got %d", n)
	}
	rels, err := svc.Rels.ListByBatch(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range rels {
		if r.Status == model.RelCandidate {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected at least one candidate relation")
	}
}
