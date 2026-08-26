package service

import (
	"fmt"

	"task268-bronzeform/internal/model"
	"task268-bronzeform/internal/relation"
)

// GenerateCandidates 对批次内全部字形两两生成候选关系并落库。
func (s *Service) GenerateCandidates(batchID int64) (int, error) {
	glyphs, err := s.Glyphs.ListByBatch(batchID)
	if err != nil {
		return 0, err
	}
	// 逐字加载构件（ListByBatch 不填充 Components）。
	for i := range glyphs {
		if glyphs[i].Status != model.GlyphValid {
			continue
		}
		full, err := s.Glyphs.Get(glyphs[i].ID)
		if err != nil {
			return 0, err
		}
		glyphs[i].Components = full.Components
	}
	pairs, err := relation.BuildPairwise(glyphs)
	if err != nil {
		return 0, err
	}
	created := 0
	for _, p := range pairs {
		cand, err := relation.BuildCandidate(p)
		if err != nil {
			if err == model.ErrSelfReference {
				continue
			}
			return created, err
		}
		if cand == nil {
			continue
		}
		rel := &model.EvolutionRelation{
			BatchID:      batchID,
			SourceGlyph:  p.Source.ID,
			TargetGlyph:  p.Target.ID,
			Kind:         cand.Kind,
			Status:       cand.Status,
			ChronoScore:  cand.ChronoScore,
			AddedParts:   cand.AddedParts,
			RemovedParts: cand.Removed,
			DirChanged:   cand.DirChanged,
			Evidence:     cand.Evidence,
		}
		if err := s.Rels.Create(rel); err != nil {
			if err == model.ErrConflict {
				continue // 已有同对关系，跳过
			}
			return created, err
		}
		created++
	}
	if created > 0 {
		_ = s.Batches.UpdateStatus(batchID, model.BatchPendingReview)
	}
	return created, nil
}

// CreateManualRelation 手动登记一对字形的关系（研究者指定源/目标，自动判定候选类型）。
// 用于覆盖年代逆序（chrono_conflict）等自动两两生成不产出的分支。
func (s *Service) CreateManualRelation(batchID, sourceGlyph, targetGlyph int64) (*model.EvolutionRelation, error) {
	src, err := s.Glyphs.Get(sourceGlyph)
	if err != nil {
		return nil, err
	}
	tgt, err := s.Glyphs.Get(targetGlyph)
	if err != nil {
		return nil, err
	}
	cand, err := relation.BuildCandidate(relation.CandidateInput{Source: *src, Target: *tgt})
	if err != nil {
		return nil, err
	}
	if cand == nil {
		return nil, fmt.Errorf("%w: pair too dissimilar to form a candidate", model.ErrBadInput)
	}
	rel := &model.EvolutionRelation{
		BatchID:      batchID,
		SourceGlyph:  sourceGlyph,
		TargetGlyph:  targetGlyph,
		Kind:         cand.Kind,
		Status:       cand.Status,
		ChronoScore:  cand.ChronoScore,
		AddedParts:   cand.AddedParts,
		RemovedParts: cand.Removed,
		DirChanged:   cand.DirChanged,
		Evidence:     cand.Evidence,
	}
	if err := s.Rels.Create(rel); err != nil {
		return nil, err
	}
	return rel, nil
}

// ConfirmRelation 确认演变关系。
func (s *Service) ConfirmRelation(relID int64) error {
	r, err := s.Rels.Get(relID)
	if err != nil {
		return err
	}
	if r.Status != model.RelCandidate {
		return fmt.Errorf("%w: only candidate can be confirmed, got %s", model.ErrInvalidState, r.Status)
	}
	return s.Rels.UpdateStatus(relID, model.RelConfirmed)
}

// RejectRelation 否决演变关系。
func (s *Service) RejectRelation(relID int64) error {
	r, err := s.Rels.Get(relID)
	if err != nil {
		return err
	}
	if r.Status != model.RelCandidate && r.Status != model.RelChronoConflict && r.Status != model.RelBorrowed {
		return fmt.Errorf("%w: relation already decided, got %s", model.ErrInvalidState, r.Status)
	}
	return s.Rels.UpdateStatus(relID, model.RelRejected)
}

// MarkBorrowed 裁决为借形（同形异源）。
func (s *Service) MarkBorrowed(relID int64) error {
	r, err := s.Rels.Get(relID)
	if err != nil {
		return err
	}
	if r.Status != model.RelCandidate && r.Status != model.RelChronoConflict {
		return fmt.Errorf("%w: cannot mark borrowed, got %s", model.ErrInvalidState, r.Status)
	}
	return s.Rels.UpdateStatus(relID, model.RelBorrowed)
}

// AddRebuttal 添加反证（如后世摹刻）。
func (s *Service) AddRebuttal(relID int64, kind, note string) (*model.Rebuttal, error) {
	if kind == "" {
		kind = "other"
	}
	if note == "" {
		return nil, fmt.Errorf("%w: rebuttal note required", model.ErrBadInput)
	}
	return s.Rels.AddRebuttal(relID, kind, note)
}
