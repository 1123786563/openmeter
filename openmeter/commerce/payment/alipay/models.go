package alipay

type precreateRequest struct {
	OutTradeNo string `json:"out_trade_no"`
	Amount     string `json:"total_amount"`
	Subject    string `json:"subject"`
}

type precreateResponse struct {
	providerResponse
	OutTradeNo string `json:"out_trade_no"`
	QRCode     string `json:"qr_code"`
}

type tradeQueryRequest struct {
	OutTradeNo string `json:"out_trade_no"`
}

type tradeQueryResponse struct {
	providerResponse
	OutTradeNo string `json:"out_trade_no"`
	TradeNo    string `json:"trade_no"`
	TradeState string `json:"trade_status"`
	Amount     string `json:"total_amount"`
}

type refundRequest struct {
	OutTradeNo   string `json:"out_trade_no"`
	Amount       string `json:"refund_amount"`
	OutRequestNo string `json:"out_request_no"`
	Reason       string `json:"refund_reason,omitempty"`
}

type refundResponse struct {
	providerResponse
	TradeNo    string `json:"trade_no"`
	OutTradeNo string `json:"out_trade_no"`
	RefundFee  string `json:"refund_fee"`
	FundChange string `json:"fund_change"`
}

type refundQueryRequest struct {
	OutTradeNo   string `json:"out_trade_no"`
	OutRequestNo string `json:"out_request_no"`
}

type refundQueryResponse struct {
	providerResponse
	TradeNo      string `json:"trade_no"`
	OutTradeNo   string `json:"out_trade_no"`
	OutRequestNo string `json:"out_request_no"`
	RefundAmount string `json:"refund_amount"`
	RefundStatus string `json:"refund_status"`
}

type providerResponse struct {
	Code    string `json:"code"`
	Message string `json:"msg"`
	SubCode string `json:"sub_code"`
	SubMsg  string `json:"sub_msg"`
}
