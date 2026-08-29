# Spec review — issue #25 (controller-executed programmatic, downgrade mode)

Reviewed against: issue #25 body + acceptance, issue #25 comment 1 (prescriptive
plan), master plan Task 13 first half. Evidence: /tmp/i25-walkthrough3.log
(acceptance walkthrough, 1 passed), SDK dist field checks (this round), final
diff base 5a4666ec7..106a59bc4.

## Acceptance item 1: 可创建账单档案并出现在列表 — PASS (wire-level)

Walkthrough (fully mocked endpoints, immune to :8888 shim):
- List page renders row: 名称/描述/供应商名+税号/三 apps 显示名（id→name 解析自
  GET /api/v3/openmeter/apps）/默认徽章/创建时间 (formatDateTime)。
- 新建档案 dialog：空提交被 zod 前端拦截（aria-invalid，POST 计数 0）。
- 填表（name/法定名称/税号/key/cn→CN/地址行1 + 三槽位下拉 + default 关）提交：
  恰好 1 次 POST /api/v3/openmeter/profiles，wire 体 deep-equal：
  {name:'新档案', supplier:{name, key, tax_id:{code}, addresses:{billing_address:
  {country:'CN', line1}}}, workflow:{}, apps:{tax:{id},invoicing:{id},payment:{id}},
  default:false} —— 空串可选字段正确省略，camelCase→snake_case 由 SDK 序列化。
- toast「账单档案已创建」→ dialog 关闭 → 列表失效重取（创建后 GET ≥1）。

## Contract checks (14/14 vs SDK dist + prescriptive plan)

1. listProfiles request page{number,size} ✓  2. ProfilePagePaginatedResponse ✓
3. createProfile required 5 项 ✓  4. name 1-256 ✓  5. description ≤1024 ✓
6. Party{key?,name?,taxId.code?,addresses.billingAddress} ✓
7. Address 7 可选字段 ✓  8. WorkflowInput 全可选→{} 合法 ✓
9. ProfileAppReferences 三槽位必填 {id} ✓  10. AppReference={id} ✓
11. Profile 列表字段（含 createdAt:Date）✓  12. default 冲突后端裁决（错误 toast
原文透出，UI 无客户端唯一性判断）✓  13. internal.apps.list 请求形状 ✓
14. useBillingProfile enabled=Boolean(id) 随单交付（#26 回填用，plan 明示）✓

## Deviations (accepted)

- D-1: plan 的 `api.apps.list` → `api.internal.apps.list`（根 client 无 apps 命名
  空间，dist/sdk/sdk.d.ts getters 实证；与 #21 D-1 同型；#21 未合入故本轨自定义
  useApps，实现与 #21 逐字同形，合并零冲突）。
- D-2: plan i18n 块的 form.validation.{required,country,key} 三键省略——plan 自身
  zod 代码用字面量消息（'invalid'），不经过 i18n，三键必为死键；本仓 locale 检查
  约束新增键全部被引用。错误显示保持 zod 默认。

## Walkthrough test-side iterations (product unchanged)

run1: 列表 apps 名断言用了 dialog 选项的后缀写法（列表仅显示名）→ 修断言；
run2: getByLabel('名称') strict 冲突（含'法定名称'）→ exact:true；run3 PASS。

## Verdict: PASS — 0 fix rounds required.
