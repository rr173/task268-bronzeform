// Package relation 实现演变候选的生成与裁决：
// 综合构形差异与年代可行性生成 candidate / chrono_conflict / borrowed 关系，
// 并支持研究者的确认、否决与借形裁决。
package relation

import (
	"fmt"

	"task268-bronzeform/internal/chrono"
	"task268-bronzeform/internal/glyph"
	"task268-bronzeform/internal/model"
)

// CandidateInput 候选生成输入：源字形、目标字形及其构件。
type CandidateInput struct {
	Source model.Glyph
	Target model.Glyph
}

// Candidate 生成的候选关系。
type Candidate struct {
	Kind        model.RelationKind
	Status      model.RelationStatus
	ChronoScore int
	AddedParts  []string
	Removed     []string
	DirChanged  []string
	Evidence    string
}

// BuildCandidate 由构形比较与年代可行性生成候选关系。
// 判定规则（按优先级）：
//  1. 构形相似度过低（<0.5）：不构成演变候选 → 返回 nil（跳过）。
//  2. 构形高度相似但存在结构差异 → 借形候选（borrowing），状态 borrowed 需研究者确认。
//  3. 年代逆序（score<0）→ chrono_conflict，不可作演变。
//  4. 年代可行（score>0）→ evolution candidate，待裁决。
func BuildCandidate(in CandidateInput) (*Candidate, error) {
	if in.Source.ID == in.Target.ID {
		return nil, model.ErrSelfReference
	}
	diff := glyph.CompareComponents(in.Source.Components, in.Target.Components)
	if diff.Similarity < 0.5 {
		return nil, nil // 构形差异过大，不成候选
	}
	feas := chrono.Eval(in.Source.EraBegin, in.Source.EraEnd, in.Target.EraBegin, in.Target.EraEnd)
	diffDesc := glyph.DescribeDiff(diff)

	// 借形：高度相似但有结构差异。
	if borrowing, reason := glyph.EvalBorrowing(diff); borrowing {
		return &Candidate{
			Kind:        model.KindBorrowing,
			Status:      model.RelBorrowed,
			ChronoScore: feas.Score,
			AddedParts:  diff.AddedParts,
			Removed:     diff.RemovedParts,
			DirChanged:  diff.DirChanged,
			Evidence:    fmt.Sprintf("%s；%s", diffDesc, reason),
		}, nil
	}
	// 年代逆序。
	if feas.Score < 0 {
		return &Candidate{
			Kind:        model.KindEvolution,
			Status:      model.RelChronoConflict,
			ChronoScore: feas.Score,
			AddedParts:  diff.AddedParts,
			Removed:     diff.RemovedParts,
			DirChanged:  diff.DirChanged,
			Evidence:    fmt.Sprintf("%s；%s", diffDesc, feas.Reason),
		}, nil
	}
	// 年代可行 → 演变候选。
	return &Candidate{
		Kind:        model.KindEvolution,
		Status:      model.RelCandidate,
		ChronoScore: feas.Score,
		AddedParts:  diff.AddedParts,
		Removed:     diff.RemovedParts,
		DirChanged:  diff.DirChanged,
		Evidence:    fmt.Sprintf("%s；%s", diffDesc, feas.Reason),
	}, nil
}

// BuildPairwise 对批次内全部字形两两生成候选（仅有效字形参与，组件需由调用方填充）。
// 返回生成的关系输入对（去重：同一对只生成一次，源按年代排序在前）。
func BuildPairwise(glyphs []model.Glyph) ([]CandidateInput, error) {
	chrono.SortGlyphsByEra(glyphs)
	eff := make([]model.Glyph, 0, len(glyphs))
	for _, g := range glyphs {
		if g.Status == model.GlyphValid || g.Status == model.GlyphExcluded {
			eff = append(eff, g)
		}
	}
	var pairs []CandidateInput
	for i := 0; i < len(eff); i++ {
		for j := i + 1; j < len(eff); j++ {
			pairs = append(pairs, CandidateInput{Source: eff[i], Target: eff[j]})
		}
	}
	return pairs, nil
}
