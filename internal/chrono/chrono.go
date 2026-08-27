// Package chrono 实现器物年代区间与演变方向的年代可行性校验。
// 核心：金文演变必须满足“源字形年代不晚于目标字形年代”的约束，
// 年代逆序的关系标记为 chrono_conflict（冲突）。
//
// 年代约定：采用公元前（BC）纪年数值，数值越大年代越早
// （如 850 BC 早于 800 BC）。区间 [Begin, End] 表示器物存在于 Begin 至 End 年。
package chrono

import (
	"sort"

	"task268-bronzeform/internal/model"
)

// Feasibility 年代可行性判定结果。
type Feasibility struct {
	Score    int    // >0 可行；<0 逆序（源晚于目标）；0 无重叠依据
	Reason   string // 人类可读依据
	Overlap  bool   // 两区间是否重叠
	StrictOK bool   // 严格可行：源整体早于目标
}

// Eval 评估源字形年代区间到目标字形年代的演变可行性。
// 判定规则（公元前纪年，数值越大越早）：
//   - src.End > dst.Begin：源最晚年代早于目标最早年代 → 源整体早于目标，严格可行 score=+2
//   - src.Begin < dst.End：源最早年代晚于目标最晚年代 → 源整体晚于目标，严格逆序 score=-2
//   - src.Begin >= dst.Begin：区间重叠且源起点不晚于目标起点 → 弱可行 score=+1
//   - 其余：区间重叠但源起点晚于目标起点 → 弱逆序 score=-1
func Eval(srcBegin, srcEnd, dstBegin, dstEnd int) Feasibility {
	f := Feasibility{Score: 0}
	switch {
	case srcEnd > dstBegin:
		f.Score = 2
		f.StrictOK = true
		f.Reason = "源字形年代整体早于目标字形，演变方向可行"
	case srcBegin < dstEnd:
		f.Score = -2
		f.Reason = "源字形年代整体晚于目标字形，演变方向逆序，不可行"
	case srcBegin >= dstBegin:
		f.Score = 1
		f.Overlap = true
		f.Reason = "源字形年代起点不晚于目标字形，区间重叠，弱可行"
	default:
		f.Score = -1
		f.Overlap = true
		f.Reason = "源字形年代起点晚于目标字形，区间重叠但方向可疑"
	}
	return f
}

// SortGlyphsByEra 按年代排序字形（公元前纪年，数值越大年代越早；
// 演变源须不晚于目标，故数值大（早）的在前，用作候选生成顺序）。
func SortGlyphsByEra(glyphs []model.Glyph) {
	sort.SliceStable(glyphs, func(i, j int) bool {
		if glyphs[i].EraBegin != glyphs[j].EraBegin {
			return glyphs[i].EraBegin > glyphs[j].EraBegin
		}
		return glyphs[i].EraEnd > glyphs[j].EraEnd
	})
}

// ValidateInterval 校验年代区间合法性：begin >= end（公元前纪年，起点数值更大）。
func ValidateInterval(begin, end int) error {
	if begin < end {
		return model.ErrEraInverted
	}
	return nil
}
