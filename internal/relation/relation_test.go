package relation

import (
	"testing"

	"task268-bronzeform/internal/model"
)

func TestBuildCandidateEvolution(t *testing.T) {
	src := model.Glyph{ID: 1, EraBegin: 900, EraEnd: 850, Components: []model.Component{
		{Part: "𠂉", Position: "top", Direction: model.DirNormal},
		{Part: "从", Position: "bottom", Direction: model.DirNormal},
	}}
	dst := model.Glyph{ID: 2, EraBegin: 800, EraEnd: 770, Components: []model.Component{
		{Part: "𠂉", Position: "top", Direction: model.DirNormal},
		{Part: "从", Position: "bottom", Direction: model.DirNormal},
	}}
	cand, err := BuildCandidate(CandidateInput{Source: src, Target: dst})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if cand == nil {
		t.Fatal("expected candidate")
	}
	if cand.Status != model.RelCandidate {
		t.Fatalf("expected candidate status, got %s", cand.Status)
	}
	if cand.ChronoScore != 2 {
		t.Fatalf("expected chrono score 2, got %d", cand.ChronoScore)
	}
}

func TestBuildCandidateBorrowing(t *testing.T) {
	src := model.Glyph{ID: 1, EraBegin: 900, EraEnd: 850, Components: []model.Component{
		{Part: "𠂉", Position: "top", Direction: model.DirNormal},
		{Part: "从", Position: "bottom", Direction: model.DirNormal},
	}}
	dst := model.Glyph{ID: 2, EraBegin: 800, EraEnd: 770, Components: []model.Component{
		{Part: "𠂉", Position: "top", Direction: model.DirNormal},
		{Part: "从", Position: "bottom", Direction: model.DirRotated},
	}}
	cand, err := BuildCandidate(CandidateInput{Source: src, Target: dst})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if cand == nil {
		t.Fatal("expected candidate")
	}
	if cand.Kind != model.KindBorrowing || cand.Status != model.RelBorrowed {
		t.Fatalf("expected borrowing/borrowed, got kind=%s status=%s", cand.Kind, cand.Status)
	}
}

func TestBuildCandidateChronoConflict(t *testing.T) {
	src := model.Glyph{ID: 1, EraBegin: 400, EraEnd: 350, Components: []model.Component{
		{Part: "𠂉", Position: "top", Direction: model.DirNormal},
		{Part: "从", Position: "bottom", Direction: model.DirNormal},
	}}
	dst := model.Glyph{ID: 2, EraBegin: 900, EraEnd: 850, Components: []model.Component{
		{Part: "𠂉", Position: "top", Direction: model.DirNormal},
		{Part: "从", Position: "bottom", Direction: model.DirNormal},
	}}
	cand, err := BuildCandidate(CandidateInput{Source: src, Target: dst})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if cand == nil {
		t.Fatal("expected candidate")
	}
	if cand.Status != model.RelChronoConflict {
		t.Fatalf("expected chrono_conflict, got %s", cand.Status)
	}
}

func TestBuildCandidateSelfReference(t *testing.T) {
	g := model.Glyph{ID: 1, EraBegin: 900, EraEnd: 850}
	if _, err := BuildCandidate(CandidateInput{Source: g, Target: g}); err != model.ErrSelfReference {
		t.Fatalf("expected self-reference error, got %v", err)
	}
}

func TestBuildCandidateTooDissimilar(t *testing.T) {
	src := model.Glyph{ID: 1, EraBegin: 900, EraEnd: 850, Components: []model.Component{
		{Part: "金", Position: "left", Direction: model.DirNormal},
	}}
	dst := model.Glyph{ID: 2, EraBegin: 800, EraEnd: 770, Components: []model.Component{
		{Part: "水", Position: "right", Direction: model.DirNormal},
	}}
	cand, err := BuildCandidate(CandidateInput{Source: src, Target: dst})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if cand != nil {
		t.Fatalf("expected nil for dissimilar pair, got %+v", cand)
	}
}

func TestBuildPairwiseSortsByEra(t *testing.T) {
	glyphs := []model.Glyph{
		{ID: 3, EraBegin: 800, EraEnd: 770, Status: model.GlyphValid, Components: []model.Component{{Part: "从"}}},
		{ID: 1, EraBegin: 900, EraEnd: 850, Status: model.GlyphValid, Components: []model.Component{{Part: "从"}}},
		{ID: 2, EraBegin: 850, EraEnd: 800, Status: model.GlyphValid, Components: []model.Component{{Part: "从"}}},
	}
	pairs, err := BuildPairwise(glyphs)
	if err != nil {
		t.Fatalf("pairwise: %v", err)
	}
	if len(pairs) != 3 {
		t.Fatalf("expected 3 pairs, got %d", len(pairs))
	}
	// 每个 pair 的源年代不晚于目标年代（源 begin >= 目标 begin）。
	for _, p := range pairs {
		if p.Source.EraBegin < p.Target.EraBegin {
			t.Errorf("pair not era-sorted: %d(begin %d) -> %d(begin %d)",
				p.Source.ID, p.Source.EraBegin, p.Target.ID, p.Target.EraBegin)
		}
	}
}

func TestBuildPairwiseExcludesNonValidGlyphs(t *testing.T) {
	comps := []model.Component{{Part: "从", Position: "center", Direction: model.DirNormal}}
	glyphs := []model.Glyph{
		{ID: 1, EraBegin: 900, EraEnd: 850, Status: model.GlyphValid, Components: comps},
		{ID: 2, EraBegin: 850, EraEnd: 800, Status: model.GlyphValid, Components: comps},
		{ID: 3, EraBegin: 800, EraEnd: 770, Status: model.GlyphExcluded, Components: comps},  // 被研究者排除
		{ID: 4, EraBegin: 800, EraEnd: 770, Status: model.GlyphDefective, Components: comps}, // 拓片残缺
		{ID: 5, EraBegin: 800, EraEnd: 770, Status: model.GlyphPendingSplit, Components: comps},
	}
	pairs, err := BuildPairwise(glyphs)
	if err != nil {
		t.Fatalf("pairwise: %v", err)
	}
	// 仅两个 valid 字形 → 1 对；excluded/defective/pending_split 不得作为源或目标。
	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair (only valid glyphs), got %d", len(pairs))
	}
	for _, p := range pairs {
		if p.Source.ID == 3 || p.Target.ID == 3 {
			t.Errorf("excluded glyph 3 should not appear in pair %d->%d", p.Source.ID, p.Target.ID)
		}
		if p.Source.ID == 4 || p.Target.ID == 4 {
			t.Errorf("defective glyph 4 should not appear in pair %d->%d", p.Source.ID, p.Target.ID)
		}
		if p.Source.ID == 5 || p.Target.ID == 5 {
			t.Errorf("pending_split glyph 5 should not appear in pair %d->%d", p.Source.ID, p.Target.ID)
		}
	}
}
