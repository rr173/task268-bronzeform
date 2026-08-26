基于 Go 实现的金文构形演变证据复核 Web 项目，一款后端服务，导入青铜器铭文字形观察并拆分构件、比较构件增删与方向变化、校验器物年代可行性与借形反证并发布不可变演变关系版本。

# BENZHI 评测说明

金文构形演变证据复核台（纯后端）。

## 运行契约

- **启动服务**：`/app/bronzeform --addr :8080 --db bronzeform.db`
- **端到端自检**：`/app/bronzeform --smoke-test --db smoke.db`
  - 真实创建铭文批次「BX-001 西周金文·旅字构形演变」、导入 5 个字形观察（毛公鼎/散氏盘/季子白盘/中山王鼎/残缺重拓）、
    拆分「旅=⿱𠂉从」构件（含 1 处方向变化），排除残缺重拓，
    触发两两候选生成（演变候选 + 借形候选 + 手动年代逆序 chrono_conflict）、
    添加摹刻反证并否决、确认演变候选、创建并冻结演变版本、封存批次，
    关闭并重新打开同一数据库验证持久化与重启恢复，最终以退出码 0 结束。
  - 这是 Docker `CMD` 与双架构验证的唯一判据，**只传 flag，不传路径位置参数**。

## Docker 双架构验证

```bash
# amd64
docker buildx build --platform linux/amd64 --load -t bronzeform:amd64 .
docker run --rm bronzeform:amd64 --smoke-test

# arm64
docker buildx build --platform linux/arm64 --load -t bronzeform:arm64 .
docker run --rm bronzeform:arm64 --smoke-test
```

## API 一览（前缀 /api）

| 能力 | 方法与路径 | 说明 |
| --- | --- | --- |
| 创建批次 | POST /api/batches | 新建铭文批次（code 唯一） |
| 批次列表 | GET /api/batches | 全部批次 |
| 批次详情 | GET /api/batches/{id} | 单批次 |
| 提交比较 | POST /api/batches/{id}/submit | organizing → pending_compare |
| 生成候选 | POST /api/batches/{id}/generate | 两两比较生成演变/借形候选 |
| 封存批次 | POST /api/batches/{id}/seal | published → sealed |
| 导入字形 | POST /api/glyphs | 导入字形观察（指纹幂等） |
| 字形列表 | GET /api/glyphs?batch_id= | 批次内字形 |
| 字形详情 | GET /api/glyphs/{id} | 含构件 |
| 构件拆分 | POST /api/glyphs/{id}/split | 解析 ⿰⿱ 结构 → valid |
| 标记残缺 | POST /api/glyphs/{id}/defective | → defective |
| 排除字形 | POST /api/glyphs/{id}/exclude | → excluded |
| 手动登记关系 | POST /api/relations | 指定源/目标自动判定候选 |
| 确认演变 | POST /api/relations/{id}/confirm | candidate → confirmed |
| 否决演变 | POST /api/relations/{id}/reject | → rejected |
| 裁决借形 | POST /api/relations/{id}/borrow | → borrowed（同形异源） |
| 添加反证 | POST /api/relations/{id}/rebuttals | 摹刻/误释反证 |
| 关系列表 | GET /api/relations?batch_id= | 批次内关系（含反证数） |
| 创建版本 | POST /api/versions | 收集已裁决关系 → draft |
| 版本列表 | GET /api/versions?batch_id= | 批次内版本 |
| 共享版本 | POST /api/versions/{id}/share | draft → shared |
| 冻结版本 | POST /api/versions/{id}/freeze | → frozen（校验内容哈希） |
| 替代版本 | POST /api/versions/{id}/supersede | frozen → superseded |
| 统计 | GET /api/stats | 批次/字形/关系计数 |
| 健康检查 | GET /api/health | ok |

## 核心不变量

- 批次 code、字形指纹（SHA-256）、(batch,source,target) 关系对均唯一。
- 年代区间不逆序（公元前纪年，begin ≥ end）；器物来源必填；源目标不得相同。
- 演变方向必须满足源字形年代不晚于目标（chrono_score>0），逆序自动标记 chrono_conflict。
- 借形候选：构形高度相似（相似度≥0.8）但存在构件增删/方向变化 → borrowed（同形异源）。
- 冻结版本内容哈希不可变、不可再流转（仅可被更新版本 supersede）；封存批次只读。
- 反证（如后世摹刻）可否决候选关系并留痕。

## 目录

- `cmd/task268-bronzeform/main.go`：入口（--addr/--db/--smoke-test）
- `internal/model`：实体、状态机、错误、指纹
- `internal/store`：SQLite 迁移与 CRUD（modernc.org/sqlite v1.52.0）
- `internal/glyph`：构件拆分、增删/方向比较、借形评估
- `internal/chrono`：年代区间与演变方向可行性
- `internal/relation`：候选生成与裁决
- `internal/version`：版本快照与内容哈希
- `internal/service`：批次/字形/关系/版本编排
- `internal/httpapi`：HTTP 路由（含日志与 panic 恢复中间件）
