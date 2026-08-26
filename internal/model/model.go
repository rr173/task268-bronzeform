// Package model 定义金文构形演变证据复核台的领域实体、状态机与业务错误。
package model

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

// BatchStatus 铭文批次状态机。
// 流转：organizing → pending_compare → pending_review → published → sealed。
type BatchStatus string

const (
	BatchOrganizing     BatchStatus = "organizing"      // 整理中：录入字形观察
	BatchPendingCompare BatchStatus = "pending_compare" // 待比较：提交后进入候选生成
	BatchPendingReview  BatchStatus = "pending_review"  // 待裁决：候选已生成，等待研究者复核
	BatchPublished      BatchStatus = "published"       // 已发布：演变版本冻结
	BatchSealed         BatchStatus = "sealed"          // 封存：只读终态
)

// GlyphStatus 字形观察状态机。
// 流转：pending_split → valid | defective → excluded。
type GlyphStatus string

const (
	GlyphPendingSplit GlyphStatus = "pending_split" // 待拆分：未做构件分解
	GlyphValid        GlyphStatus = "valid"         // 有效：构件已拆分且可用于比较
	GlyphDefective    GlyphStatus = "defective"     // 残缺：拓片残缺，不可作为演变来源
	GlyphExcluded     GlyphStatus = "excluded"      // 排除：被研究者排除
)

// RelationKind 演变关系类别。
type RelationKind string

const (
	KindEvolution RelationKind = "evolution" // 构形演变
	KindBorrowing RelationKind = "borrowing" // 借形（同形异源）
)

// RelationStatus 演变关系状态机。
// 候选生成后：candidate → confirmed | rejected；年代倒置/借形直接标 borrowed；年代不可行标 chrono_conflict。
type RelationStatus string

const (
	RelCandidate      RelationStatus = "candidate"       // 候选：待裁决
	RelChronoConflict RelationStatus = "chrono_conflict" // 年代冲突：源晚于目标，不可作演变
	RelBorrowed       RelationStatus = "borrowed"        // 借形：同形但为借形关系
	RelConfirmed      RelationStatus = "confirmed"       // 确认：演变关系成立
	RelRejected       RelationStatus = "rejected"        // 否决：演变关系不成立
)

// VersionStatus 演变版本状态机。
// 流转：draft → shared → frozen → superseded。
type VersionStatus string

const (
	VersionDraft      VersionStatus = "draft"      // 草稿：可编辑
	VersionShared     VersionStatus = "shared"     // 共享：证据已公示
	VersionFrozen     VersionStatus = "frozen"     // 冻结：不可变发布
	VersionSuperseded VersionStatus = "superseded" // 替代：被新版本取代
)

// ComponentDirection 构件方向（镜像/旋转等构形变化）。
type ComponentDirection string

const (
	DirNormal   ComponentDirection = "normal"   // 正
	DirMirrored ComponentDirection = "mirrored" // 镜像
	DirRotated  ComponentDirection = "rotated"  // 旋转
)

// Batch 铭文批次：一批青铜器铭文字形观察的集合。
type Batch struct {
	ID        int64       `json:"id"`
	Code      string      `json:"code"`
	Name      string      `json:"name"`
	Status    BatchStatus `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// Glyph 字形观察：单件器物上的一个金文字形，含器物年代与拓片来源。
type Glyph struct {
	ID           int64       `json:"id"`
	BatchID      int64       `json:"batch_id"`
	Code         string      `json:"code"`
	Graph        string      `json:"graph"` // 字形描述（如 金、𠂤、旅）
	EraBegin     int         `json:"era_begin"`
	EraEnd       int         `json:"era_end"`
	Source       string      `json:"source"` // 器物来源（如 毛公鼎）
	Status       GlyphStatus `json:"status"`
	Fingerprint  string      `json:"fingerprint"` // 观察指纹（幂等去重）
	Components   []Component `json:"components,omitempty"`
	ComponentSum string      `json:"component_sum"` // 构件摘要（指纹输入）
	CreatedAt    time.Time   `json:"created_at"`
}

// Component 构件：字形分解出的偏旁/部首单元，含方向变化。
type Component struct {
	ID        int64             `json:"id"`
	GlyphID   int64             `json:"glyph_id"`
	Part      string            `json:"part"`
	Position  string            `json:"position"`
	Direction ComponentDirection `json:"direction"`
	Seq       int               `json:"seq"`
}

// EvolutionRelation 演变关系：source → target 的构形演变候选。
type EvolutionRelation struct {
	ID           int64          `json:"id"`
	BatchID      int64          `json:"batch_id"`
	SourceGlyph  int64          `json:"source_glyph"`
	TargetGlyph  int64          `json:"target_glyph"`
	Kind         RelationKind   `json:"kind"`
	Status       RelationStatus `json:"status"`
	ChronoScore  int            `json:"chrono_score"` // 年代可行性评分（>0 可行，<0 逆序）
	AddedParts   []string       `json:"added_parts,omitempty"`
	RemovedParts []string       `json:"removed_parts,omitempty"`
	DirChanged   []string       `json:"dir_changed,omitempty"`
	Evidence     string         `json:"evidence"`
	RebutCount   int            `json:"rebut_count"`
	CreatedAt    time.Time      `json:"created_at"`
	DecidedAt    *time.Time     `json:"decided_at,omitempty"`
}

// Rebuttal 反证：对候选关系的反驳证据（摹刻、误释等）。
type Rebuttal struct {
	ID         int64     `json:"id"`
	RelationID int64     `json:"relation_id"`
	Kind       string    `json:"kind"` // forged（后世摹刻）/ miscopy（误释）/ other
	Note       string    `json:"note"`
	CreatedAt  time.Time `json:"created_at"`
}

// EvolutionVersion 演变版本：关系裁决结果的不可变快照。
type EvolutionVersion struct {
	ID           int64          `json:"id"`
	BatchID      int64          `json:"batch_id"`
	Name         string         `json:"name"`
	Status       VersionStatus  `json:"status"`
	ContentHash  string         `json:"content_hash"`
	RelationIDs  []int64        `json:"relation_ids,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	FrozenAt     *time.Time     `json:"frozen_at,omitempty"`
	SupersededAt *time.Time     `json:"superseded_at,omitempty"`
}

// FingerprintOf 计算字形观察指纹：器物来源+字形+构件摘要+年代区间 的 SHA-256。
func FingerprintOf(graph, source, compSum string, eraBegin, eraEnd int) string {
	raw := fmt.Sprintf("%s|%s|%s|%d|%d", graph, source, compSum, eraBegin, eraEnd)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// ComponentSumOf 计算构件摘要：按 构件名+方位+方向 排序拼接。
func ComponentSumOf(comps []Component) string {
	parts := make([]string, 0, len(comps))
	for _, c := range comps {
		parts = append(parts, fmt.Sprintf("%s@%s:%s", c.Part, c.Position, c.Direction))
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}
