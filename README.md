# task268-bronzeform 金文构形演变证据复核台

基于 Go 实现的金文构形演变证据复核 Web 项目：导入青铜器铭文字形观察、拆分构件、比较构件增删与方向变化、校验器物年代可行性、记录借形反证并发布不可变演变关系版本。

## 业务闭环

1. 创建铭文批次（organizing）→ 导入字形观察（器物年代 + 拓片来源，指纹幂等去重）。
2. 构件拆分（⿰/⿱ 结构解析，pending_split → valid；残缺拓片 → defective → excluded）。
3. 提交比较（→ pending_compare）→ 两两生成候选：演变候选 / 借形候选 / 年代冲突。
4. 研究者裁决：确认（confirmed）、否决（rejected，可附摹刻反证）、借形（borrowed）。
5. 创建演变版本（draft）→ 共享（shared）→ 冻结（frozen，内容哈希校验）→ 批次发布/封存。

## 状态机

- 铭文批次：organizing → pending_compare → pending_review → published → sealed
- 字形观察：pending_split → valid / defective → excluded
- 演变关系：candidate → confirmed / rejected；借形候选 → borrowed；年代逆序 → chrono_conflict
- 演变版本：draft → shared → frozen → superseded

## 标准命令

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...
go run ./cmd/task268-bronzeform --smoke-test --db smoke.db
go run ./cmd/task268-bronzeform --addr :8080 --db bronzeform.db
```

## 示例

```bash
# 创建批次
curl -s -X POST localhost:8080/api/batches -d '{"code":"BX-001","name":"西周金文·旅字构形演变"}'
# 导入字形（带构件，直接 valid）
curl -s -X POST localhost:8080/api/glyphs -d '{"batch_id":1,"code":"G1","graph":"旅","era_begin":900,"era_end":850,"source":"毛公鼎","components":[{"part":"𠂉","position":"top","direction":"normal"},{"part":"从","position":"bottom","direction":"normal"}]}'
# 提交并生成候选
curl -s -X POST localhost:8080/api/batches/1/submit
curl -s -X POST localhost:8080/api/batches/1/generate
# 确认关系、创建版本并冻结
curl -s -X POST localhost:8080/api/relations/1/confirm
curl -s -X POST localhost:8080/api/versions -d '{"batch_id":1,"name":"演变定本 v1"}'
curl -s -X POST localhost:8080/api/versions/1/freeze
```

## 持久化

SQLite（modernc.org/sqlite v1.52.0，纯 Go 驱动，CGO 无关）。表：batches / glyphs / components / relations / rebuttals / versions / version_relations。WAL 模式 + busy_timeout，崩溃恢复由 SQLite 保证；`--smoke-test` 关闭重开同一数据库验证重启恢复。
