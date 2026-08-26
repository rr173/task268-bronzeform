package chrono

import "testing"

func TestEvalStrictFeasible(t *testing.T) {
	// 源 [900,850] → 目标 [800,770]：源整体早于目标（公元前数值大=早）。
	f := Eval(900, 850, 800, 770)
	if f.Score != 2 {
		t.Fatalf("expected score 2, got %d (%s)", f.Score, f.Reason)
	}
	if !f.StrictOK {
		t.Fatal("expected strict ok")
	}
}

func TestEvalStrictReversed(t *testing.T) {
	// 源 [400,350] → 目标 [900,850]：源整体晚于目标 → 逆序。
	f := Eval(400, 350, 900, 850)
	if f.Score != -2 {
		t.Fatalf("expected score -2, got %d (%s)", f.Score, f.Reason)
	}
}

func TestEvalOverlapWeakFeasible(t *testing.T) {
	// 源 [900,800] → 目标 [850,770]：区间重叠，源起点不晚于目标起点 → 弱可行。
	f := Eval(900, 800, 850, 770)
	if f.Score != 1 {
		t.Fatalf("expected score 1, got %d (%s)", f.Score, f.Reason)
	}
	if !f.Overlap {
		t.Fatal("expected overlap")
	}
}

func TestEvalOverlapWeakReversed(t *testing.T) {
	// 源 [850,770] → 目标 [900,800]：区间重叠但源起点晚于目标起点 → 弱逆序。
	f := Eval(850, 770, 900, 800)
	if f.Score != -1 {
		t.Fatalf("expected score -1, got %d (%s)", f.Score, f.Reason)
	}
}

func TestValidateInterval(t *testing.T) {
	// 公元前纪年：begin（数值大）应 >= end（数值小）。
	if err := ValidateInterval(900, 850); err != nil {
		t.Fatalf("valid interval rejected: %v", err)
	}
	if err := ValidateInterval(850, 900); err == nil {
		t.Fatal("inverted interval accepted")
	}
}
