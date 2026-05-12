//go:build unit

package service

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestValidateRefundRequestRejectsLegacyGuessedProviderInstance(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	user, err := client.User.Create().
		SetEmail("refund-legacy@example.com").
		SetPasswordHash("hash").
		SetUsername("refund-legacy-user").
		Save(ctx)
	require.NoError(t, err)

	_, err = client.PaymentProviderInstance.Create().
		SetProviderKey(payment.TypeAlipay).
		SetName("alipay-refund-instance").
		SetConfig("{}").
		SetSupportedTypes("alipay").
		SetEnabled(true).
		SetAllowUserRefund(true).
		SetRefundEnabled(true).
		Save(ctx)
	require.NoError(t, err)

	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(88).
		SetPayAmount(88).
		SetFeeRate(0).
		SetRechargeCode("REFUND-LEGACY-ORDER").
		SetOutTradeNo("sub2_refund_legacy_order").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade-legacy-refund").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusCompleted).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetPaidAt(time.Now()).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		Save(ctx)
	require.NoError(t, err)

	svc := &PaymentService{
		entClient: client,
	}

	_, err = svc.validateRefundRequest(ctx, order.ID, user.ID)
	require.Error(t, err)
	require.Equal(t, "USER_REFUND_DISABLED", infraerrors.Reason(err))
}

func TestPrepareRefundRejectsLegacyGuessedProviderInstance(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	user, err := client.User.Create().
		SetEmail("refund-legacy-admin@example.com").
		SetPasswordHash("hash").
		SetUsername("refund-legacy-admin-user").
		Save(ctx)
	require.NoError(t, err)

	_, err = client.PaymentProviderInstance.Create().
		SetProviderKey(payment.TypeAlipay).
		SetName("alipay-refund-admin-instance").
		SetConfig("{}").
		SetSupportedTypes("alipay").
		SetEnabled(true).
		SetAllowUserRefund(true).
		SetRefundEnabled(true).
		Save(ctx)
	require.NoError(t, err)

	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(188).
		SetPayAmount(188).
		SetFeeRate(0).
		SetRechargeCode("REFUND-LEGACY-ADMIN-ORDER").
		SetOutTradeNo("sub2_refund_legacy_admin_order").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade-legacy-admin-refund").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusCompleted).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetPaidAt(time.Now()).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		Save(ctx)
	require.NoError(t, err)

	svc := &PaymentService{
		entClient: client,
	}

	plan, result, err := svc.PrepareRefund(ctx, order.ID, 0, "", false, false)
	require.Nil(t, plan)
	require.Nil(t, result)
	require.Error(t, err)
	require.Equal(t, "REFUND_DISABLED", infraerrors.Reason(err))
}

func TestGwRefundRejectsAlipayMerchantIdentitySnapshotMismatch(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	user, err := client.User.Create().
		SetEmail("refund-snapshot-mismatch@example.com").
		SetPasswordHash("hash").
		SetUsername("refund-snapshot-mismatch-user").
		Save(ctx)
	require.NoError(t, err)

	inst, err := client.PaymentProviderInstance.Create().
		SetProviderKey(payment.TypeAlipay).
		SetName("alipay-refund-mismatch-instance").
		SetConfig(encryptWebhookProviderConfig(t, map[string]string{
			"appId":      "runtime-alipay-app",
			"privateKey": "runtime-private-key",
		})).
		SetSupportedTypes("alipay").
		SetEnabled(true).
		SetRefundEnabled(true).
		Save(ctx)
	require.NoError(t, err)

	instID := strconv.FormatInt(inst.ID, 10)
	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(88).
		SetPayAmount(88).
		SetFeeRate(0).
		SetRechargeCode("REFUND-SNAPSHOT-MISMATCH-ORDER").
		SetOutTradeNo("sub2_refund_snapshot_mismatch_order").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade-refund-snapshot-mismatch").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusCompleted).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetPaidAt(time.Now()).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		SetProviderInstanceID(instID).
		SetProviderKey(payment.TypeAlipay).
		SetProviderSnapshot(map[string]any{
			"schema_version":       2,
			"provider_instance_id": instID,
			"provider_key":         payment.TypeAlipay,
			"merchant_app_id":      "expected-alipay-app",
		}).
		Save(ctx)
	require.NoError(t, err)

	svc := &PaymentService{
		entClient:    client,
		loadBalancer: newWebhookProviderTestLoadBalancer(client),
	}

	_, err = svc.gwRefund(ctx, &RefundPlan{
		OrderID:       order.ID,
		Order:         order,
		RefundAmount:  order.Amount,
		GatewayAmount: order.Amount,
		Reason:        "snapshot mismatch",
	})
	require.ErrorContains(t, err, "alipay app_id mismatch")
}

type refundAffiliateRepoStub struct {
	authRegisterAffiliateRepoStub
	reverseErr       error
	reversed         bool
	reverseCalls     int
	lastUserID       int64
	lastSourceOrder  int64
	lastReverseValue float64
}

func (s *refundAffiliateRepoStub) RecordPaidCredit(context.Context, int64, int64, float64, time.Time) (bool, error) {
	return false, nil
}

func (s *refundAffiliateRepoStub) ReversePaidCredit(_ context.Context, userID, sourceOrderID int64, amountUSD float64, _ time.Time) (bool, error) {
	s.reverseCalls++
	s.lastUserID = userID
	s.lastSourceOrder = sourceOrderID
	s.lastReverseValue = amountUSD
	if s.reverseErr != nil {
		return false, s.reverseErr
	}
	return s.reversed, nil
}

func createRefundMarkOrder(t *testing.T, ctx context.Context, client *dbent.Client, suffix string, status string) *dbent.PaymentOrder {
	t.Helper()

	user, err := client.User.Create().
		SetEmail("refund-mark-" + suffix + "@example.com").
		SetPasswordHash("hash").
		SetUsername("refund-mark-" + suffix).
		Save(ctx)
	require.NoError(t, err)

	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(88).
		SetPayAmount(88).
		SetFeeRate(0).
		SetRechargeCode("REFUND-MARK-" + suffix).
		SetOutTradeNo("sub2_refund_mark_" + suffix).
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade-refund-mark-" + suffix).
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(status).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetPaidAt(time.Now()).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		Save(ctx)
	require.NoError(t, err)
	return order
}

func TestMarkRefundOkFailsRetryablyWhenPaidCreditReversalFails(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	order := createRefundMarkOrder(t, ctx, client, "reversal-fail", OrderStatusRefunding)
	repo := &refundAffiliateRepoStub{reverseErr: errors.New("reversal db down")}
	svc := &PaymentService{
		entClient:        client,
		affiliateService: &AffiliateService{repo: repo},
	}

	result, err := svc.markRefundOk(ctx, &RefundPlan{
		OrderID:       order.ID,
		Order:         order,
		RefundAmount:  order.Amount,
		GatewayAmount: order.PayAmount,
		Reason:        "test reversal failure",
	})

	require.Nil(t, result)
	require.Error(t, err)
	require.Equal(t, 1, repo.reverseCalls)
	updated, err := client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, OrderStatusRefundFailed, updated.Status)
	require.NotNil(t, updated.FailedReason)
	require.Contains(t, *updated.FailedReason, "affiliate distribution paid credit reversal failed")
	require.True(t, svc.hasAuditLog(ctx, order.ID, "AFFILIATE_DISTRIBUTION_PAID_CREDIT_REVERSAL_FAILED"))
	require.False(t, svc.hasAuditLog(ctx, order.ID, "REFUND_SUCCESS"))
}

func TestMarkRefundOkReversesPaidCreditBeforeFinalRefundStatus(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	order := createRefundMarkOrder(t, ctx, client, "reversal-ok", OrderStatusRefunding)
	repo := &refundAffiliateRepoStub{reversed: true}
	svc := &PaymentService{
		entClient:        client,
		affiliateService: &AffiliateService{repo: repo},
	}

	result, err := svc.markRefundOk(ctx, &RefundPlan{
		OrderID:       order.ID,
		Order:         order,
		RefundAmount:  order.Amount,
		GatewayAmount: order.PayAmount,
		Reason:        "test reversal success",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Success)
	require.Equal(t, 1, repo.reverseCalls)
	require.Equal(t, order.UserID, repo.lastUserID)
	require.Equal(t, order.ID, repo.lastSourceOrder)
	require.InDelta(t, order.Amount, repo.lastReverseValue, 1e-9)
	updated, err := client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, OrderStatusRefunded, updated.Status)
	require.True(t, svc.hasAuditLog(ctx, order.ID, "AFFILIATE_DISTRIBUTION_PAID_CREDIT_REVERSED"))
	require.True(t, svc.hasAuditLog(ctx, order.ID, "REFUND_SUCCESS"))
}
