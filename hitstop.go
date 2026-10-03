// Package hitstop 是顿帧预算：多来源覆盖、优先级、每秒配额与到期清理。
package hitstop

import "container/heap"

// sourceState 是一个来源当前生效的状态。
type sourceState struct {
	priority int64
	until    int64
	gen      int64
}

// heapItem 是优先级堆里的一条记录；续期会产生新的代，旧记录惰性丢弃。
type heapItem struct {
	source   string
	priority int64
	until    int64
	gen      int64
}

// budgetHeap 优先级高者在前，同级按来源名升序，再同代按新代在前。
type budgetHeap []heapItem

func (h budgetHeap) Len() int { return len(h) }

func (h budgetHeap) Less(i, j int) bool {
	if h[i].priority != h[j].priority {
		return h[i].priority > h[j].priority
	}
	if h[i].source != h[j].source {
		return h[i].source < h[j].source
	}
	return h[i].gen > h[j].gen
}

func (h budgetHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *budgetHeap) Push(x any) { *h = append(*h, x.(heapItem)) }

func (h *budgetHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

// Budget 是顿帧管理器。
type Budget struct {
	quota   int64
	current map[string]sourceState
	order   *budgetHeap
	gen     int64
	used    map[int64]int64
	scanned int
}

// New 建管理器，quota 是每秒允许的顿帧总量。
func New(quota int64) *Budget {
	order := &budgetHeap{}
	heap.Init(order)
	return &Budget{quota: quota, current: map[string]sourceState{},
		order: order, used: map[int64]int64{}}
}

// Push 请求一次顿帧，返回实际生效的毫秒数（受每秒配额截断）。
func (b *Budget) Push(source string, ms, nowMs, priority int64) int64 {
	second := nowMs / 1000
	granted := ms
	if remaining := b.quota - b.used[second]; granted > remaining {
		granted = remaining
	}
	if granted < 0 {
		granted = 0
	}
	b.used[second] += granted

	b.gen++
	state := sourceState{priority: priority, until: nowMs + granted, gen: b.gen}
	b.current[source] = state
	heap.Push(b.order, heapItem{
		source:   source,
		priority: priority,
		until:    state.until,
		gen:      b.gen,
	})
	return granted
}

// Active 返回当前生效来源（优先级最高、同级名称最小）的剩余毫秒，过期返回 0。
func (b *Budget) Active(nowMs int64) int64 {
	for b.order.Len() > 0 {
		b.scanned++
		item := (*b.order)[0]
		state, ok := b.current[item.source]
		if !ok || state.gen != item.gen {
			heap.Pop(b.order)
			continue
		}
		if state.until <= nowMs {
			heap.Pop(b.order)
			delete(b.current, item.source)
			continue
		}
		return state.until - nowMs
	}
	return 0
}

// Spent 返回某一秒已经用掉的顿帧毫秒数。
func (b *Budget) Spent(nowMs int64) int64 { return b.used[nowMs/1000] }

// Scanned 是累计扫描量（规模观测），只统计 Active 检查过的堆条目数。
func (b *Budget) Scanned() int { return b.scanned }
