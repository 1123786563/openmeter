# Task 5 Report

## Fix round 1

- 在支付宝退款异步回调验签后，绑定 `sign_type`、`app_id` 与 `seller_id` 至已配置身份；要求 `out_trade_no`、`out_request_no` 非空，`refund_fee` 为正数，并拒绝显式非 CNY 的 `refund_currency`。
- 仅接受 `REFUND_SUCCESS`、`REFUND_PROCESSING` 与 `REFUND_FAIL`：仅成功状态映射为 `Success=true`；处理中不产生 `RawHash`，终态成功或失败保留回调哈希。
- 添加了上述签名上下文、金额、币种、状态及哈希语义的回归测试。
- 验证：`go test ./openmeter/commerce/payment/alipay ./openmeter/commerce/refund -count=1`；`git diff --check`。
