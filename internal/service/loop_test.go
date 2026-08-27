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

// TestGenerateCandidatesExcludesExcludedGlyph 回归测试：
// 复核员把拓片质量差的字形标成 excluded 后生成候选，
// 两两比较链路不得把该字形作为演变对的源或目标。
func TestGenerateCandidatesExcludesExcludedGlyph(t *testing.T) {
	svc := newTestService(t)
	b, err := svc.CreateBatch("BX-EXC", "排除字形回归")
	if err != nil {
		t.Fatal(err)
	}
	comps := []model.Component{{Part: "从", Position: "center", Direction: model.DirNormal}}
	g1, err := svc.ImportGlyph(b.ID, "A", "从", 900, 850, "毛公鼎", comps)
	if err != nil {
		t.Fatal(err)
	}
	g2, err := svc.ImportGlyph(b.ID, "B", "从", 850, 800, "散氏盘", comps)
	if err != nil {
		t.Fatal(err)
	}
	g3, err := svc.ImportGlyph(b.ID, "C", "从", 800, 770, "季子白盘", comps)
	if err != nil {
		t.Fatal(err)
	}
	// 复核员把 g3（拓片质量差）排除。
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
	if len(rels) == 0 {
		t.Fatal("expected at least one candidate (g1↔g2)")
	}
	for _, r := range rels {
		if r.SourceGlyph == g3.ID || r.TargetGlyph == g3.ID {
			t.Errorf("excluded glyph %d still appears in relation %d (%d->%d, status=%s)",
				g3.ID, r.ID, r.SourceGlyph, r.TargetGlyph, r.Status)
		}
	}
	// g1↔g2 仍应作为候选保留。
	var pair12 bool
	for _, r := range rels {
		if (r.SourceGlyph == g1.ID && r.TargetGlyph == g2.ID) ||
			(r.SourceGlyph == g2.ID && r.TargetGlyph == g1.ID) {
			pair12 = true
		}
	}
	if !pair12 {
		t.Error("expected g1↔g2 candidate to be generated after excluding g3")
	}
}
