# OpenMeter 管理端（Admin Console）

面向内部运营的 Web 控制台，用于管理 OpenMeter 的计费数据（客户、订阅、发票、用量、额度）。本文件是该上下文的术语表；OpenMeter 服务端域语义以 `openmeter/` 各包文档为准。

## Language

**管理端（Admin Console）**:
面向运营用户的内部 Web 控制台，用于查询与操作 OpenMeter 的计费数据。
_Avoid_: 后台管理系统、Dashboard、Portal

**运营用户（Staff）**:
通过公司身份系统（Casdoor）登录管理端的内部员工，是管理端的使用者。
_Avoid_: 管理员、用户（与计费客户混淆）

**只读运营（Viewer）**:
由 Casdoor 角色映射而来、仅能查看管理端数据的运营用户角色。
_Avoid_: 访客、观察者

**可写运营（Operator）**:
由 Casdoor 角色映射而来、可执行管理端全部写操作的运营用户角色。
_Avoid_: 超级管理员

**计费客户（Customer）**:
OpenMeter 中被计量与计费的主体（人或组织），是管理端管理的对象，不是管理端的使用者。
_Avoid_: Client、Account、用户

**客户门户（Consumer Portal）**:
OpenMeter 面向计费客户自助查看数据的既有机制（portal token 保护），与管理端受众不同。
_Avoid_: 与管理端混称

**命名空间（Namespace）**:
OpenMeter 的资源隔离单位。服务端配置 default 与可选 allowlist；管理端通过全局切换器以 X-Namespace 头选择，当前命名空间过滤所有页面。
_Avoid_: 租户（Tenant）、环境

**额度发放（Credit Grant）**:
向计费客户授予预付额度的一次性操作及其记录。
_Avoid_: 充值（Recharge，指钱包支付行为）、优惠券

**权益（Entitlement）**:
计费客户对某项功能（feature）的可用额度或使用权视图，由订阅或额度发放产生。
_Avoid_: 权限（Permission）

**仪表（Meter）**:
定义一类用量的计量口径与聚合方式；事件按主体归属、按仪表聚合。
_Avoid_: 指标（Metric）、度量

**主体（Subject）**:
事件计量的归属键，实践中通常即计费客户的标识。
_Avoid_: 账号

**钱包（Wallet）**:
计费客户的预付余额载体及其充值入账记录。
_Avoid_: 与额度发放（Credit Grant）混用

**充值产品（Recharge Product）**:
定义可购买充值档位（金额、赠送、货币）的商品配置。
_Avoid_: 套餐（Plan）

**订单（Order）**:
计费客户购买充值产品等行为的交易记录，含支付状态。
_Avoid_: 账单（Invoice）

**退款（Refund）**:
对已支付订单的资金退回记录。
_Avoid_: 撤销

**线下支付（Offline Payment）**:
人工登记的非线上渠道支付入账（如对公转账）。
_Avoid_: 人工订单

**计划（Plan）**:
可被订阅的计费套餐定义，由阶段与价目卡构成，须发布后方可被订阅。
_Avoid_: 套餐（Recharge Product 语境）、商品

**阶段（Phase）**:
计划生效时间线上的一个区段，持有该区段适用的价目卡集合。
_Avoid_: 周期（Billing Cadence）

**价目卡（Rate Card）**:
计划内一条计价定义：固定费、用量单价或权益额度。
_Avoid_: 价格表

**功能（Feature）**:
产品能力目录项，可关联计价并被权益授予客户。
_Avoid_: 特性、开关

**附加组件（Addon）**:
计划的可选增购项，独立于主阶段计价。
_Avoid_: 插件

**通知渠道（Notification Channel）**:
通知的投递目的地（如 Slack、Webhook、邮件）。
_Avoid_: 群、机器人

**通知规则（Notification Rule）**:
决定何种事件经何渠道投递的配置。
_Avoid_: 告警策略

**自定义货币（Custom Currency）**:
ISO 货币之外、由运营定义的内部计价货币（如 CREDIT 额度货币）。
_Avoid_: 虚拟币、积分

**税码（Tax Code）**:
发票税务处理的分类标识。
_Avoid_: 税率

**账单档案（Billing Profile）**:
计费客户的开票信息集合（抬头、地址、税号等）。
_Avoid_: 客户资料

**门户令牌（Portal Token）**:
供计费客户访问客户门户的限时凭据。
_Avoid_: API Key

**应收周期（Receivable Period）**:
客户维度的应收账款账期区间。
_Avoid_: 账单周期

**应用（App）**:
与计费域集成的外部系统连接（如 Stripe 收单、自定义开票、沙箱）。
_Avoid_: 插件、集成（口语可用，命名不用）
