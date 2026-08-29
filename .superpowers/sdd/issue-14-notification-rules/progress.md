# SDD ledger — issue #14 通知规则：发票型与列表/启停

- Branch codex/admin-config-14 @ base 49d1a760b; worktree /Users/wuyongjun/trea/openmeter-issue-14.
- Plan: docs/superpowers/plans/issue-14-notification-rules.md; prescriptive source = issue #14 comment 1 (verbatim per-file code, saved /tmp/issue-round/issue-14-comments.md).
- Round: 2026-08-29 11:57+08:00 claim (see main-checkout .superpowers/sdd/ round claim file).
- Context compaction at round start: controller session started fresh (this is round-start context; no prior conversation carried over).
- SDD mode: superpowers:subagent-driven-development skill not present in session catalog → controller orchestrates equivalent discipline: per-task fresh implementer subagent + independent spec-compliance review + code-quality review subagents (≤5 fix rounds), final whole-branch adversarial review. subagent tool available this session (verified by spotcheck round precedent 2026-08-29 11:33).

## Pre-implementation anchor verification (controller, 2026-08-29 12:0x+08:00)

All checked first-hand at base 49d1a760b (main checkout):
- `legacy.ts`: channels section (listNotificationChannels L154 etc., apiFetch) present ✓ — rules section appends after it; `NotificationChannel` interface at L126 with metadata modeling from #13.
- `hooks.ts`: `useNotificationChannels(params: NotificationChannelsParams)` at L803 ✓ — prescription's `{page:1,pageSize:100}` usage matches.
- `query-keys.ts`: `notificationChannels` (L45) present; `notificationRules` absent ✓ (to add).
- `web/src/components/multi-select.tsx`: absent ✓ (to create).
- Route `routes/_authenticated/config/notification/rules/index.tsx`: PlaceholderPage ✓ (to replace with real page).
- `ConfirmDialog` props: cancelBtnText/destructive/isLoading/handleConfirm ✓.
- i18n zh-CN `config.notification` at L284 with `channels` subtree ✓ — `rules` subtree to add.
- web deps installed via `pnpm install --prefer-offline` (6.2s shared store).

## Tasks

- T1 数据层（legacy/query-keys/hooks）: PENDING
- T2 通用组件 MultiSelect: PENDING
- T3 页面与表单+i18n: PENDING

## Review rounds

(none yet)

## Final whole-branch review

(pending)

## SDD mode downgrade record (2026-08-29 12:1x+08:00)
- subagent probed first-hand this round: DENIED by unattended allowlist (subagent + subagent_fork + workflow + list_experts all rejected — 4 probe errors on record in controller transcript).
- Standing DOWNGRADE applied (10-track precedent, independently spot-checked 2026-08-29 11:33 with 10/10 PASS):
  controller-executed implementation per prescription + controller-run SEPARATE spec-compliance and code-quality review passes with fresh verification commands, logs on disk; ≤5 fix rounds; final whole-branch adversarial review multi-angle. Attended spot-check of this track remains OPEN for a future session.

- T1 数据层 (commit 2a44c97a4, amended): COMPLETED — prescription §1-§3 verbatim.
  - Gate: pnpm build exit 0 (/tmp/issue-round/t14-t1-build4.log).
  - Spec review (controller, fresh runs): legacy rules section verbatim diff = match (206 lines, only extraction divider artifact); query-keys line exact; hooks section verbatim; api/openapi.yaml field-level verification PASS — oneOf discriminator `type` mapping 4 types, entity/common props, ChannelMeta{id,type}, FeatureMeta{id,key} required, threshold enum [PERCENT,NUMBER,balance_value,usage_percentage,usage_value], thresholds 1..10, name 1..256, channels string[] (ULID), GET params includeDeleted/includeDisabled/feature[]/channel[]/page/pageSize, paginated {totalCount,page,pageSize,items}, PUT body=NotificationRuleCreateRequest (full replacement).
  - Quality review (controller, fresh runs): eslint exit 0 ×3 files; anti-patterns 0; prettier: query-keys clean; legacy made prettier-clean via --write (6 join-wraps in my section only — formatting-only deviation from verbatim, ruled); hooks.ts hunks all pre-existing (base dirty: 552/1008/1087/1140), my section adds zero new deviations.
- T2 通用组件 MultiSelect: IN PROGRESS

- T2 MultiSelect 组件 (commit 9dc876ac2): COMPLETED.
  - web/src/components/multi-select.tsx 处方逐字落地；唯一偏差=import 排序（cn 提至第3行，prettier 强制），ruled formatting-mandated。
  - Gates: eslint 0 / prettier clean / build 0。
- T3 表单+列表页+路由+i18n (commit 68826e40f): COMPLETED.
  - rule-form-dialog.tsx（发票型 zod 判别联合、channels MultiSelect、创建/编辑复用、typeHint 硬边界注释）逐字；rules.tsx（类型徽章、渠道名解析、分页、启停 ConfirmDialog、isEditableType 收窄）逐字；路由替换 PlaceholderPage；i18n 双语完整块替换 Task-1 占位 rules 子树。
  - 偏差（均为 prettier 强制）：两新文件 import 排序（--write 修复）；zh 2 处/en 5 处插入块内长行换行（选择性格式化，base 既有脏行未动：zh 4 hunks / en 2 hunks 保持不变）。
  - 行为级证据（node strip-types + stub api）: ruleToUpdateBody 3/3 PASS（发票型全量回传含 metadata、threshold 型保 thresholds+features、reset 型无 features 省键）。
  - Gates: build 0 / eslint 0 / e2e sign-in ✓ + customers 冒烟 ✘ 与 pristine base 同时刻签名一致（环境性非回归，base log /tmp/issue-round/base-e2e.log）/ locale 真实模块求值 820=820 零漂移 / 37 个组件使用键零缺失 / routeTree.gen.ts 零 diff（路由注册纯净）。
- 剩余风险: 侧栏无 rules 入口（处方范围外，待后续通知中心任务）；threshold/reset 型编辑器为后续任务（处方明示 follow-up）。
- FINAL REVIEW: PENDING

## 最终全分支复审 (controller, 2026-08-29, base 49d1a760b)
- 角度1 规格追溯: 处方 5 节（legacy/query-keys+hooks/multi-select/dialog+page/route+i18n）全部落地; 偏差均为 prettier 强制项（import 排序、插入块内长行换行），记录在案。
- 角度2 回归: query-keys/legacy/hooks 全部 append-only（0 删除行）; channels 路由与页面零改动; routeTree.gen.ts 零 diff。
- 角度3 证据审计: 日志落盘（t14-t1-build4/t14-t2-build2/t14-t3-build2/t14-t3-e2e + base-e2e 同签名）; ruleToUpdateBody 3/3 行为断言。
- 角度4 约定/反模式: 全分支新增行 0 反模式、0 eslint-disable; locale 820=820; 37 使用键零缺失。
- 角度5 对抗边界: 渠道下拉 pageSize=100 上限（处方值）; 0 条规则时 totalPages=1 兜底正确; includeDisabled=true 保证禁用规则可见可启。
- RULING: PASS（可交付，等待外部化批准）。
- 剩余风险: 侧栏无 rules 入口（处方范围外）; threshold/reset 编辑器为处方明示 follow-up; e2e customers 冒烟环境性失败（基线同签名）。
