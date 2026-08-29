# Spec 符合性审查报告 — issue #11（分遍 1，新鲜命令）

审查人：控制器分遍审查（Standing DOWNGRADE 模式，见 progress.md SDD 模式节）。
方法：独立于实施记忆，从处方源与计划重新推导要求，用新鲜 git/grep/node 命令
逐锚点核验；证据日志在 /tmp/issue-round3/。

## Task 1（bd942b64c，BASE 6aaa5b916）

要求源：处方步骤 1（hooks 逐字）+ 步骤 5（i18n）+ 计划 Ruling-2（query-keys
补位）+ 任务简报。

| # | 要求 | 证据（新鲜命令） | 判定 |
|---|---|---|---|
| 1 | hooks.ts 追加 Plan addons 分节=处方逐字 | 处方 L50-115 代码块 vs diff 新增行 diff：仅 1 行节尾空行差（`64a65 > 空`），代码体逐字一致（/tmp/issue-round3/t11-hooks-verdict.txt） | ✅ |
| 2 | 四 hooks 命名/签名/onSuccess 失效 nsPrefix('plan-addons')/enabled 门控 | 同上逐字比对覆盖 | ✅ |
| 3 | query-keys 补 planAddons 处方原式 | diff 输出：`planAddons: (planId: string, params: object = {}) => ns('plan-addons', planId, params)` | ✅ |
| 4 | i18n zh planAddons 键集=处方步骤 5 | 去缩进逐行 diff：除提取窗口头部 planDetail 两行预期噪声外 IDENTICAL（t11-presc-zh.txt vs t11-actual-zh.txt） | ✅ |
| 5 | config.planDetail.tabs 新子树（zh/en） | node 求值：zh {overview:'概览',addons:'附加组件'} / en {Overview, Add-ons} | ✅ |
| 6 | zh/en 同构 + 全文件奇偶 | node 求值：zh=993 en=993，only-zh=0 only-en=0；config.planAddons 27=27 | ✅ |
| 7 | 提交面=恰好 4 文件纯追加 | numstat：65/2/44/47 insertions，deletions 全 0 | ✅ |
| 8 | 未越界（无 useAllAddons、未动 useAddons/其他分节） | numstat 0 删除 + 新增行全在 planAddons/planDetail/Plan addons 分节 | ✅ |
| 9 | build/lint 门禁 | t11-t1-build.log exit 0；t11-t1-lint.log exit 0；git status 无 routeTree 变更 | ✅ |
| 10 | Ruling-2 条件成立性复核 | main 实测 query-keys 无 planAddons（本轮 pre-flight 已核） | ✅ |

**Task 1 判定：SPEC PASS，无发现，零修复轮。**

## Task 2（26eeb9e5a，BASE bd942b64c；提交含一次 prettier 自查 amend）

要求源：处方步骤 2（PlanAddonFormDialog 逐字）+ 步骤 3（PlanAddonsTab 逐字）+
步骤 4（plan-detail 包 Tabs）+ 计划 Ruling-1（useAddons 信封适配）+ T1 产物。

| # | 要求 | 证据（新鲜命令） | 判定 |
|---|---|---|---|
| 1 | dialog = 处方逐字（除 Ruling-1） | 处方 L125-447 vs 文件 diff：仅 3 处 Ruling-1 适配（import useAllAddons→useAddons；data 解构；`(addonsData?.data ?? [])` 扁平化）+ 提取窗口首尾行（useEffect import/收尾 `}`）；无其他差异 | ✅ |
| 2 | tab = 处方逐字（除 Ruling-1） | 处方 L457-649 vs 文件 diff：仅同型 3 处 Ruling-1 + 窗口行；无其他差异 | ✅ |
| 3 | plan-detail overview 内容等价移入 | 去缩进 diff：base 卡片+阶段块 vs TabsContent 内块——差异仅 1 处 JSX 换行（priceType TableHead，prettier 深缩进产物）；prettier 两侧格式化对照 +114/-70 恰为包裹+重缩进 | ✅ |
| 4 | Tabs 挂载锚点 | defaultValue='overview' L184；addons trigger L189；`<PlanAddonsTab planId={plan.id} />` L294 | ✅ |
| 5 | AC 锚点：增删改 | create 按钮 + 编辑（editTarget 回填）+ 删除 ConfirmDialog（destructive + toast） | ✅ |
| 6 | spec 语义约束 | 编辑态 addon Select disabled（不可换关联）；maxQuantity 仅 isMultiple 可填、single 提交省略；selectedAddon 默认回填 name 仅 isCreate；create 提交体含 addon {id}，update 体无 addon 字段 | ✅ |
| 7 | 新组件 t() 键覆盖 | node 求值 34/34 键存在于 zh（en 由奇偶保证） | ✅ |
| 8 | build/lint/e2e | t11-t2-build2/lint2 exit 0；e2e sign-in ✓ + customers ✘ 与 pristine 基线（e2e-base.log）同签名同测试行同错误类 → 环境性非回归（历轮共识） | ✅ |

**Task 2 判定：SPEC PASS，无发现，零修复轮。**（实施自查发现新文件 prettier 不
clean → 已修复并 amend，重验 build/lint 双 0；属实施内修正，非审查发现。）
