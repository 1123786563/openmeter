# T2 规格符合性审查（分遍 1）— 2026-08-29

审查域：eab5a1d67..5f9e18274（events.tsx 新建 575 行 + 路由占位替换）。
方法：与处方步骤 4/5 逐字归一化比对（difflib 全差异枚举）+ t() 键覆盖 +
锚点复核 + 门禁新鲜证据。

## 逐项结论

1. 路由文件 vs 处方步骤 5：**归一化 IDENTICAL**（占位 PlaceholderPage →
   NotificationEventsPage 直挂）。
2. events.tsx vs 处方步骤 4：全差异枚举恰五类，全部为 formatter/授权项：
   - import 顺序重排（@trivago/prettier-plugin-sort-imports 之 importOrder
     规则：page-header 后置、@/api 组内 hooks 先于 legacy）——formatter 强制；
   - Tailwind class 顺序重排 ×2（break-all 位置，prettier-plugin-tailwindcss）；
   - 数组尾逗号移除 ×1（prettier trailingComma 归一）；
   - **PD1 修复**（Ruling）：处方 L96 `useState(toLocalInputValue(new Date(
     Date.now()-…)))` 惰性求值表达式在 render 期调用 Date.now 触发
     react-hooks/purity error → 改为 lazy initializer `useState(() => …)`
     （行为等价：首渲染求值一次；附注释）。此为处方缺陷，非转写误差；
   - 其余 9000+ 归一化字符 IDENTICAL（含 DELIVERY_STATE_CLASS、
     eventOverrides 单条刷新、ConfirmDialog+MultiSelect、1.5s refetch、
     分页/空态/骨架）。
3. t() 键覆盖：静态 27 键全部命中 zh-CN（递归存在性校验 NONE missing）；
   动态模板键 `config.notification.rules.types.${event.type}` 四值全在
   （#15 产物复用 ✓）。
4. AC 锚点：「事件列表按过滤条件出数」= 过滤草稿→应用→单选升数组→
   listNotificationEvents→setPage(1) ✓；「resend 后投递状态可见变化（或
   明确错误 toast）」= 202 受理 toast + invalidate(notification.events) +
   1.5s refetch（RESENDING/状态变化可见）+ onError handleServerError ✓。
5. 范围：diff 仅 2 文件（577+/9-）；未触 T1 文件与 notification 域其他文件。
6. 门禁新鲜证据：build2/lint2 exit 0（唯一 warning 为 #15 既有 Q1）；
   routeTree 零 diff；定向 eslint 0；prettier 合规；e2e sign-in ✓ +
   customers ✘ 与 base 新鲜重跑**同签名**（locator/超时/element(s) not found
   三要素一致）= 环境性非回归。

## 裁定

**PASS**。偏差恰五类且全数 formatter 强制或 Ruling-PD1 授权；无规格缺口；
无 ⚠️ 不可验证项（浏览器真实走查按计划归 #29/attended 欠账，非遗漏）。
