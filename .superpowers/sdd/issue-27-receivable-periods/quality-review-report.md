# Quality review — issue #27 (controller-executed programmatic, downgrade mode)

## Gates at final HEAD 919f2bfc3

- build ✓ /tmp/i27-f-build.log；lint ✓ /tmp/i27-f-lint.log；复审独立复跑 build ✓
  /tmp/i27-rev-build.log、lint ✓ /tmp/i27-rev-lint.log。
- test:e2e：sign-in+customers 冒烟失败 = 环境性（同 #25 轮裁定：三仓同时刻对照
  —— pristine base 5a4666ec7、#25（20 分钟前 sign-in 尚 PASS）、本轨，登录探针
  输出逐行同型 TypeError（header chunk namespace-switcher，.length of undefined，
  用户侧 :8888 shim 对未 mock /namespaces 的载荷所致）。非回归。验收由全端点
  mock 走查覆盖（见 spec-review-report）。
- locale parity：zh 616 = en 616，零漂移；新增 26 键 = 21 静态引用 + 5 经
  StatusBadge 模板字面量域契约解析（`${domain}.status.${value}`，与 #19 轮先例
  同口径）。

## Diff scope == plan

9 files +540/−0：query-keys(+2) / hooks(+35) / status-badge(+7) /
customer-detail(+7) / receivable-periods-tab(new 163) / external-invoice-dialog
(new 236) / idempotency.ts(new 15) / zh-CN(+37) / en(+38)。（plan 文档「8 个
文件」为笔误，其范围清单本身枚举的就是这 9 个。）无 plan 外文件；Go 模块零触碰。

## Anti-pattern scan — 0

grep console.log|debugger|TODO|FIXME|@ts-ignore|@ts-expect-error|as any|: any 于
全部改动文件：clean。eslint 0 error。无多余 helper（idempotency 助手有 #28 复用
计划，plan 明示共用）。

## In-flight defect caught by own verification loop (pre-commit)

T3 首版 customers.receivablePeriods 子树误插入顶层 subscriptions 段尾（锚点与
subscriptions 的 timeline 块相撞）——locale parity 脚本的 added-keys 计数（6≠26）
当场抓出，T3 commit 前修复并复验（residue 0）。未流入任何提交。

## Residue

worktree 仅未跟踪 ledger 与 plan 文档；test-results/ 与临时/调试 spec 已删；
无调试代码残留。

## Verdict: PASS — 0 fix rounds required.
