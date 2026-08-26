package service

import (
	"fmt"

	"task268-bronzeform/internal/glyph"
	"task268-bronzeform/internal/model"
)

// ImportGlyph 导入字形观察：校验器物来源、年代区间与自包含，计算指纹去重。
func (s *Service) ImportGlyph(batchID int64, code, graph string, eraBegin, eraEnd int, source string, comps []model.Component) (*model.Glyph, error) {
	if source == "" {
		return nil, model.ErrMissingSource
	}
	if err := validateInterval(eraBegin, eraEnd); err != nil {
		return nil, err
	}
	if err := validateComponents(comps); err != nil {
		return nil, err
	}
	g := &model.Glyph{
		BatchID:  batchID,
		Code:     code,
		Graph:    graph,
		EraBegin: eraBegin,
		EraEnd:   eraEnd,
		Source:   source,
	}
	g.Components = comps
	g.ComponentSum = model.ComponentSumOf(comps)
	g.Fingerprint = model.FingerprintOf(graph, source, g.ComponentSum, eraBegin, eraEnd)
	if err := s.Glyphs.Create(g); err != nil {
		return nil, err
	}
	if len(comps) > 0 {
		if err := s.Glyphs.SaveComponents(g.ID, comps); err != nil {
			return nil, err
		}
		if err := s.Glyphs.UpdateStatus(g.ID, model.GlyphValid); err != nil {
			return nil, err
		}
		g.Status = model.GlyphValid
	}
	return g, nil
}

// SplitGlyph 对字形做构件拆分：解析字形描述 → 保存构件 → 置为 valid。
func (s *Service) SplitGlyph(glyphID int64, graph string) (*model.Glyph, error) {
	g, err := s.Glyphs.Get(glyphID)
	if err != nil {
		return nil, err
	}
	if g.Status != model.GlyphPendingSplit {
		return nil, fmt.Errorf("%w: glyph must be pending_split to split, got %s", model.ErrInvalidState, g.Status)
	}
	comps, err := glyph.SplitGraph(graph)
	if err != nil {
		return nil, err
	}
	if err := s.Glyphs.SaveComponents(glyphID, comps); err != nil {
		return nil, err
	}
	if err := s.Glyphs.UpdateStatus(glyphID, model.GlyphValid); err != nil {
		return nil, err
	}
	g.Components = comps
	g.Status = model.GlyphValid
	return g, nil
}

// MarkGlyphDefective 标记字形为残缺（不可作为演变来源）。
func (s *Service) MarkGlyphDefective(glyphID int64) error {
	g, err := s.Glyphs.Get(glyphID)
	if err != nil {
		return err
	}
	if g.Status == model.GlyphExcluded || g.Status == model.GlyphDefective {
		return fmt.Errorf("%w: glyph already %s", model.ErrInvalidState, g.Status)
	}
	return s.Glyphs.UpdateStatus(glyphID, model.GlyphDefective)
}

// ExcludeGlyph 排除字形观察。
func (s *Service) ExcludeGlyph(glyphID int64) error {
	g, err := s.Glyphs.Get(glyphID)
	if err != nil {
		return err
	}
	if g.Status == model.GlyphExcluded {
		return fmt.Errorf("%w: glyph already excluded", model.ErrInvalidState)
	}
	return s.Glyphs.UpdateStatus(glyphID, model.GlyphExcluded)
}

func validateInterval(begin, end int) error {
	// 公元前纪年，起点数值更大（如 900 BC 早于 850 BC）。
	if begin < end {
		return model.ErrEraInverted
	}
	return nil
}

func validateComponents(comps []model.Component) error {
	seen := map[string]bool{}
	for _, c := range comps {
		if c.Part == "" {
			return fmt.Errorf("%w: component part required", model.ErrBadInput)
		}
		key := fmt.Sprintf("%s@%s", c.Part, c.Position)
		if seen[key] {
			return fmt.Errorf("%w: duplicate component %s", model.ErrBadInput, key)
		}
		seen[key] = true
	}
	return nil
}
