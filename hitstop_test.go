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
