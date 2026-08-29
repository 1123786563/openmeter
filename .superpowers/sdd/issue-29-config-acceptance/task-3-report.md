# Task 3 Report — 浏览器全链路走查四大块

- 状态：DONE_WITH_ENVIRONMENT_SHORTFALL（四块全链路写操作全部通过并截图；「真实后端」在本环境不可达，已按计划前置 Ruling 以 stateful-mock 全链路执行，Tier-2 真实探查尽力后不可达，如实记录）
- 产物（不入仓库，已归档本工作区 walkthrough/）：
  - walkthrough.spec.ts（走查脚本副本）
  - walkthrough-evidence.json（全部 wire 请求/响应记录 + 断言记录）
  - shots/01-config-features-created.png … 04-config-app-installed.png（4 张 1440x900 整窗截图）

## 四块结果（全部通过）

| 块 | 页面 | 写操作 | 断言（DOM，全过） | 截图 |
| --- | --- | --- | --- | --- |
| ① | /config/features | 新建功能 name=验收-功能 key=acceptance_feature | 侧栏「功能」链接可见；toast「功能已创建」；列表行 验收-功能 + acceptance_feature | 01（md5 069db0dccc，1826 色） |
| ② | /config/notification/channels | 新建渠道 name=验收-Webhook url=https://example.com/webhook | 侧栏「通知」；toast「通知渠道已创建。」；行 验收-Webhook 可见且**不含**「已禁用」 | 02（md5 57bad57bba，1396 色） |
| ③ | /config/tax-codes | 新建税码 name=验收税码 key=acceptance_tax_code | 侧栏「税码」；toast「税码已创建」；单元格 验收税码（组织默认卡两下拉同时解析到新税码——顺带证据） | 03（md5 8a48c22363，1345 色） |
| ④ | /config/apps | 目录安装 sandbox 应用 name=验收-Sandbox | 侧栏「应用」；toast「应用已安装」（精确匹配）；已装列表 验收-Sandbox | 04（md5 ced7e77a00，1147 色） |

## 全链路证据要点

- 每块均为真实 UI 回环：mock IdP 登录 → 侧栏断言 → 页面 → 新建对话框 → 表单填写 → 提交 → **真实 UI 请求体捕获**（evidence JSON）→ wire 精确响应（strictObject 校验通过=SDK validate 不抛）→ React Query invalidate → refetch（GET 回读 stateful store）→ 新行渲染 → toast → 整窗截图。
- 捕获的真实 UI 写请求体（来自页面自身，非脚本伪造）：
  - feature create：`{name:'验收-功能', key:'acceptance_feature'}`（v3）
  - channel create：`{type:'WEBHOOK', name:'验收-Webhook', url:'https://example.com/webhook'}`（v1 camelCase）
  - tax code create：`{name:'验收税码', key:'acceptance_tax_code'}`（v3）
  - app install：`{type:'sandbox', name:'验收-Sandbox', create_billing_profile:false}`（v3，billingInstallAppResponseWire 包裹响应 `{app, default_for_capability_types}` 正确消费）
- 像素证据（视觉通道损坏替代，同前序 28 轨模式）：4 张均 1440x900 非空白（1147-1826 unique colors），侧栏区（左 256px）非空白（376-436 色），4 md5 互异，两两像素差 3.6%-99.8% 全部有效区分。
- 未捕获任何未知 /api 端点（catch-all abort 监听零触发）——四块首屏读端点清单完整：namespaces + 各域 list + tax-codes 页的 org-defaults。
- **产品缺陷：零**。走查过程中的三次失败均为走查脚本自身问题（install 响应包裹形状、Playwright glob `*` 不跨 `/`、严格模式断言多匹配），修正脚本后全过；UI 在 wire 精确响应下行为全部正确。

## Tier-2 真实环境探查（不可达，如实记录）

- browser_* 工具：被无人值守自动化允许列表拒（browser_status 探测在案）。
- :5173 现役 dev server：本轮走查时已停止监听（curl 000 / exit 7，与轮首 lsof 记录对比为会话中途停止）。
- :8888：用户 38 行无状态 shim（GET 恒 `{"data":[]}`、POST 恒 `{}`，无持久化）——真实写+列表断言在此必然失败。
- 真实 OpenMeter 全栈（compose 依赖栈+原生 server+Casdoor 接线）：本仓库 compose 栈未运行、web/.env 缺失，拉起需用户侧认证接线决策（计划前置 Ruling 已载）。

## 剩余风险（走查维度）

「真实 OpenMeter 后端 + Casdoor」字面走查未执行——由用户在具备真实栈时补做，或接受 stateful-mock 全链路证据（外化批准时一并裁决）。截图上传 issue 评论属外发操作，等待批准。
