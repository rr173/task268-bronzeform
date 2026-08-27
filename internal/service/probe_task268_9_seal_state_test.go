package service

import (
	"errors"
	"testing"

	"task268-bronzeform/internal/model"
)

func TestSealBatchRequiresPublished(t *testing.T) {
	svc := newTestService(t)
	b, err := svc.CreateBatch("BX-SEL", "封存状态探针")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ImportGlyph(b.ID, "A", "金", 900, 850, "毛公鼎", []model.Component{
		{Part: "金", Position: "center", Direction: model.DirNormal},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ImportGlyph(b.ID, "B", "金", 850, 800, "散氏盘", []model.Component{
		{Part: "金", Position: "center", Direction: model.DirNormal},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SubmitBatch(b.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GenerateCandidates(b.ID); err != nil {
		t.Fatal(err)
	}
	got, err := svc.Batches.Get(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.BatchPendingReview {
		t.Fatalf("expected pending_review, got %s", got.Status)
	}
	err = svc.SealBatch(b.ID)
	if !errors.Is(err, model.ErrInvalidState) {
		t.Fatalf("pending_review batch must not seal, got %v", err)
	}
}
