// Package service 编排各领域包与存储层，实现批次级业务用例：
// 批次流转、字形导入/拆分、候选生成、关系裁决、版本发布。
package service

import (
	"task268-bronzeform/internal/store"
)

// Service 聚合全部存储与用例方法。
type Service struct {
	Batches  *store.BatchStore
	Glyphs   *store.GlyphStore
	Rels     *store.RelationStore
	Versions *store.VersionStore
}

// New 创建服务。
func New(b *store.BatchStore, g *store.GlyphStore, r *store.RelationStore, v *store.VersionStore) *Service {
	return &Service{Batches: b, Glyphs: g, Rels: r, Versions: v}
}


