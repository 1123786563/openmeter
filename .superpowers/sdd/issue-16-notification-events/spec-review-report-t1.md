# T1 规格符合性审查（分遍 1）— 2026-08-29

审查域：3a3a2f2c6..eab5a1d67（legacy.ts / query-keys.ts / hooks.ts / zh-CN.ts / en.ts）。
方法：diff 派生新增行与处方代码块做空白归一化比对（排除纯 prettier 换行差异）+ 锚点/键集/门禁新鲜复核。

## 逐项结论

1. legacy.ts events 段 vs 处方步骤 1：**归一化 IDENTICAL**（含分隔注释、
   URLSearchParams 重复数组手法、resend 202 → void 语义）。
2. query-keys `notificationEvents` vs 处方步骤 2：内容 IDENTICAL；行因 >80 列
   被 prettier 折行（Ruling-P1 授权的格式性偏差，与相邻 notificationChannels
   同式）。
3. hooks.ts events 段 vs 处方步骤 3：主体（exportinterface 起）IDENTICAL；
   `resendNotificationEvent(eventId, channels?.length ? { channels } : {})`
   一行化为 prettier 折叠（内容不变）；import 三项按字母序插入（处方未定
   顺序，合规）。
4. zh-CN/en events 子树 vs 处方步骤 6：内容 IDENTICAL（zh/en 各 31 键；
   title 值与占位一致为未变上下文；description 按处方新文案替换 =
   Ruling-i18n 授权；3 条长文案行 prettier 折行）。
5. 既有键零删改：全 diff 删除行恰 2 条（zh/en 占位 description），events
   子树外无任何删改。git diff --stat：202+/2-。
6. 锚点：legacy 段位于 testNotificationRule 后（currencies 前）✓；query-key
   紧随 notificationRules ✓；hooks 段紧随 useTestRule（Currencies 分节前）✓；
   i18n 子树位于原占位处（notification 内 rules 后）✓。
7. Ruling 落实：Ruling-prefix——queryKey `ns('notification.events')` 与
   useResendEvent/useTestRule 失效前缀同域 ✓；Ruling-i18n ✓；单选升数组
   （rule→[rule]、channel→[channel]）✓；省略空 channels=原渠道 ✓。
8. 门禁新鲜复核：build3 exit 0 / lint3 exit 0（唯一 warning 为 #15 既有
   Ruling-Q1，非本 diff）/ routeTree 零 diff / locale 结构 diff 与 base 完全
   同构（仅 3 条 base 既有伪差，零新增）/ 新增区 prettier 合规（与
   prettier --write 产物逐字节一致）。

## 裁定

**PASS**。偏差恰两类且均已获 Ruling 授权：① prettier 折行（3 条 i18n 长行 +
qk 行 + hooks 一行化，内容零变化）；② import 插入位（处方未规定）。
未发现规格缺口；无 ⚠️ 不可验证项。
