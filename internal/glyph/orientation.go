package glyph

import "task268-bronzeform/internal/model"

// OrientationDelta 描述构件方向变化。
type OrientationDelta struct {
	Part      string
	From      model.ComponentDirection
	To        model.ComponentDirection
	IsChange  bool
}

// CompareOrientation 比较同一构件在两个字形中的方向。
// 返回变化标记与人类可读说明（用于证据链记录）。
func CompareOrientation(part string, from, to model.ComponentDirection) OrientationDelta {
	d := OrientationDelta{Part: part, From: from, To: to}
	if from != to {
		d.IsChange = true
	}
	return d
}

// DirectionDescription 生成方向变化的证据描述。
func DirectionDescription(d OrientationDelta) string {
	if !d.IsChange {
		return ""
	}
	names := map[model.ComponentDirection]string{
		model.DirNormal:   "正",
		model.DirMirrored: "镜像",
		model.DirRotated:  "旋转",
	}
	return "构件「" + d.Part + "」方向由" + names[d.From] + "变为" + names[d.To]
}

// AllDirectionChanges 汇总一组构件方向变化描述。
func AllDirectionChanges(deltas []OrientationDelta) []string {
	out := make([]string, 0, len(deltas))
	for _, d := range deltas {
		if s := DirectionDescription(d); s != "" {
			out = append(out, s)
		}
	}
	return out
}
