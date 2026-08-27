// Package version 实现演变版本的生命周期管理：
// 收集已裁决关系生成版本内容哈希、冻结不可变发布、替代旧版本。
package version

import (
	"fmt"
	"sort"

	"task268-bronzeform/internal/model"
)

// SnapshotCollector 版本快照收集：从批次内已裁决关系构建版本内容。
type SnapshotCollector struct {
	BatchID     int64
	Relations   []model.EvolutionRelation
	Name        string
	ContentHash string
}

// CollectDecided 收集批次内全部已裁决（confirmed/borrowed）关系作为版本内容。
// 仅纳入 confirmed 与 borrowed（有效结论），排除 rejected / candidate / chrono_conflict。
func CollectDecided(rels []model.EvolutionRelation, name string) (*SnapshotCollector, error) {
	decided := make([]model.EvolutionRelation, 0, len(rels))
	for _, r := range rels {
		switch r.Status {
		case model.RelConfirmed, model.RelBorrowed:
			decided = append(decided, r)
		}
	}
	if len(decided) == 0 {
		return nil, fmt.Errorf("%w: no decided relations to snapshot", model.ErrBadInput)
	}
	sort.Slice(decided, func(i, j int) bool { return decided[i].ID < decided[j].ID })
	sc := &SnapshotCollector{Name: name, Relations: decided}
	if len(decided) > 0 {
		sc.BatchID = decided[0].BatchID
	}
	sc.ContentHash = sc.computeHash()
	return sc, nil
}

// computeHash 计算版本内容哈希（关系 ID 有序拼接）。
func (sc *SnapshotCollector) computeHash() string {
	h := newVersionHash()
	for _, r := range sc.Relations {
		h.addInt(r.ID).addInt(r.SourceGlyph).addInt(r.TargetGlyph).
			add(string(r.Kind)).add(string(r.Status))
	}
	return h.sum()
}

// RelationIDs 返回快照内的关系 ID 列表。
func (sc *SnapshotCollector) RelationIDs() []int64 {
	ids := make([]int64, 0, len(sc.Relations))
	for _, r := range sc.Relations {
		ids = append(ids, r.ID)
	}
	return ids
}
