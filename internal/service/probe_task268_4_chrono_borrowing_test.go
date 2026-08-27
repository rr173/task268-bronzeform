package service

import (
	"testing"

	"task268-bronzeform/internal/model"
)

func TestReversedEraHighSimilarityIsChronoConflict(t *testing.T) {
	svc := newTestService(t)
	b, err := svc.CreateBatch("BX-CHR", "年代借形探针")
	if err != nil {
		t.Fatal(err)
	}
	late, err := svc.ImportGlyph(b.ID, "L", "金", 400, 350, "晚期器", []model.Component{
		{Part: "𠂉", Position: "top", Direction: model.DirNormal},
		{Part: "从", Position: "bottom", Direction: model.DirNormal},
	})
	if err != nil {
		t.Fatal(err)
	}
	early, err := svc.ImportGlyph(b.ID, "E", "金", 900, 850, "早期器", []model.Component{
		{Part: "𠂉", Position: "top", Direction: model.DirNormal},
		{Part: "从", Position: "bottom", Direction: model.DirNormal},
	})
	if err != nil {
		t.Fatal(err)
	}
	rel, err := svc.CreateManualRelation(b.ID, late.ID, early.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rel.Status != model.RelChronoConflict {
		t.Fatalf("reversed era with high similarity must be chrono_conflict, got %s", rel.Status)
	}
}
