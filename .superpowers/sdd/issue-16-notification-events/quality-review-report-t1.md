# T1 代码质量审查（分遍 2）— 2026-08-29

审查域：3a3a2f2c6..eab5a1d67。新鲜命令复核。

## 逐项结论

1. 定向 eslint（legacy/query-keys/hooks 三文件）：exit 0，0 error 0 warning
   （全仓 lint 的 1 条 warning 为 #15 rule-form-dialog 既有 Ruling-Q1）。
2. i18n 两文件定向 eslint：含在全仓 lint exit 0 内；i18n 文件不在 eslint
   失败清单。
3. 反模式扫描（新增行 grep）：console/debugger/: any/as any/@ts-ignore/
   TODO/FIXME 全零命中。
4. 死代码：新增 9 个导出全部为处方 Produces 契约（useNotificationEvents/
   useResendEvent/getNotificationEvent 由 T2 页面消费，其余为本层互消费）；
   无多余导出、无未用导入（import 三项均被使用——build 零 TS 错误佐证）。
5. diff 卫生：202+/2-，删除仅 2 条 Ruling-i18n 授权的占位行；无既有块重排
   （Ruling-P1：撤销了 prettier --write 对既有块的 8 处重排，仅保留新增内容
   格式化——本轨不扩大外部化合并面）。
6. 命名/风格：query key `notification.events` 域命名与相邻 channels/rules
   一致；分节注释风格与文件既有分节一致；satisfies 收窄、无类型逃逸。
7. 类型安全：无隐式 any（build 通过）；resend 202 无内容 → Promise<void>
   与既有 invalidatePortalTokens 同型先例。

## 裁定

**PASS**，无发现。Ruling-P1（prettier 范围）执行过程记录：prettier --write
曾重排 8 处既有块（hooks 4 处调用折叠 + zh 5 行 + en 3行长行重折），已全部
回退并改为仅新增内容合规；代价：三文件整体 --check 仍失败（base 既有债，
非本轨引入，外部化后可另行统一）。
