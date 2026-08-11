package wechat

type amount struct {
	Total    int64  `json:"total"`
	Currency string `json:"currency"`
}

type nativeCreateRequest struct {
	AppID       string `json:"appid"`
	MchID       string `json:"mchid"`
	Description string `json:"description"`
	OutTradeNo  string `json:"out_trade_no"`
	NotifyURL   string `json:"notify_url"`
	Amount      amount `json:"amount"`
}

type nativeCreateResponse struct {
	CodeURL string `json:"code_url"`
}

type transaction struct {
	AppID         string `json:"appid"`
	MchID         string `json:"mchid"`
	OutTradeNo    string `json:"out_trade_no"`
	TransactionID string `json:"transaction_id"`
	TradeState    string `json:"trade_state"`
	SuccessTime   string `json:"success_time"`
	Amount        amount `json:"amount"`
}

type encryptedResource struct {
	Algorithm      string `json:"algorithm"`
	Ciphertext     string `json:"ciphertext"`
	Nonce          string `json:"nonce"`
	AssociatedData string `json:"associated_data"`
	OriginalType   string `json:"original_type"`
}

type notification struct {
	ID           string            `json:"id"`
	CreateTime   string            `json:"create_time"`
	EventType    string            `json:"event_type"`
	ResourceType string            `json:"resource_type"`
	Resource     encryptedResource `json:"resource"`
	Summary      string            `json:"summary"`
}

type refundAmount struct {
	Refund   int64  `json:"refund"`
	Total    int64  `json:"total"`
	Currency string `json:"currency"`
}

type refundRequest struct {
	OutTradeNo  string       `json:"out_trade_no"`
	OutRefundNo string       `json:"out_refund_no"`
	Reason      string       `json:"reason,omitempty"`
	NotifyURL   string       `json:"notify_url"`
	Amount      refundAmount `json:"amount"`
}

type refund struct {
	RefundID    string       `json:"refund_id"`
	OutRefundNo string       `json:"out_refund_no"`
	OutTradeNo  string       `json:"out_trade_no"`
	Status      string       `json:"status"`
	SuccessTime string       `json:"success_time"`
	Amount      refundAmount `json:"amount"`
}

type refundNotificationResource struct {
	MchID         string                   `json:"mchid"`
	OutTradeNo    string                   `json:"out_trade_no"`
	TransactionID string                   `json:"transaction_id"`
	OutRefundNo   string                   `json:"out_refund_no"`
	RefundID      string                   `json:"refund_id"`
	RefundStatus  string                   `json:"refund_status"`
	SuccessTime   string                   `json:"success_time"`
	Amount        refundNotificationAmount `json:"amount"`
}

type refundNotificationAmount struct {
	Refund      int64 `json:"refund"`
	Total       int64 `json:"total"`
	PayerRefund int64 `json:"payer_refund"`
	PayerTotal  int64 `json:"payer_total"`
}

type wechatErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
