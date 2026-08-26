package glyph

import (
	"testing"

	"task268-bronzeform/internal/model"
)

func TestSplitGraph(t *testing.T) {
	comps, err := SplitGraph("⿱𠂉从")
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	if len(comps) != 2 {
		t.Fatalf("expected 2 components, got %d", len(comps))
	}
	if comps[0].Part != "𠂉" || comps[0].Position != "top" {
		t.Errorf("unexpected first component: %+v", comps[0])
	}
	if comps[1].Part != "从" || comps[1].Position != "bottom" {
		t.Errorf("unexpected second component: %+v", comps[1])
	}
}

func TestSplitGraphTruncated(t *testing.T) {
	if _, err := SplitGraph("⿰金"); err == nil {
		t.Fatal("expected error for truncated structure")
	}
}

func TestCompareComponentsAddRemoveDir(t *testing.T) {
	src := []model.Component{
		{Part: "𠂉", Position: "top", Direction: model.DirNormal},
		{Part: "从", Position: "bottom", Direction: model.DirNormal},
	}
	dst := []model.Component{
		{Part: "𠂉", Position: "top", Direction: model.DirNormal},
		{Part: "从", Position: "bottom", Direction: model.DirRotated},
		{Part: "口", Position: "bottom", Direction: model.DirNormal},
	}
	res := CompareComponents(src, dst)
	if len(res.AddedParts) != 1 || res.AddedParts[0] != "口" {
		t.Errorf("expected added [口], got %v", res.AddedParts)
	}
	if len(res.DirChanged) != 1 || res.DirChanged[0] != "从" {
		t.Errorf("expected dir changed [从], got %v", res.DirChanged)
	}
	if res.SharedParts != 2 {
		t.Errorf("expected 2 shared parts, got %d", res.SharedParts)
	}
	if res.Similarity < 0.8 {
		t.Errorf("expected similarity >= 0.8, got %f", res.Similarity)
	}
}

func TestCompareComponentsIdentical(t *testing.T) {
	comps := []model.Component{
		{Part: "𠂉", Position: "top", Direction: model.DirNormal},
		{Part: "从", Position: "bottom", Direction: model.DirNormal},
	}
	res := CompareComponents(comps, comps)
	if res.Similarity != 1.0 {
		t.Errorf("expected similarity 1.0, got %f", res.Similarity)
	}
	if len(res.AddedParts) != 0 || len(res.RemovedParts) != 0 || len(res.DirChanged) != 0 {
		t.Errorf("expected no diffs, got %+v", res)
	}
}

func TestEvalBorrowing(t *testing.T) {
	// 高度相似但结构一致 → 非借形候选。
	identical := CompareResult{Similarity: 1.0}
	if ok, _ := EvalBorrowing(identical); ok {
		t.Fatal("identical shape should not be borrowing candidate")
	}
	// 高度相似且有结构差异 → 借形候选。
	diff := CompareResult{Similarity: 0.9, DirChanged: []string{"从"}}
	if ok, reason := EvalBorrowing(diff); !ok || reason == "" {
		t.Fatalf("expected borrowing candidate, got %v %q", ok, reason)
	}
	// 相似度过低 → 非候选。
	low := CompareResult{Similarity: 0.5}
	if ok, _ := EvalBorrowing(low); ok {
		t.Fatal("low similarity should not be borrowing candidate")
	}
}
