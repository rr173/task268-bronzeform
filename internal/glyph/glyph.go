// Package glyph 实现金文字形的构件分解与构形比较：
// 构件增删检测、方向（镜像/旋转）变化检测、以及构形相似度评分。
package glyph

import (
	"fmt"
	"sort"
	"strings"

	"task268-bronzeform/internal/model"
)

// CompareResult 两个字形构形比较的结果。
type CompareResult struct {
	AddedParts   []string // 目标字形新增的构件
	RemovedParts []string // 目标字形相比源字形缺失的构件
	DirChanged   []string // 方向发生变化的共有构件
	SharedParts  int      // 共有构件数
	TotalParts   int      // 构件并集数
	Similarity   float64  // 构形相似度 [0,1]
}

// CompareComponents 比较源字形与目标字形的构件集合。
// 相似度 = 2*共有 / (源构件数 + 目标构件数)（Dice 系数）。
func CompareComponents(src, dst []model.Component) CompareResult {
	srcMap := indexByPart(src)
	dstMap := indexByPart(dst)
	res := CompareResult{}

	for part, sc := range srcMap {
		if dc, ok := dstMap[part]; ok {
			res.SharedParts++
			if sc.Direction != dc.Direction {
				res.DirChanged = append(res.DirChanged, part)
			}
		} else {
			res.RemovedParts = append(res.RemovedParts, part)
		}
	}
	for part := range dstMap {
		if _, ok := srcMap[part]; !ok {
			res.AddedParts = append(res.AddedParts, part)
		}
	}
	sort.Strings(res.AddedParts)
	sort.Strings(res.RemovedParts)
	sort.Strings(res.DirChanged)

	res.TotalParts = len(srcMap) + len(dstMap) - res.SharedParts
	if res.TotalParts == 0 {
		res.Similarity = 1.0
	} else {
		res.Similarity = 2.0 * float64(res.SharedParts) / float64(len(srcMap)+len(dstMap))
	}
	return res
}

// indexByPart 按构件名建立索引（同名单个构件取第一条）。
func indexByPart(comps []model.Component) map[string]model.Component {
	m := make(map[string]model.Component, len(comps))
	for _, c := range comps {
		if _, ok := m[c.Part]; !ok {
			m[c.Part] = c
		}
	}
	return m
}

// SplitGraph 把字形描述拆分为构件列表。
// 输入形如 "⿰金⿱立口"（⿰ 左右结构、⿱ 上下结构），输出构件名+方位。
func SplitGraph(graph string) ([]model.Component, error) {
	var comps []model.Component
	runes := []rune(graph)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		switch ch {
		case '⿰', '⿱', '⿲', '⿳':
			if i+2 >= len(runes) {
				return nil, fmt.Errorf("%w: graph %q truncated structure at %d", model.ErrBadInput, graph, i)
			}
			left, right := runes[i+1], runes[i+2]
			comps = append(comps, model.Component{Part: string(left), Position: posName(ch, 0)})
			comps = append(comps, model.Component{Part: string(right), Position: posName(ch, 1)})
			i += 2
		default:
			comps = append(comps, model.Component{Part: string(ch), Position: "center"})
		}
	}
	if len(comps) == 0 {
		return nil, fmt.Errorf("%w: empty graph %q", model.ErrBadInput, graph)
	}
	return comps, nil
}

func posName(structure rune, idx int) string {
	switch structure {
	case '⿰':
		if idx == 0 {
			return "left"
		}
		return "right"
	case '⿱':
		if idx == 0 {
			return "top"
		}
		return "bottom"
	case '⿲':
		if idx == 0 {
			return "left"
		}
		return "right"
	case '⿳':
		if idx == 0 {
			return "top"
		}
		return "bottom"
	default:
		return "center"
	}
}

// DescribeDiff 生成人类可读的构形差异描述。
func DescribeDiff(res CompareResult) string {
	parts := []string{}
	if len(res.AddedParts) > 0 {
		parts = append(parts, "增构件:"+strings.Join(res.AddedParts, "、"))
	}
	if len(res.RemovedParts) > 0 {
		parts = append(parts, "省构件:"+strings.Join(res.RemovedParts, "、"))
	}
	if len(res.DirChanged) > 0 {
		parts = append(parts, "方向变化:"+strings.Join(res.DirChanged, "、"))
	}
	if len(parts) == 0 {
		return "构形一致"
	}
	return strings.Join(parts, "；")
}

// EvalBorrowing 评估两个字形是否构成借形候选：
// 构形高度相似（相似度 ≥0.8）且存在关键差异（构件增删或方向变化）时，提示可能为借形而非演变。
// 构形完全一致（无增删、无方向变化）不构成借形，应作为普通演变候选处理。
func EvalBorrowing(res CompareResult) (isBorrowingCandidate bool, reason string) {
	if res.Similarity < 0.8 {
		return false, "构形相似度不足，不构成借形候选"
	}
	if len(res.AddedParts) > 0 || len(res.RemovedParts) > 0 || len(res.DirChanged) > 0 {
		return true, "构形高度相似但存在结构差异，可能为同形异源借形"
	}
	return false, "构形完全一致，为普通演变候选而非借形"
}
