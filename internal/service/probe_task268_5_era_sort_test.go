package service

import (
	"testing"

	"task268-bronzeform/internal/model"
	"task268-bronzeform/internal/relation"
)

func TestPairwiseRespectsEraOrdering(t *testing.T) {
	svc := newTestService(t)
	b, err := svc.CreateBatch("BX-ERA", "年代排序探针")
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
	glyphs := []model.Glyph{*g1, *g2, *g3}
	pairs, err := relation.BuildPairwise(glyphs)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range pairs {
		if p.Source.EraBegin < p.Target.EraBegin {
			t.Fatalf("source era must not be later than target: %d(%d)->%d(%d)",
				p.Source.ID, p.Source.EraBegin, p.Target.ID, p.Target.EraBegin)
		}
	}
}
