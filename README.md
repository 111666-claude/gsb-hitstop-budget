# hitstop-budget

顿帧预算：多来源覆盖、优先级、每秒配额与到期清理，只用 Go 标准库。

```
go run ./cmd/hitstop-budget --sample=priority
go run ./cmd/hitstop-budget --sample=quota
go run ./cmd/hitstop-budget --sample=expire
go run ./cmd/hitstop-budget --sample=scan
go test ./...
go vet ./...
```

## 口径（README 为准）

- **覆盖而不是叠加**：同一时刻只生效优先级最高的来源；优先级相同按来源名升序。
- **到期**：来源的顿帧结束后不再计入，`Active` 返回剩余毫秒，过期返回 0。
- **每秒配额**：同一秒内已经生效的顿帧总量不超过 `quota`；
  超出部分必须截断，`Push` 返回实际生效的毫秒数。
- **跨秒重置**：配额按整秒分桶，进入下一秒重新计算。
- **不变量**：`Active` 单调不增（同一来源不再续期时）；同一串请求重复执行结果相同。
- **代价**：`Active` 不许遍历全部来源，`Scanned` 不随来源数线性增长。
- **规模**：10 万个来源、每秒 10 万次查询，单次摊还 O(活跃来源数)。

## 输出契约（不改格式）

```
active=..
second=..
scanned=..
```

## 目录

```
hitstop.go              覆盖、配额与到期
hitstop_test.go         单元测试
cmd/hitstop-budget/     命令行入口
```
