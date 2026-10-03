// Package hitstop 是顿帧预算：多来源覆盖、优先级、每秒配额与到期清理。
// 缺陷：多个来源叠加而不是取最高优先级、每秒配额不生效、来源永不过期、
// 到期时间不清理、每次查询遍历全部来源。
package hitstop

// Budget 是顿帧管理器。
type Budget struct {
	quota   int64
	until   map[string]int64
	prio    map[string]int64
	used    map[int64]int64
	scanned int
}

// New 建管理器，quota 是每秒允许的顿帧总量。
func New(quota int64) *Budget {
	return &Budget{quota: quota, until: map[string]int64{}, prio: map[string]int64{},
		used: map[int64]int64{}}
}

// Push 请求一次顿帧，返回实际生效的毫秒数。
// 缺陷：配额只记账不截断。
func (b *Budget) Push(source string, ms, nowMs, priority int64) int64 {
	b.scanned += len(b.until)
	b.until[source] = nowMs + ms
	b.prio[source] = priority
	b.used[nowMs/1000] += ms
	return ms
}

// Active 返回当前生效的顿帧时长。
// 缺陷：把所有来源的剩余时长相加，也不看到期。
func (b *Budget) Active(nowMs int64) int64 {
	b.scanned += len(b.until)
	total := int64(0)
	for source, until := range b.until {
		_ = source
		total += until - nowMs
	}
	return total
}

// Spent 返回某一秒已经用掉的顿帧毫秒数。
func (b *Budget) Spent(nowMs int64) int64 { return b.used[nowMs/1000] }

// Scanned 是累计扫描量（规模观测）。
func (b *Budget) Scanned() int { return b.scanned }
