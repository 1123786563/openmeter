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
