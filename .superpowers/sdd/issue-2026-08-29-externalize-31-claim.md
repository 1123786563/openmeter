# 外化轮台账 — #31（运行锁 acquiredAt=2026-08-29T14:36:32Z）

## 第 0 步 — 运行互斥

- `run-lock.json` 不存在（上轮 22:30 +0800 已释放）→ 直取运行权，写入 acquiredAt=2026-08-29T14:36:32Z。

## 批准凭据

- 用户在对话中直接回复「**批准**」——对应 wait-round-31-claim-2.md 等待事项 1 的指定通道（如「批准外发 #31」），凭记录执行外发链。

## 第 1 步 — 外化前新鲜核验（全过）

- worktree issue-31 干净（0 改动）、tip=b40d256c7 匹配台账、分支未 push（ls-remote 空）、main=origin/main=158251056 零漂移。
- 合并前置（Ruling-2）：6 个未跟踪副本 cmp 复验逐字节一致 6/6 后 rm（零信息损失）。

## 第 2 步 — 普查

- open = {#31}（本轮外化对象），无其他候选 → 0 新领取；外化后 open = 0。

## 外化链执行记录（按计划「外部化预案」5 步）

| 步骤 | 结果 | 证据 |
|---|---|---|
| 1. push 分支 | ✅ | codex/admin-config-31 → origin，远端 tip b40d256c7 |
| 2. rm 未跟踪副本 | ✅ | 6/6 逐字节一致复验后删除；目录内其余 tracked 文件未动 |
| 3. merge --no-ff | ✅ | f9b682cf6，零冲突，8 文件 +1483/-1；ls-files 6 恢复文件全命中；`/server` 锚定对根二进制仍生效、auth.go 不再被忽略 |
| 4. 合并后门禁 | ✅ | go build ./... exit 0；go vet ./... exit 0；三包 go test 3/3 ok；make lint-go exit 2 = 恰 5 发现与既档集合逐项一致（4 项 main 预存 + Ruling-4 parked），**零新增** |
| 5. 关单 + push main | ✅（见下） | 附证据评论 close --reason completed；main 推送 fork |

## Rulings 消费

- Ruling-2（rm 前置）本轮消费 ✅；Ruling-4（sloglint parked）保持 parked，lint 对照证明集合无变化，归未来 lint 清理轮（备忘延续，非阻塞）。

## 轨道终态

- #31：RELEASED（本地完成四要素 + 外化五步全过）。
- 分支 codex/admin-config-31 已推 origin 并合入 main（--no-ff f9b682cf6）。
- closed 集 = {1..31} 全清；open = 0。
- SDD 工作区工件（progress.md 含外化记录、final-review.md、briefs/reports、review diffs）随本 docs 提交强制入库（复刻 #29 外化模式 158251056）。
- 运行锁于本轮结束时删除。
