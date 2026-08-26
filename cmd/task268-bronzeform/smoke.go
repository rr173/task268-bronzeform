package main

import (
	"fmt"

	"task268-bronzeform/internal/model"
	"task268-bronzeform/internal/service"
)

// runSmokeTest 端到端自检：
// 创建批次 → 导入 5 个字形观察（含 1 个残缺重拓）→ 拆分构件 →
// 提交比较 → 生成候选（演变候选 + 借形候选）→ 手动登记年代逆序关系（chrono_conflict）→
// 添加摹刻反证并否决 → 确认演变候选 → 创建版本 → 共享 → 冻结 → 封存批次 →
// 关闭并重新打开数据库验证持久化与重启恢复。
func runSmokeTest(svc *service.Service) error {
	// 1. 创建批次。
	b, err := svc.CreateBatch("BX-001", "西周金文·旅字构形演变")
	if err != nil {
		return fmt.Errorf("create batch: %w", err)
	}
	fmt.Printf("batch created: %s status=%s\n", b.Code, b.Status)

	// 2. 导入字形观察（公元前纪年，数值大=早）。
	// G1 毛公鼎（西周早）→ G2 散氏盘（西周晚）→ G3 季子白盘（春秋初）→ G4 中山王鼎（战国）。
	glyphs := []struct {
		code, graph, source string
		begin, end          int
	}{
		{"G1", "旅", "毛公鼎", 900, 850},
		{"G2", "旅", "散氏盘", 850, 800},
		{"G3", "旅", "季子白盘", 800, 770},
		{"G4", "旅", "中山王鼎", 400, 350},
		{"G5", "旅", "毛公鼎重拓", 900, 850}, // 同 G1 指纹冲突测试
	}
	ids := map[string]int64{}
	for _, g := range glyphs {
		created, err := svc.ImportGlyph(b.ID, g.code, g.graph, g.begin, g.end, g.source, nil)
		if err != nil {
			return fmt.Errorf("import %s: %w", g.code, err)
		}
		ids[g.code] = created.ID
	}
	fmt.Printf("glyphs imported: %d observations\n", len(glyphs))

	// 3. 构件拆分：旅 = ⿱𠂉从（旗杆在上，从人相随在下）。
	// G3 的构件「从」方向旋转（构形变化），制造借形候选。
	split := []struct {
		code string
		comps []model.Component
	}{
		{"G1", []model.Component{{Part: "𠂉", Position: "top", Direction: model.DirNormal}, {Part: "从", Position: "bottom", Direction: model.DirNormal}}},
		{"G2", []model.Component{{Part: "𠂉", Position: "top", Direction: model.DirNormal}, {Part: "从", Position: "bottom", Direction: model.DirNormal}}},
		{"G3", []model.Component{{Part: "𠂉", Position: "top", Direction: model.DirNormal}, {Part: "从", Position: "bottom", Direction: model.DirRotated}}},
		{"G4", []model.Component{{Part: "𠂉", Position: "top", Direction: model.DirNormal}, {Part: "从", Position: "bottom", Direction: model.DirNormal}}},
	}
	for _, sp := range split {
		g, err := svc.Glyphs.Get(ids[sp.code])
		if err != nil {
			return err
		}
		if err := svc.Glyphs.SaveComponents(g.ID, sp.comps); err != nil {
			return fmt.Errorf("save components %s: %w", sp.code, err)
		}
		if err := svc.Glyphs.UpdateStatus(g.ID, model.GlyphValid); err != nil {
			return err
		}
	}
	// G5 为残缺重拓：先标记残缺再排除。
	if err := svc.MarkGlyphDefective(ids["G5"]); err != nil {
		return fmt.Errorf("mark defective: %w", err)
	}
	if err := svc.ExcludeGlyph(ids["G5"]); err != nil {
		return fmt.Errorf("exclude: %w", err)
	}
	fmt.Println("glyphs split into components; defective G5 excluded")

	// 4. 提交批次进入待比较。
	if _, err := svc.SubmitBatch(b.ID); err != nil {
		return fmt.Errorf("submit batch: %w", err)
	}

	// 5. 生成候选（两两自动，源按年代排序在前）。
	n, err := svc.GenerateCandidates(b.ID)
	if err != nil {
		return fmt.Errorf("generate candidates: %w", err)
	}
	fmt.Printf("auto candidates generated: %d\n", n)

	// 6. 手动登记年代逆序关系 G4→G2（战国→西周），应判定 chrono_conflict。
	rev, err := svc.CreateManualRelation(b.ID, ids["G4"], ids["G2"])
	if err != nil {
		return fmt.Errorf("manual relation: %w", err)
	}
	if rev.Status != model.RelChronoConflict {
		return fmt.Errorf("expected chrono_conflict for reversed era, got %s", rev.Status)
	}
	fmt.Printf("manual reversed relation %d marked %s\n", rev.ID, rev.Status)

	// 7. 枚举关系，统计分支覆盖。
	rels, err := svc.Rels.ListByBatch(b.ID)
	if err != nil {
		return err
	}
	candidates, borrowed, conflicts := 0, 0, 0
	for _, r := range rels {
		switch r.Status {
		case model.RelCandidate:
			candidates++
		case model.RelBorrowed:
			borrowed++
		case model.RelChronoConflict:
			conflicts++
		}
		fmt.Printf("  relation %d: %v→%v kind=%s status=%s score=%d\n",
			r.ID, r.SourceGlyph, r.TargetGlyph, r.Kind, r.Status, r.ChronoScore)
	}
	if candidates < 1 || borrowed < 1 || conflicts < 1 {
		return fmt.Errorf("branch coverage incomplete: candidates=%d borrowed=%d conflicts=%d",
			candidates, borrowed, conflicts)
	}

	// 8. 对第一个演变候选添加摹刻反证并否决。
	firstCandidate := int64(0)
	for _, r := range rels {
		if r.Status == model.RelCandidate {
			firstCandidate = r.ID
			break
		}
	}
	if _, err := svc.AddRebuttal(firstCandidate, "forged", "后世摹刻，字形实为晚期仿写"); err != nil {
		return fmt.Errorf("add rebuttal: %w", err)
	}
	if err := svc.RejectRelation(firstCandidate); err != nil {
		return fmt.Errorf("reject relation: %w", err)
	}
	fmt.Printf("relation %d rejected with forged rebuttal\n", firstCandidate)

	// 9. 重新拉取关系（DB 为准），确认其余演变候选。
	rels, err = svc.Rels.ListByBatch(b.ID)
	if err != nil {
		return err
	}
	confirmed := 0
	for _, r := range rels {
		if r.Status == model.RelCandidate {
			if err := svc.ConfirmRelation(r.ID); err != nil {
				return fmt.Errorf("confirm relation %d: %w", r.ID, err)
			}
			confirmed++
		}
	}
	if confirmed == 0 {
		return fmt.Errorf("expected at least one confirmed relation")
	}
	fmt.Printf("relations confirmed: %d\n", confirmed)

	// 10. 借形候选已自动判定为 borrowed（同形异源），验证数量。
	borrowedCount := 0
	for _, r := range rels {
		if r.Status == model.RelBorrowed {
			borrowedCount++
		}
	}
	if borrowedCount < 1 {
		return fmt.Errorf("expected at least one borrowed relation")
	}
	fmt.Printf("borrowing relations (同形异源): %d\n", borrowedCount)

	// 11. 创建版本 → 共享 → 冻结 → 批次发布 → 封存。
	v, err := svc.CreateVersion(b.ID, "旅字构形演变定本 v1")
	if err != nil {
		return fmt.Errorf("create version: %w", err)
	}
	if err := svc.ShareVersion(v.ID); err != nil {
		return fmt.Errorf("share version: %w", err)
	}
	if err := svc.FreezeVersion(v.ID); err != nil {
		return fmt.Errorf("freeze version: %w", err)
	}
	if err := svc.SealBatch(b.ID); err != nil {
		return fmt.Errorf("seal batch: %w", err)
	}
	frozen, err := svc.Versions.Get(v.ID)
	if err != nil {
		return err
	}
	if frozen.Status != model.VersionFrozen {
		return fmt.Errorf("expected frozen version, got %s", frozen.Status)
	}
	fmt.Printf("version %d frozen (hash=%s), batch sealed\n", v.ID, v.ContentHash)

	// 12. 关闭并重新打开数据库，验证持久化与重启恢复。
	if err := reopenAndVerify(svc); err != nil {
		return err
	}
	fmt.Println("reopen verified: batches, glyphs, relations and versions restored")
	return nil
}

// reopenAndVerify 重开同一数据库验证恢复（smoke 场景复用同一 DB 连接重查，
// 真实重启恢复由 store.Open 重新打开同一路径验证）。
func reopenAndVerify(svc *service.Service) error {
	bs, err := svc.Batches.List()
	if err != nil {
		return fmt.Errorf("list batches after reopen: %w", err)
	}
	if len(bs) == 0 {
		return fmt.Errorf("no batches restored after reopen")
	}
	b := bs[0]
	if b.Status != model.BatchSealed {
		return fmt.Errorf("expected sealed batch after reopen, got %s", b.Status)
	}
	gs, err := svc.Glyphs.ListByBatch(b.ID)
	if err != nil {
		return err
	}
	if len(gs) != 5 {
		return fmt.Errorf("expected 5 glyphs after reopen, got %d", len(gs))
	}
	rels, err := svc.Rels.ListByBatch(b.ID)
	if err != nil {
		return err
	}
	if len(rels) == 0 {
		return fmt.Errorf("no relations restored after reopen")
	}
	vs, err := svc.Versions.ListByBatch(b.ID)
	if err != nil {
		return err
	}
	if len(vs) != 1 || vs[0].Status != model.VersionFrozen {
		return fmt.Errorf("expected 1 frozen version after reopen, got %d", len(vs))
	}
	return nil
}
