# SDD ledger — plan: docs/superpowers/plans/issue-22-apps-catalog-install.md

- 轨道：issue #22 应用目录+安装表单；worktree /Users/wuyongjun/trea/openmeter-issue-22；
  分支 codex/admin-config-22；base f60cb90b0。
- Issue: https://github.com/1123786563/openmeter/issues/22

## Preflight 冲突扫描（dispatch 前）

| 交叠对 | 检查结论 |
|---|---|
| T1 hooks ↔ T2 组件 | T1 产出 useAppCatalog/useInstallApp/queryKeys.appCatalog，T2 消费三者；命名已在计划 Interfaces 固定，一致 |
| T2 ↔ index.tsx 既有页 | 仅追加目录区挂载（installed 列表之后），#21 的已装列表/卸载/换 Key 零触碰 |
| T2 i18n ↔ 既有 config.apps 子树 | 只追加 catalog/install 子键，不改动既有键 |
| 本轨 ↔ 并行轨 #24/#26/#28 | 不同 worktree；共享面仅主仓无——各 worktree 独立 hooks.ts/query-keys.ts 副本，合并期按 Issue 号序处理 locale 文件并集（先例 #9+#14） |
| 计划文本自洽 | Ruling 已记录 createBillingProfile 必填 boolean（SDK 事实 vs 总纲笔误）；非 stripe 分支固定 false |

Ruling: install 三分支 createBillingProfile 均必填（SDK InstallApp* 定义），
非 stripe 表单不渲染开关、提交 false —— 按 Issue 文案「仅名称」执行；若后端
对 sandbox 亦支持建档案则 UI 未暴露，代价=功能隐藏，无正确性风险。

## SDD 模式

首 dispatch 探测 subagent 工具；被拒则 Standing DOWNGRADE（控制器实施+分遍审查）。

## BASE

- T1 BASE: f60cb90b0

## T1 数据层

- Implementer: fresh subagent（3e510d19）DONE_WITH_CONCERNS @ 76ecccf9c
  （+queryKeys.appCatalog/+useAppCatalog/+useInstallApp；build/lint exit 0）。
- Concerns：(1) SDK dist 环境事项（见 Ruling 环境段）；(2) useAppCatalog 未传
  分页参数——AppCatalogItem.type 联合仅 3 类，默认页大小远超目录规模，留待
  任务审查裁定。
- 审查中：review-ba20de72f..76ecccf9c.diff。

- 审查（e9b1a074，独立分遍双轴）：SPEC PASS 无发现；QUALITY PASS 无发现。
  Minor 备忘（不构成修复要求）：useAppCatalog 依赖服务端默认页大小，目录
  ≤3 类够用；T2 如需全量可切 listCatalogAll（SDK internal.ts:250-259）——
  记为 T2 已知决策点。
- T1 complete（76ecccf9c，clean，零修复轮）。

## T2 目录区 + 安装 dialog + 挂载 + i18n

- Implementer: fresh subagent（3996ef78）DONE @ a99dd981b（5 文件 +438/-1；
  build/lint 双 0；config.apps 子树 zh/en 52=52；新文件 prettier clean）。
- 自查备忘：① 目录表 installMethods 列表头空（brief 键集封闭）；② locale/
  index.tsx HEAD 本就非 prettier-clean，保持纯插入；③ 浏览器 mock 走查留终态。
- 审查中：review-76ecccf9c..a99dd981b.diff。

- 审查（d3a0c470）：SPEC PASS 无发现（验收锚点全在位）；QUALITY FAIL——
  Q1 Important 必修：schema 内 message 为裸键串/裸 zod 默认串，FormMessage
  （form.tsx:137 error 优先）渲染字面量 'config.apps.install.validation.
  apiKeyRequired' 与未翻译默认串，children 翻译永不生效。修复方向（审查者
  给出）：createInstallSchema(isStripe, t) 注入文案，per-type 重挂键已保证
  resolver 重建；勿改 ui/form.tsx。
- Ruling：Q2（installMethods 列空表头，plan-mandated 键集缺口）由控制器修订
  计划并入修复轮 1 补键；Q3（查询失败=空态，沿 #21 模式）/Q4（重挂键下
  reset effect 冗余，无害）/Q5（行 key=item.type 契约内成立）记录不改。
- 跨轨备忘：同病（schema message 未本地化）存在于已合并 #21 的
  stripe-key-dialog.tsx:118-122——不扩本轨范围，记入轮报告遗留项。
- 修复轮 1 派发（fresh implementer）。

## 终态门禁

- 修复轮 1（dfb95c64）：86b831bc8——Q1 schema 注入 t（useMemo([isStripe,t])）
  + FormMessage 死 children 删除；Q2 installMethodsLabel 补键（计划已修订）；
  build/lint 双 0；locale parity zh=843=en=843；复审派发（3d64e46b）。
- e2e（/tmp/issue-round/e2e-22.log）：sign-in smoke PASS；customers smoke FAIL
  与原始基线同签名（e2e-base.log）→ 既有环境问题非本轨回归。

- 修复轮 1 复审（3d64e46b）：FIX PASS 无发现（t 注入闭环/语言切换 resolver
  重建经 react-i18next v16 与 RHF 7.72 源码级实证/Q2 键同构+表头+计划同步/
  恰 5 文件零漂移）。

- 全分支终审（7501a17f）：FINAL RELEASE_READY 无发现（失效链路/SDK 三分支
  createBillingProfile 必填实证、验收逐条、纯插入、修复轮终态、i18n 求值
  zh=en=843 对称差集空；已知 Minor 维持记录）。
- 轨道终态：T1 76ecccf9c + T2 a99dd981b + fix1 86b831bc8，门禁全绿。
