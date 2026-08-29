# Browser Walkthrough Report — Issue #6 计划创建向导（修复轮 1 后，HEAD f6e767dc3）

- 执行者：takeover #4 控制器（隔离降级同前：本会话允许列表拒绝 subagent 派生；探针脚本 /tmp/issue6-walkthrough.cjs，Playwright chromium 无头，真实构建产物 preview :4173 + mock IdP :9999 + page.route wire 级 mock）
- 日期：2026-08-28 21:2x+08:00
- 运行记录：/tmp/issue6-t4-walkthrough.log（WALKTHROUGH: ALL PASS，exit 0）；结果 JSON /tmp/issue6-walkthrough-results.json；截图 /tmp/issue6-shots/ 6 张

## 结果总览：17/17 PASS，pageerror 0

| # | 场景 | 结果 |
|---|------|------|
| a | 列表入口「新建计划」打开向导（基本信息 1/3） | PASS |
| b1 | I1：描述 Textarea maxlength=1024 属性存在 | PASS |
| b2 | I1：原生 setter 强注 1030 字符 → 步骤 1 被门禁拦截，描述字段可见错误（zod 默认英文 "Invalid input"，与处方 max(1024) 无自定义消息一致） | PASS |
| b3 | I1 恢复：合法描述 → 进入步骤 2 | PASS |
| c1 | **C1 修复**：两阶段（P1M + 末段留空）→「下一步」→ 价目卡编辑 (3/3) 可达（修复前该转换恒死锁） | PASS |
| c2 | 步骤 3 挂载两张卡金额输入（inputmode=decimal ×2） | PASS |
| d | 叶子门禁回归：阶段 1 期限清空 → 步骤 2 拦截 + 行级错误「仅最后一个阶段可以无期限」 | PASS |
| d2 | 恢复后重新到达步骤 3 | PASS |
| e1 | 创建计划恰好触发 1 次 POST /api/v3/openmeter/plans | PASS |
| e2 | **wire 体深比较相等**（稳定键序）：name/key/description trim、末段空 duration 省略、卡 billing_cadence 默认 P1M、phase1 flat amount '99.9' 字符串、phase2 free {type:'free'}、snake_case（v3 SDK 序列化） | PASS |
| e3 | 成功 toast「计划已创建（草稿）」 | PASS |
| e4 | 对话框关闭 | PASS |
| e5 | invalidation → 列表 GET 重取（1→2） | PASS |
| f | en locale（localStorage openmeter-admin.language=en）：向导 Basics (1/3) | PASS |
| g | 全程 pageerror = 0 | PASS |

## 截图像素完备性（Pillow 11.x）

- 6 张全部 1280×720（Playwright 默认视口）、unique colors 869–1714（非空白）
- 15 组成对差异全部可区分（0.0055–0.677）；01↔05 同为列表页（05 仅多 toast）0.0055 符合预期
- 视觉通道（read_image/describe_image）与 #1–#5 各会话同况不可用，程序化证据（DOM 断言 + wire 记录 + 像素指标）代替，建议用户目验 /tmp/issue6-shots/

## 过程记录（真实性）

- 探针迭代 4 轮定位修正（价目卡名称 label→placeholder 定位；价格类型 combobox DOM 序 type→billingCadence→priceKind，第二卡价格类型 = nth(5)；en 登录 waitForURL 需等完整回到应用根而非 IdP 中转 URL）——前 3 轮的 FAIL 均为探针定位问题，非产品缺陷；最终运行全绿且 e2 wire 断言把首跑的“FAIL”纠正为断言侧错误（camelCase→snake_case、卡默认 billing_cadence P1M 来自 defaultRateCard() 处方默认）。

Ruling: WALKTHROUGH PASS（17/17，含验收标准「向导可完整走通并创建含固定费价卡的计划」的 wire 级证明）
