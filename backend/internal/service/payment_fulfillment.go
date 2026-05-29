package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

	"entgo.io/ent/dialect"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentauditlog"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// ErrOrderNotFound is returned by HandlePaymentNotification when the webhook
// references an out_trade_no that does not exist in our DB. Callers (webhook
// handlers) should treat this as a terminal, non-retryable condition and still
// respond with a 2xx success to the provider — otherwise the provider will keep
// retrying forever (e.g. when a foreign environment's webhook endpoint is
// misconfigured to point at us, or when our orders table has been wiped).
var ErrOrderNotFound = errors.New("payment order not found")

// --- Payment Notification & Fulfillment ---

func (s *PaymentService) HandlePaymentNotification(ctx context.Context, n *payment.PaymentNotification, pk string) error {
	if n.Status != payment.NotificationStatusSuccess {
		return nil
	}
	// Look up order by out_trade_no (the external order ID we sent to the provider)
	order, err := s.entClient.PaymentOrder.Query().Where(paymentorder.OutTradeNo(n.OrderID)).Only(ctx)
	if err != nil {
		// Fallback only for true legacy "sub2_N" DB-ID payloads when the
		// current out_trade_no lookup genuinely did not find an order.
		if oid, ok := parseLegacyPaymentOrderID(n.OrderID, err); ok {
			return s.confirmPayment(ctx, oid, n.TradeNo, n.Amount, pk, n.Metadata)
		}
		if dbent.IsNotFound(err) {
			return fmt.Errorf("%w: out_trade_no=%s", ErrOrderNotFound, n.OrderID)
		}
		return fmt.Errorf("lookup order failed for out_trade_no %s: %w", n.OrderID, err)
	}
	return s.confirmPayment(ctx, order.ID, n.TradeNo, n.Amount, pk, n.Metadata)
}

func parseLegacyPaymentOrderID(orderID string, lookupErr error) (int64, bool) {
	if !dbent.IsNotFound(lookupErr) {
		return 0, false
	}
	orderID = strings.TrimSpace(orderID)
	if !strings.HasPrefix(orderID, orderIDPrefix) {
		return 0, false
	}
	trimmed := strings.TrimPrefix(orderID, orderIDPrefix)
	if trimmed == "" || trimmed == orderID {
		return 0, false
	}
	oid, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil || oid <= 0 {
		return 0, false
	}
	return oid, true
}

func (s *PaymentService) confirmPayment(ctx context.Context, oid int64, tradeNo string, paid float64, pk string, metadata map[string]string) error {
	o, err := s.entClient.PaymentOrder.Get(ctx, oid)
	if err != nil {
		if dbent.IsNotFound(err) {
			return fmt.Errorf("%w: order_id=%d", ErrOrderNotFound, oid)
		}
		return fmt.Errorf("get order by id %d failed: %w", oid, err)
	}
	instanceProviderKey := ""
	if inst, instErr := s.getOrderProviderInstance(ctx, o); instErr == nil && inst != nil {
		instanceProviderKey = inst.ProviderKey
	}
	expectedProviderKey := expectedNotificationProviderKeyForOrder(s.registry, o, instanceProviderKey)
	if expectedProviderKey != "" && strings.TrimSpace(pk) != "" && !strings.EqualFold(expectedProviderKey, strings.TrimSpace(pk)) {
		s.writeAuditLog(ctx, o.ID, "PAYMENT_PROVIDER_MISMATCH", pk, map[string]any{
			"expectedProvider": expectedProviderKey,
			"actualProvider":   pk,
			"tradeNo":          tradeNo,
		})
		return fmt.Errorf("provider mismatch: expected %s, got %s", expectedProviderKey, pk)
	}
	if err := validateProviderNotificationMetadata(o, pk, metadata); err != nil {
		s.writeAuditLog(ctx, o.ID, "PAYMENT_PROVIDER_METADATA_MISMATCH", pk, map[string]any{
			"detail":  err.Error(),
			"tradeNo": tradeNo,
		})
		return err
	}
	if !isValidProviderAmount(paid) {
		s.writeAuditLog(ctx, o.ID, "PAYMENT_INVALID_AMOUNT", pk, map[string]any{
			"expected": o.PayAmount,
			"paid":     paid,
			"tradeNo":  tradeNo,
		})
		return fmt.Errorf("invalid paid amount from provider: %v", paid)
	}
	if math.Abs(paid-o.PayAmount) > amountToleranceCNY {
		s.writeAuditLog(ctx, o.ID, "PAYMENT_AMOUNT_MISMATCH", pk, map[string]any{"expected": o.PayAmount, "paid": paid, "tradeNo": tradeNo})
		return fmt.Errorf("amount mismatch: expected %.2f, got %.2f", o.PayAmount, paid)
	}
	return s.toPaid(ctx, o, tradeNo, paid, pk)
}

func isValidProviderAmount(amount float64) bool {
	return amount > 0 && !math.IsNaN(amount) && !math.IsInf(amount, 0)
}

func validateProviderNotificationMetadata(order *dbent.PaymentOrder, providerKey string, metadata map[string]string) error {
	return validateProviderSnapshotMetadata(order, providerKey, metadata)
}

func expectedNotificationProviderKey(registry *payment.Registry, orderPaymentType string, orderProviderKey string, instanceProviderKey string) string {
	if key := strings.TrimSpace(instanceProviderKey); key != "" {
		return key
	}
	if key := strings.TrimSpace(orderProviderKey); key != "" {
		return key
	}
	if registry != nil {
		if key := strings.TrimSpace(registry.GetProviderKey(payment.PaymentType(orderPaymentType))); key != "" {
			return key
		}
	}
	return strings.TrimSpace(orderPaymentType)
}

func (s *PaymentService) toPaid(ctx context.Context, o *dbent.PaymentOrder, tradeNo string, paid float64, pk string) error {
	now := time.Now()
	grace := now.Add(-paymentGraceMinutes * time.Minute)
	dailyLimit, limitErr := s.currentPaymentDailyLimit(ctx)
	if limitErr != nil {
		slog.Error("load payment config failed during payment confirmation", "orderID", o.ID, "error", limitErr)
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)

	lockRelease, err := lockAuthPendingIdentityKeys(txCtx, tx.Client(), fmt.Sprintf("payment-daily-limit-user:%d", o.UserID))
	if err != nil {
		return fmt.Errorf("acquire payment user lock: %w", err)
	}
	defer lockRelease()

	orderQuery := tx.PaymentOrder.Query().Where(paymentorder.IDEQ(o.ID))
	if tx.Client().Driver().Dialect() == dialect.Postgres {
		orderQuery = orderQuery.ForUpdate()
	}
	lockedOrder, err := orderQuery.Only(txCtx)
	if err != nil {
		return fmt.Errorf("reload order in transaction: %w", err)
	}
	previousStatus := lockedOrder.Status

	if !paymentOrderCanTransitionToPaid(lockedOrder.Status, lockedOrder.UpdatedAt, grace) {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit no-op transaction: %w", err)
		}
		return s.alreadyProcessed(ctx, lockedOrder)
	}

	if dailyLimit > 0 {
		limitAmount := paymentOrderLimitAmount(lockedOrder.OrderType, lockedOrder.Amount, paid)
		if err := s.checkDailyLimit(txCtx, tx, lockedOrder.UserID, limitAmount, dailyLimit, false); err != nil {
			s.writeAuditLog(ctx, lockedOrder.ID, "PAYMENT_DAILY_LIMIT_BLOCKED", pk, map[string]any{
				"dailyLimit": dailyLimit,
				"status":     lockedOrder.Status,
				"tradeNo":    tradeNo,
				"paidAmount": paid,
			})
			return err
		}
	}

	if _, err := tx.PaymentOrder.UpdateOneID(lockedOrder.ID).
		SetStatus(OrderStatusPaid).
		SetPayAmount(paid).
		SetPaymentTradeNo(tradeNo).
		SetPaidAt(now).
		ClearFailedAt().
		ClearFailedReason().
		Save(txCtx); err != nil {
		return fmt.Errorf("update to PAID: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit paid transition: %w", err)
	}

	if previousStatus == OrderStatusCancelled || previousStatus == OrderStatusExpired {
		slog.Info("order recovered from webhook payment success",
			"orderID", lockedOrder.ID,
			"previousStatus", previousStatus,
			"tradeNo", tradeNo,
			"provider", pk,
		)
		s.writeAuditLog(ctx, lockedOrder.ID, "ORDER_RECOVERED", pk, map[string]any{
			"previous_status": previousStatus,
			"tradeNo":         tradeNo,
			"paidAmount":      paid,
			"reason":          "webhook payment success received after order " + previousStatus,
		})
	}
	s.writeAuditLog(ctx, lockedOrder.ID, "ORDER_PAID", pk, map[string]any{"tradeNo": tradeNo, "paidAmount": paid})
	return s.executeFulfillment(ctx, lockedOrder.ID)
}

func (s *PaymentService) currentPaymentDailyLimit(ctx context.Context) (float64, error) {
	if s == nil || s.configService == nil {
		return 0, nil
	}
	cfg, err := s.configService.GetPaymentConfig(ctx)
	if err != nil {
		return 0, err
	}
	return cfg.DailyLimit, nil
}

func paymentOrderCanTransitionToPaid[S ~string](status S, updatedAt time.Time, graceBoundary time.Time) bool {
	switch status {
	case OrderStatusPending, OrderStatusCancelled:
		return true
	case OrderStatusExpired:
		return !updatedAt.Before(graceBoundary)
	default:
		return false
	}
}

func (s *PaymentService) alreadyProcessed(ctx context.Context, o *dbent.PaymentOrder) error {
	cur, err := s.entClient.PaymentOrder.Get(ctx, o.ID)
	if err != nil {
		if dbent.IsNotFound(err) {
			return fmt.Errorf("%w: order_id=%d", ErrOrderNotFound, o.ID)
		}
		return fmt.Errorf("reload order %d failed: %w", o.ID, err)
	}
	switch cur.Status {
	case OrderStatusCompleted, OrderStatusRefunded:
		return nil
	case OrderStatusFailed:
		return s.executeFulfillment(ctx, o.ID)
	case OrderStatusPaid, OrderStatusRecharging:
		return fmt.Errorf("order %d is being processed", o.ID)
	case OrderStatusExpired:
		slog.Warn("webhook payment success for expired order beyond grace period",
			"orderID", o.ID,
			"status", cur.Status,
			"updatedAt", cur.UpdatedAt,
		)
		s.writeAuditLog(ctx, o.ID, "PAYMENT_AFTER_EXPIRY", "system", map[string]any{
			"status":    cur.Status,
			"updatedAt": cur.UpdatedAt,
			"reason":    "payment arrived after expiry grace period",
		})
		return nil
	default:
		return nil
	}
}

func (s *PaymentService) executeFulfillment(ctx context.Context, oid int64) error {
	o, err := s.entClient.PaymentOrder.Get(ctx, oid)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}
	if o.OrderType == payment.OrderTypeSubscription {
		return s.ExecuteSubscriptionFulfillment(ctx, oid)
	}
	return s.ExecuteBalanceFulfillment(ctx, oid)
}

func (s *PaymentService) ExecuteBalanceFulfillment(ctx context.Context, oid int64) error {
	o, err := s.entClient.PaymentOrder.Get(ctx, oid)
	if err != nil {
		return infraerrors.NotFound("NOT_FOUND", "order not found")
	}
	if o.Status == OrderStatusCompleted {
		return s.recordDistributionPaidCreditForOrder(ctx, o)
	}
	if psIsRefundStatus(o.Status) {
		return infraerrors.BadRequest("INVALID_STATUS", "refund-related order cannot fulfill")
	}
	if o.Status != OrderStatusPaid && o.Status != OrderStatusFailed {
		return infraerrors.BadRequest("INVALID_STATUS", "order cannot fulfill in status "+string(o.Status))
	}
	c, err := s.entClient.PaymentOrder.Update().Where(paymentorder.IDEQ(oid), paymentorder.StatusIn(OrderStatusPaid, OrderStatusFailed)).SetStatus(OrderStatusRecharging).Save(ctx)
	if err != nil {
		return fmt.Errorf("lock: %w", err)
	}
	if c == 0 {
		return nil
	}
	if err := s.doBalance(ctx, o); err != nil {
		s.markFailed(ctx, oid, err)
		return err
	}
	return nil
}

// redeemAction represents the idempotency decision for balance fulfillment.
type redeemAction int

const (
	// redeemActionCreate: code does not exist — create it, then redeem.
	redeemActionCreate redeemAction = iota
	// redeemActionRedeem: code exists but is unused — skip creation, redeem only.
	redeemActionRedeem
	// redeemActionSkipCompleted: code exists and is already used — skip to mark completed.
	redeemActionSkipCompleted
)

// resolveRedeemAction decides the idempotency action based on an existing redeem code lookup.
// existing is the result of GetByCode; lookupErr is the error from that call.
func resolveRedeemAction(existing *RedeemCode, lookupErr error) redeemAction {
	if existing == nil || lookupErr != nil {
		return redeemActionCreate
	}
	if existing.IsUsed() {
		return redeemActionSkipCompleted
	}
	return redeemActionRedeem
}

func (s *PaymentService) doBalance(ctx context.Context, o *dbent.PaymentOrder) error {
	// Idempotency: check if redeem code already exists (from a previous partial run)
	existing, lookupErr := s.redeemService.GetByCode(ctx, o.RechargeCode)
	action := resolveRedeemAction(existing, lookupErr)

	switch action {
	case redeemActionSkipCompleted:
		if err := s.recordDistributionPaidCreditForOrder(ctx, o); err != nil {
			return err
		}
		if err := s.applyAffiliateRebateForOrder(ctx, o); err != nil {
			return err
		}
		// Code already created and redeemed — just mark completed
		return s.markCompleted(ctx, o, "RECHARGE_SUCCESS")
	case redeemActionCreate:
		rc := &RedeemCode{Code: o.RechargeCode, Type: RedeemTypeBalance, Value: o.Amount, Status: StatusUnused}
		if err := s.redeemService.CreateCode(ctx, rc); err != nil {
			return fmt.Errorf("create redeem code: %w", err)
		}
	case redeemActionRedeem:
		// Code exists but unused — skip creation, proceed to redeem
	}
	if _, err := s.redeemService.Redeem(ContextSkipRedeemAffiliate(ctx), o.UserID, o.RechargeCode); err != nil {
		return fmt.Errorf("redeem balance: %w", err)
	}
	if err := s.recordDistributionPaidCreditForOrder(ctx, o); err != nil {
		return err
	}
	if err := s.applyAffiliateRebateForOrder(ctx, o); err != nil {
		return err
	}
	return s.markCompleted(ctx, o, "RECHARGE_SUCCESS")
}

func (s *PaymentService) recordDistributionPaidCreditForOrder(ctx context.Context, o *dbent.PaymentOrder) error {
	if s == nil || o == nil {
		return nil
	}
	amount, ok := distributionPaidCreditAmountForOrder(o)
	if !ok {
		return nil
	}
	orderType := string(o.OrderType)
	if o.UserID <= 0 || o.ID <= 0 {
		s.writeAuditLog(ctx, o.ID, "AFFILIATE_DISTRIBUTION_PAID_CREDIT_FAILED", "system", map[string]any{
			"error":      "invalid order identity for affiliate distribution paid credit",
			"amount":     amount,
			"order_type": orderType,
		})
		slog.Warn("[PaymentService] affiliate distribution paid credit skipped due to invalid order identity", "orderID", o.ID, "userID", o.UserID)
		return nil
	}
	if s.affiliateService == nil {
		s.writeAuditLog(ctx, o.ID, "AFFILIATE_DISTRIBUTION_PAID_CREDIT_FAILED", "system", map[string]any{
			"error":      "affiliate distribution service unavailable",
			"amount":     amount,
			"order_type": orderType,
		})
		slog.Warn("[PaymentService] affiliate distribution paid credit service unavailable", "orderID", o.ID, "userID", o.UserID)
		return nil
	}
	creditedAt := time.Now().UTC()
	if o.PaidAt != nil && !o.PaidAt.IsZero() {
		creditedAt = o.PaidAt.UTC()
	}
	applied, err := s.affiliateService.RecordDistributionPaidCredit(ctx, o.UserID, o.ID, amount, creditedAt)
	if err != nil {
		s.writeAuditLog(ctx, o.ID, "AFFILIATE_DISTRIBUTION_PAID_CREDIT_FAILED", "system", map[string]any{
			"error":      err.Error(),
			"amount":     amount,
			"creditedAt": creditedAt.Format(time.RFC3339Nano),
			"order_type": orderType,
		})
		slog.Warn("[PaymentService] affiliate distribution paid credit failed", "orderID", o.ID, "userID", o.UserID, "error", err)
		return nil
	}
	if applied && !s.hasAuditLog(ctx, o.ID, "AFFILIATE_DISTRIBUTION_PAID_CREDIT_APPLIED") {
		s.writeAuditLog(ctx, o.ID, "AFFILIATE_DISTRIBUTION_PAID_CREDIT_APPLIED", "system", map[string]any{
			"amount":     amount,
			"creditedAt": creditedAt.Format(time.RFC3339Nano),
			"order_type": orderType,
		})
	}
	return nil
}

func (s *PaymentService) markCompleted(ctx context.Context, o *dbent.PaymentOrder, auditAction string) error {
	return s.markCompletedWithClient(ctx, s.entClient, o, auditAction)
}

func (s *PaymentService) ExecuteSubscriptionFulfillment(ctx context.Context, oid int64) error {
	o, err := s.entClient.PaymentOrder.Get(ctx, oid)
	if err != nil {
		return infraerrors.NotFound("NOT_FOUND", "order not found")
	}
	if o.Status == OrderStatusCompleted {
		return s.recordDistributionPaidCreditForOrder(ctx, o)
	}
	if psIsRefundStatus(o.Status) {
		return infraerrors.BadRequest("INVALID_STATUS", "refund-related order cannot fulfill")
	}
	if o.Status != OrderStatusPaid && o.Status != OrderStatusFailed {
		return infraerrors.BadRequest("INVALID_STATUS", "order cannot fulfill in status "+string(o.Status))
	}
	if o.SubscriptionGroupID == nil || o.SubscriptionDays == nil {
		return infraerrors.BadRequest("INVALID_STATUS", "missing subscription info")
	}
	c, err := s.entClient.PaymentOrder.Update().Where(paymentorder.IDEQ(oid), paymentorder.StatusIn(OrderStatusPaid, OrderStatusFailed)).SetStatus(OrderStatusRecharging).Save(ctx)
	if err != nil {
		return fmt.Errorf("lock: %w", err)
	}
	if c == 0 {
		return nil
	}
	if err := s.doSub(ctx, o); err != nil {
		s.markFailed(ctx, oid, err)
		return err
	}
	return nil
}

func (s *PaymentService) doSub(ctx context.Context, o *dbent.PaymentOrder) error {
	gid := *o.SubscriptionGroupID
	days := *o.SubscriptionDays
	g, err := s.groupRepo.GetByID(ctx, gid)
	if err != nil || g.Status != payment.EntityStatusActive {
		return fmt.Errorf("group %d no longer exists or inactive", gid)
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin subscription fulfillment tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)

	claimAction := "SUBSCRIPTION_FULFILLMENT_CLAIM"
	claimed, err := s.claimPaymentAuditAction(txCtx, tx.Client(), o.ID, claimAction, map[string]any{
		"group_id":   gid,
		"days":       days,
		"order_type": string(o.OrderType),
	})
	if err != nil {
		return fmt.Errorf("claim subscription fulfillment: %w", err)
	}
	if !claimed {
		slog.Info("subscription fulfillment already claimed, skip extension", "orderID", o.ID, "groupID", gid)
		if err := s.markCompletedWithClient(txCtx, tx.Client(), o, "SUBSCRIPTION_SUCCESS"); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit idempotent subscription completion: %w", err)
		}
		return s.recordDistributionPaidCreditForOrder(ctx, o)
	}

	orderNote := fmt.Sprintf("payment order %d", o.ID)
	_, _, err = s.subscriptionSvc.AssignOrExtendSubscription(txCtx, &AssignSubscriptionInput{UserID: o.UserID, GroupID: gid, ValidityDays: days, AssignedBy: 0, Notes: orderNote})
	if err != nil {
		return fmt.Errorf("assign subscription: %w", err)
	}
	if err := s.markCompletedWithClient(txCtx, tx.Client(), o, "SUBSCRIPTION_SUCCESS"); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit subscription fulfillment: %w", err)
	}
	return s.recordDistributionPaidCreditForOrder(ctx, o)
}

func (s *PaymentService) hasAuditLog(ctx context.Context, orderID int64, action string) bool {
	oid := strconv.FormatInt(orderID, 10)
	c, _ := s.entClient.PaymentAuditLog.Query().
		Where(paymentauditlog.OrderIDEQ(oid), paymentauditlog.ActionEQ(action)).
		Limit(1).Count(ctx)
	return c > 0
}

func (s *PaymentService) applyAffiliateRebateForOrder(ctx context.Context, o *dbent.PaymentOrder) error {
	if o == nil || o.OrderType != payment.OrderTypeBalance || o.Amount <= 0 {
		return nil
	}
	if s.affiliateService == nil {
		return nil
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		s.writeAuditLog(ctx, o.ID, "AFFILIATE_REBATE_FAILED", "system", map[string]any{
			"error": fmt.Sprintf("begin affiliate rebate tx: %v", err),
		})
		return fmt.Errorf("begin affiliate rebate tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	claimed, err := s.tryClaimAffiliateRebateAudit(txCtx, tx.Client(), o.ID, o.Amount)
	if err != nil {
		s.writeAuditLog(ctx, o.ID, "AFFILIATE_REBATE_FAILED", "system", map[string]any{
			"error": err.Error(),
		})
		return fmt.Errorf("claim affiliate rebate audit: %w", err)
	}
	if !claimed {
		return nil
	}

	sourceOrderID := o.ID
	rebateAmount, err := s.affiliateService.AccrueInviteRebateForOrder(txCtx, o.UserID, o.Amount, &sourceOrderID)
	if err != nil {
		s.writeAuditLog(ctx, o.ID, "AFFILIATE_REBATE_FAILED", "system", map[string]any{
			"error": err.Error(),
		})
		return fmt.Errorf("accrue affiliate rebate: %w", err)
	}

	if rebateAmount <= 0 {
		if err := s.updateClaimedAffiliateRebateAudit(txCtx, tx.Client(), o.ID, "AFFILIATE_REBATE_SKIPPED", map[string]any{
			"baseAmount": o.Amount,
			"reason":     "no inviter bound or rebate amount <= 0",
		}); err != nil {
			s.writeAuditLog(ctx, o.ID, "AFFILIATE_REBATE_FAILED", "system", map[string]any{
				"error": err.Error(),
			})
			return fmt.Errorf("update affiliate rebate skipped audit: %w", err)
		}
		if err := tx.Commit(); err != nil {
			s.writeAuditLog(ctx, o.ID, "AFFILIATE_REBATE_FAILED", "system", map[string]any{
				"error": fmt.Sprintf("commit affiliate rebate tx: %v", err),
			})
			return fmt.Errorf("commit affiliate rebate tx: %w", err)
		}
		return nil
	}

	if err := s.updateClaimedAffiliateRebateAudit(txCtx, tx.Client(), o.ID, "AFFILIATE_REBATE_APPLIED", map[string]any{
		"baseAmount":   o.Amount,
		"rebateAmount": rebateAmount,
	}); err != nil {
		s.writeAuditLog(ctx, o.ID, "AFFILIATE_REBATE_FAILED", "system", map[string]any{
			"error": err.Error(),
		})
		return fmt.Errorf("update affiliate rebate applied audit: %w", err)
	}

	if err := tx.Commit(); err != nil {
		s.writeAuditLog(ctx, o.ID, "AFFILIATE_REBATE_FAILED", "system", map[string]any{
			"error": fmt.Sprintf("commit affiliate rebate tx: %v", err),
		})
		return fmt.Errorf("commit affiliate rebate tx: %w", err)
	}
	return nil
}

func (s *PaymentService) tryClaimAffiliateRebateAudit(ctx context.Context, client *dbent.Client, orderID int64, baseAmount float64) (bool, error) {
	if client == nil {
		return false, errors.New("nil payment client")
	}
	oid := strconv.FormatInt(orderID, 10)
	detail, _ := json.Marshal(map[string]any{
		"baseAmount": baseAmount,
		"status":     "reserved",
	})
	rows, err := client.QueryContext(ctx, `
INSERT INTO payment_audit_logs (order_id, action, detail, operator, created_at)
SELECT $1::text, 'AFFILIATE_REBATE_APPLIED', $2::text, 'system', NOW()
WHERE NOT EXISTS (
	SELECT 1
	FROM payment_audit_logs
	WHERE order_id = $1::text
	  AND action IN ('AFFILIATE_REBATE_APPLIED', 'AFFILIATE_REBATE_SKIPPED')
)
ON CONFLICT (order_id, action) DO NOTHING
RETURNING id`, oid, string(detail))
	if err != nil {
		return false, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return false, err
		}
		return false, nil
	}
	var claimID int64
	if err := rows.Scan(&claimID); err != nil {
		return false, err
	}
	return true, nil
}

func (s *PaymentService) updateClaimedAffiliateRebateAudit(ctx context.Context, client *dbent.Client, orderID int64, action string, detail map[string]any) error {
	if client == nil {
		return errors.New("nil payment client")
	}
	oid := strconv.FormatInt(orderID, 10)
	detailJSON, _ := json.Marshal(detail)
	updated, err := client.PaymentAuditLog.Update().
		Where(
			paymentauditlog.OrderIDEQ(oid),
			paymentauditlog.ActionEQ("AFFILIATE_REBATE_APPLIED"),
		).
		SetAction(action).
		SetDetail(string(detailJSON)).
		SetOperator("system").
		Save(ctx)
	if err != nil {
		return err
	}
	if updated == 0 {
		return errors.New("affiliate rebate claim log not found")
	}
	return nil
}

func distributionPaidCreditAmountForOrder(o *dbent.PaymentOrder) (float64, bool) {
	if o == nil {
		return 0, false
	}
	if o.OrderType != payment.OrderTypeBalance && o.OrderType != payment.OrderTypeSubscription {
		return 0, false
	}
	if o.Amount <= 0 || math.IsNaN(o.Amount) || math.IsInf(o.Amount, 0) {
		return 0, false
	}
	return o.Amount, true
}

func (s *PaymentService) markFailed(ctx context.Context, oid int64, cause error) {
	now := time.Now()
	r := psErrMsg(cause)
	// Only mark FAILED if still in RECHARGING state — prevents overwriting
	// a COMPLETED order when markCompleted failed but fulfillment succeeded.
	c, e := s.entClient.PaymentOrder.Update().
		Where(paymentorder.IDEQ(oid), paymentorder.StatusEQ(OrderStatusRecharging)).
		SetStatus(OrderStatusFailed).SetFailedAt(now).SetFailedReason(r).Save(ctx)
	if e != nil {
		slog.Error("mark FAILED", "orderID", oid, "error", e)
	}
	if c > 0 {
		s.writeAuditLog(ctx, oid, "FULFILLMENT_FAILED", "system", map[string]any{"reason": r})
	}
}

func (s *PaymentService) markCompletedWithClient(ctx context.Context, client *dbent.Client, o *dbent.PaymentOrder, auditAction string) error {
	now := time.Now()
	updated, err := client.PaymentOrder.Update().
		Where(paymentorder.IDEQ(o.ID), paymentorder.StatusEQ(OrderStatusRecharging)).
		SetStatus(OrderStatusCompleted).
		SetCompletedAt(now).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("mark completed: %w", err)
	}
	if updated == 0 {
		return infraerrors.Conflict("ORDER_STATE_CONFLICT", "order is not in recharging status")
	}
	if err := s.writeAuditLogWithClient(ctx, client, o.ID, auditAction, "system", map[string]any{
		"rechargeCode":   o.RechargeCode,
		"creditedAmount": o.Amount,
		"payAmount":      o.PayAmount,
	}); err != nil {
		return fmt.Errorf("write audit log: %w", err)
	}
	return nil
}

func (s *PaymentService) claimPaymentAuditAction(ctx context.Context, client *dbent.Client, orderID int64, action string, detail map[string]any) (bool, error) {
	if client == nil {
		return false, errors.New("nil payment client")
	}
	encodedDetail, _ := json.Marshal(detail)
	orderIDText := strconv.FormatInt(orderID, 10)
	query := `
INSERT INTO payment_audit_logs (order_id, action, detail, operator, created_at)
SELECT $1::text, $2::text, $3::text, 'system', NOW()
WHERE NOT EXISTS (
	SELECT 1 FROM payment_audit_logs WHERE order_id = $1::text AND action = $2::text
)`
	args := []any{orderIDText, action, string(encodedDetail)}
	if client.Driver().Dialect() == dialect.SQLite {
		query = `
INSERT INTO payment_audit_logs (order_id, action, detail, operator, created_at)
SELECT ?, ?, ?, 'system', CURRENT_TIMESTAMP
WHERE NOT EXISTS (
	SELECT 1 FROM payment_audit_logs WHERE order_id = ? AND action = ?
)`
		args = []any{orderIDText, action, string(encodedDetail), orderIDText, action}
	}
	res, err := client.ExecContext(ctx, query, args...)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (s *PaymentService) RetryFulfillment(ctx context.Context, oid int64) error {
	o, err := s.entClient.PaymentOrder.Get(ctx, oid)
	if err != nil {
		return infraerrors.NotFound("NOT_FOUND", "order not found")
	}
	if o.PaidAt == nil {
		return infraerrors.BadRequest("INVALID_STATUS", "order is not paid")
	}
	if psIsRefundStatus(o.Status) {
		return infraerrors.BadRequest("INVALID_STATUS", "refund-related order cannot retry")
	}
	if o.Status == OrderStatusRecharging {
		return infraerrors.Conflict("CONFLICT", "order is being processed")
	}
	if o.Status == OrderStatusCompleted {
		return infraerrors.BadRequest("INVALID_STATUS", "order already completed")
	}
	if o.Status != OrderStatusFailed && o.Status != OrderStatusPaid {
		return infraerrors.BadRequest("INVALID_STATUS", "only paid and failed orders can retry")
	}
	_, err = s.entClient.PaymentOrder.Update().Where(paymentorder.IDEQ(oid), paymentorder.StatusIn(OrderStatusFailed, OrderStatusPaid)).SetStatus(OrderStatusPaid).ClearFailedAt().ClearFailedReason().Save(ctx)
	if err != nil {
		return fmt.Errorf("reset for retry: %w", err)
	}
	s.writeAuditLog(ctx, oid, "RECHARGE_RETRY", "admin", map[string]any{"detail": "admin manual retry"})
	return s.executeFulfillment(ctx, oid)
}
