package provider

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

func stableRefundRequestID(req payment.RefundRequest) string {
	if value := strings.TrimSpace(req.RefundRequestID); value != "" {
		return value
	}
	if orderID := strings.TrimSpace(req.OrderID); orderID != "" {
		return orderID + "-refund"
	}
	if tradeNo := strings.TrimSpace(req.TradeNo); tradeNo != "" {
		return tradeNo + "-refund"
	}
	return ""
}
