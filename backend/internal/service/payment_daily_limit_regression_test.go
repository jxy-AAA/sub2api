//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestCheckDailyLimitCountsPendingOrders(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	user, err := client.User.Create().
		SetEmail("daily-limit-pending@example.com").
		SetPasswordHash("hash").
		SetUsername("daily-limit-pending").
		Save(ctx)
	require.NoError(t, err)

	_, err = client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(60).
		SetPayAmount(60).
		SetFeeRate(0).
		SetRechargeCode("PENDING-LIMIT-1").
		SetOutTradeNo("sub2_pending_limit_1").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusPending).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		Save(ctx)
	require.NoError(t, err)

	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	svc := &PaymentService{}
	err = svc.checkDailyLimit(ctx, tx, user.ID, 50, 100, true)
	require.Error(t, err)
	require.Equal(t, "DAILY_LIMIT_EXCEEDED", infraerrors.Reason(err))
}

func TestToPaidRechecksDailyLimitBeforeTransition(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	user, err := client.User.Create().
		SetEmail("daily-limit-confirm@example.com").
		SetPasswordHash("hash").
		SetUsername("daily-limit-confirm").
		Save(ctx)
	require.NoError(t, err)

	paidAt := time.Now()
	_, err = client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(80).
		SetPayAmount(80).
		SetFeeRate(0).
		SetRechargeCode("PAID-LIMIT-EXISTING").
		SetOutTradeNo("sub2_paid_limit_existing").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade-existing").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusCompleted).
		SetPaidAt(paidAt).
		SetCompletedAt(paidAt).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		Save(ctx)
	require.NoError(t, err)

	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(30).
		SetPayAmount(30).
		SetFeeRate(0).
		SetRechargeCode("PAID-LIMIT-PENDING").
		SetOutTradeNo("sub2_paid_limit_pending").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusPending).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		Save(ctx)
	require.NoError(t, err)

	settingRepo := &paymentConfigSettingRepoStub{
		values: map[string]string{
			SettingPaymentEnabled:     "true",
			SettingDailyRechargeLimit: "100",
		},
	}
	configSvc := NewPaymentConfigService(client, settingRepo, nil)
	svc := &PaymentService{
		entClient:     client,
		configService: configSvc,
	}

	err = svc.toPaid(ctx, order, "trade-new", 30, payment.TypeAlipay)
	require.Error(t, err)
	require.Equal(t, "DAILY_LIMIT_EXCEEDED", infraerrors.Reason(err))

	reloaded, reloadErr := client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, reloadErr)
	require.Equal(t, paymentorder.Status(OrderStatusPending), reloaded.Status)
	require.Empty(t, reloaded.PaymentTradeNo)
	require.Nil(t, reloaded.PaidAt)
}
