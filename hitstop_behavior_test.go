package hitstop

import (
	"fmt"
	"testing"
)

// 覆盖：同一时刻只生效优先级最高的来源，不叠加。
func TestHigherPriorityOverrides(t *testing.T) {
	budget := New(1000)
	budget.Push("hit", 100, 0, 1)
	budget.Push("crit", 50, 0, 5)
	if got := budget.Active(0); got != 50 {
		t.Fatalf("高优先级来源应覆盖低优先级：active=%v，期望 50", got)
	}
}

// 覆盖：优先级相同按来源名升序生效。
func TestSamePriorityPicksSmallerName(t *testing.T) {
	budget := New(1000)
	budget.Push("zeta", 30, 0, 1)
	budget.Push("alpha", 80, 0, 1)
	if got := budget.Active(0); got != 80 {
		t.Fatalf("同级应取名称最小的来源：active=%v，期望 80", got)
	}
}

// 覆盖：高优先级来源到期后，低优先级来源继续生效。
func TestLowerPriorityResumesAfterTopExpires(t *testing.T) {
	budget := New(1000)
	budget.Push("hit", 100, 0, 1)
	budget.Push("crit", 50, 0, 5)
	if got := budget.Active(60); got != 40 {
		t.Fatalf("高优先级到期后低优先级应继续生效：active=%v，期望 40", got)
	}
	if got := budget.Active(100); got != 0 {
		t.Fatalf("全部到期后应为 0：active=%v", got)
	}
}

// 覆盖：续期产生新代，旧代记录不影响结果。
func TestRenewalReplacesStaleGeneration(t *testing.T) {
	budget := New(1000)
	budget.Push("hit", 100, 0, 1)
	budget.Push("hit", 40, 10, 9)
	if got := budget.Active(10); got != 40 {
		t.Fatalf("续期后应以新状态为准：active=%v，期望 40", got)
	}
	if got := budget.Active(50); got != 0 {
		t.Fatalf("续期到期后应为 0：active=%v", got)
	}
}

// 配额：同一秒内生效总量不超过 quota，超出部分截断。
func TestQuotaTruncatesOverflow(t *testing.T) {
	budget := New(120)
	if got := budget.Push("hit", 100, 0, 1); got != 100 {
		t.Fatalf("第一笔应全额生效：got=%v，期望 100", got)
	}
	if got := budget.Push("crit", 100, 0, 5); got != 20 {
		t.Fatalf("第二笔应截断到剩余配额：got=%v，期望 20", got)
	}
	if got := budget.Spent(0); got != 120 {
		t.Fatalf("当秒用量应封顶到配额：spent=%v，期望 120", got)
	}
	if got := budget.Push("more", 50, 999, 1); got != 0 {
		t.Fatalf("配额用完后应返回 0：got=%v", got)
	}
}

// 配额：整秒分桶，进入下一秒重新计算。
func TestQuotaResetsAcrossSeconds(t *testing.T) {
	budget := New(100)
	budget.Push("hit", 100, 0, 1)
	if got := budget.Push("crit", 100, 1000, 5); got != 100 {
		t.Fatalf("下一秒应重新获得完整配额：got=%v，期望 100", got)
	}
	if got := budget.Spent(0); got != 100 {
		t.Fatalf("上一秒的用量不应被改写：spent=%v，期望 100", got)
	}
	if got := budget.Spent(1000); got != 100 {
		t.Fatalf("当前秒的用量应独立记账：spent=%v，期望 100", got)
	}
}

// 到期：Active 返回剩余毫秒，过期返回 0，不为负。
func TestExpiryReturnsZero(t *testing.T) {
	budget := New(1000)
	budget.Push("hit", 100, 0, 1)
	if got := budget.Active(40); got != 60 {
		t.Fatalf("应返回剩余毫秒：active=%v，期望 60", got)
	}
	if got := budget.Active(100); got != 0 {
		t.Fatalf("到期边界应为 0：active=%v", got)
	}
	if got := budget.Active(5000); got != 0 {
		t.Fatalf("过期后应为 0 而不是负数：active=%v", got)
	}
}

// 不变量：同一来源不再续期时 Active 单调不增。
func TestActiveNonIncreasingWithoutRenewal(t *testing.T) {
	budget := New(1000)
	budget.Push("hit", 100, 0, 1)
	previous := budget.Active(0)
	for now := int64(1); now <= 200; now++ {
		if got := budget.Active(now); got > previous {
			t.Fatalf("Active 不应回升：t=%v 时 %v > %v", now, got, previous)
		} else {
			previous = got
		}
	}
}

// 不变量：同一串请求重复执行结果相同。
func TestDeterministicReplay(t *testing.T) {
	run := func() (int64, int64, int64) {
		budget := New(120)
		first := budget.Push("hit", 100, 0, 1)
		second := budget.Push("crit", 100, 0, 5)
		return first, second, budget.Active(0)
	}
	a1, a2, a3 := run()
	b1, b2, b3 := run()
	if a1 != b1 || a2 != b2 || a3 != b3 {
		t.Fatalf("重复执行结果应一致：(%v,%v,%v) vs (%v,%v,%v)",
			a1, a2, a3, b1, b2, b3)
	}
}

// 代价：扫描量不随来源数线性增长。
func TestScannedDoesNotScaleWithSources(t *testing.T) {
	budget := New(1000000)
	for index := 0; index < 3000; index++ {
		budget.Push(fmt.Sprintf("s-%d", index), 1, 0, 1)
	}
	for index := 0; index < 3000; index++ {
		budget.Active(0)
	}
	if got := budget.Scanned(); got > 6000 {
		t.Fatalf("扫描量不应随来源数线性增长：scanned=%v，期望 <= 6000", got)
	}
}

// 代价：大量来源过期后，扫描量摊还有界。
func TestScannedAmortizedAfterExpiry(t *testing.T) {
	budget := New(1000000)
	for index := 0; index < 100000; index++ {
		budget.Push(fmt.Sprintf("s-%d", index), 10, 0, 1)
	}
	for index := 0; index < 100000; index++ {
		budget.Active(100000)
	}
	if got := budget.Scanned(); got > 200000 {
		t.Fatalf("过期清理应摊还有界：scanned=%v，期望 <= 200000", got)
	}
	if got := budget.Active(100000); got != 0 {
		t.Fatalf("全部过期后应为 0：active=%v", got)
	}
}
