package hitstop

import "testing"

func TestEmptyBudgetIsInactive(t *testing.T) {
	if New(1000).Active(0) != 0 {
		t.Fatal("没有来源时不该有顿帧")
	}
}

func TestPushReturnsMs(t *testing.T) {
	if got := New(1000).Push("hit", 80, 0, 0); got != 80 {
		t.Fatalf("第一次请求应该拿到 80：%v", got)
	}
}

func TestQuotaIsStored(t *testing.T) {
	if New(250).quota != 250 {
		t.Fatal("配额应该能读回来")
	}
}

func TestSpentStartsAtZero(t *testing.T) {
	if New(1000).Spent(0) != 0 {
		t.Fatal("没有请求时用量应该是 0")
	}
}

func TestScannedStartsAtZero(t *testing.T) {
	if New(1000).Scanned() != 0 {
		t.Fatal("Scanned 初始应该是 0")
	}
}

func TestHigherPriorityOverrides(t *testing.T) {
	budget := New(1000)
	budget.Push("hit", 100, 0, 1)
	budget.Push("crit", 50, 0, 5)
	if got := budget.Active(0); got != 50 {
		t.Fatalf("高优先级应该覆盖而不是叠加：%v", got)
	}
	if got := budget.Active(30); got != 20 {
		t.Fatalf("覆盖来源的剩余时间应该倒计时：%v", got)
	}
}

func TestSamePriorityPicksSmallerName(t *testing.T) {
	budget := New(1000)
	budget.Push("b", 100, 0, 1)
	budget.Push("a", 40, 0, 1)
	if got := budget.Active(0); got != 40 {
		t.Fatalf("同级应该按来源名升序取 a：%v", got)
	}
}

func TestQuotaTruncates(t *testing.T) {
	budget := New(120)
	if got := budget.Push("hit", 100, 0, 1); got != 100 {
		t.Fatalf("第一次应该全额生效：%v", got)
	}
	if got := budget.Push("crit", 100, 0, 5); got != 20 {
		t.Fatalf("超出配额应该截断到 20：%v", got)
	}
	if got := budget.Push("more", 100, 0, 9); got != 0 {
		t.Fatalf("配额用完后应该生效 0：%v", got)
	}
	if got := budget.Spent(0); got != 120 {
		t.Fatalf("这一秒用量应该是 120：%v", got)
	}
}

func TestQuotaResetsNextSecond(t *testing.T) {
	budget := New(120)
	budget.Push("hit", 120, 0, 1)
	if got := budget.Push("crit", 100, 1000, 5); got != 100 {
		t.Fatalf("进入下一秒配额应该重置：%v", got)
	}
	if got := budget.Spent(1000); got != 100 {
		t.Fatalf("下一秒用量应该重新计算：%v", got)
	}
}

func TestExpiryReturnsZero(t *testing.T) {
	budget := New(1000)
	budget.Push("hit", 100, 0, 1)
	if got := budget.Active(100); got != 0 {
		t.Fatalf("到期瞬间应该返回 0：%v", got)
	}
	if got := budget.Active(5000); got != 0 {
		t.Fatalf("过期后应该返回 0：%v", got)
	}
}

func TestActiveCountsDownAndNeverNegative(t *testing.T) {
	budget := New(1000)
	budget.Push("hit", 100, 0, 1)
	previous := int64(101)
	for now := int64(0); now <= 200; now += 10 {
		got := budget.Active(now)
		if got < 0 {
			t.Fatalf("Active 不该为负：%v", got)
		}
		if got > previous {
			t.Fatalf("不再续期时 Active 应该单调不增：%v -> %v", previous, got)
		}
		previous = got
	}
}

func TestRenewalReplacesOldEntry(t *testing.T) {
	budget := New(1000)
	budget.Push("hit", 100, 0, 1)
	budget.Push("hit", 200, 0, 1)
	if got := budget.Active(0); got != 200 {
		t.Fatalf("续期应该以最新请求为准：%v", got)
	}
}

func TestDeterministicReplay(t *testing.T) {
	run := func() [3]int64 {
		budget := New(120)
		budget.Push("hit", 100, 0, 1)
		granted := budget.Push("crit", 100, 0, 5)
		return [3]int64{granted, budget.Active(0), budget.Active(5000)}
	}
	if run() != run() {
		t.Fatal("同一串请求重复执行结果应该相同")
	}
}

func TestScannedDoesNotGrowWithSources(t *testing.T) {
	budget := New(1000000)
	for index := 0; index < 100000; index++ {
		budget.Push(string(rune(index)), 1, 0, 1)
	}
	for index := 0; index < 100000; index++ {
		if got := budget.Active(0); got <= 0 {
			t.Fatalf("查询应该看到生效来源：%v", got)
		}
	}
	if scanned := budget.Scanned(); scanned > 200000 {
		t.Fatalf("Scanned 不该随来源数线性增长：%v", scanned)
	}
}
