//go:build unit

package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentauditlog"
	"github.com/Wei-Shaw/sub2api/ent/usersubscription"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

type entGroupRepoForPayment struct {
	groupRepoNoop
	client *dbent.Client
}

func (r *entGroupRepoForPayment) GetByID(ctx context.Context, id int64) (*Group, error) {
	entity, err := r.client.Group.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return &Group{
		ID:               entity.ID,
		Name:             entity.Name,
		Status:           entity.Status,
		SubscriptionType: entity.SubscriptionType,
	}, nil
}

type entUserSubRepoForPayment struct {
	userSubRepoNoop
	client *dbent.Client
}

func (r *entUserSubRepoForPayment) clientFromCtx(ctx context.Context) *dbent.Client {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return tx.Client()
	}
	return r.client
}

func (r *entUserSubRepoForPayment) Create(ctx context.Context, sub *UserSubscription) error {
	client := r.clientFromCtx(ctx)
	entity, err := client.UserSubscription.Create().
		SetUserID(sub.UserID).
		SetGroupID(sub.GroupID).
		SetStartsAt(sub.StartsAt).
		SetExpiresAt(sub.ExpiresAt).
		SetStatus(sub.Status).
		SetAssignedAt(sub.AssignedAt).
		SetNotes(sub.Notes).
		Save(ctx)
	if err != nil {
		return err
	}
	sub.ID = entity.ID
	return nil
}

func (r *entUserSubRepoForPayment) GetByID(ctx context.Context, id int64) (*UserSubscription, error) {
	client := r.clientFromCtx(ctx)
	entity, err := client.UserSubscription.Get(ctx, id)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, err
	}
	return &UserSubscription{
		ID:        entity.ID,
		UserID:    entity.UserID,
		GroupID:   entity.GroupID,
		StartsAt:  entity.StartsAt,
		ExpiresAt: entity.ExpiresAt,
		Status:    entity.Status,
		AssignedAt: entity.AssignedAt,
		Notes: func() string {
			if entity.Notes == nil {
				return ""
			}
			return *entity.Notes
		}(),
	}, nil
}

func (r *entUserSubRepoForPayment) GetByUserIDAndGroupID(ctx context.Context, userID, groupID int64) (*UserSubscription, error) {
	client := r.clientFromCtx(ctx)
	entity, err := client.UserSubscription.Query().
		Where(
			usersubscription.UserIDEQ(userID),
			usersubscription.GroupIDEQ(groupID),
		).Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, err
	}
	return &UserSubscription{
		ID:        entity.ID,
		UserID:    entity.UserID,
		GroupID:   entity.GroupID,
		StartsAt:  entity.StartsAt,
		ExpiresAt: entity.ExpiresAt,
		Status:    entity.Status,
		AssignedAt: entity.AssignedAt,
		Notes: func() string {
			if entity.Notes == nil {
				return ""
			}
			return *entity.Notes
		}(),
	}, nil
}

func (r *entUserSubRepoForPayment) ExtendExpiry(ctx context.Context, subscriptionID int64, newExpiresAt time.Time) error {
	client := r.clientFromCtx(ctx)
	_, err := client.UserSubscription.UpdateOneID(subscriptionID).SetExpiresAt(newExpiresAt).Save(ctx)
	return err
}

func (r *entUserSubRepoForPayment) UpdateStatus(ctx context.Context, subscriptionID int64, status string) error {
	client := r.clientFromCtx(ctx)
	_, err := client.UserSubscription.UpdateOneID(subscriptionID).SetStatus(status).Save(ctx)
	return err
}

func (r *entUserSubRepoForPayment) UpdateNotes(ctx context.Context, subscriptionID int64, notes string) error {
	client := r.clientFromCtx(ctx)
	_, err := client.UserSubscription.UpdateOneID(subscriptionID).SetNotes(notes).Save(ctx)
	return err
}

func TestAssignOrExtendSubscriptionConcurrentRenewalsAccumulate(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	if client.Driver().Dialect() != dialect.Postgres {
		t.Skip("requires postgres row-lock semantics for deterministic concurrent renewal coverage")
	}

	user, err := client.User.Create().
		SetEmail("concurrent-renew@example.com").
		SetPasswordHash("hash").
		SetUsername("concurrent-renew").
		Save(ctx)
	require.NoError(t, err)

	group, err := client.Group.Create().
		SetName("concurrent-renew-group").
		SetStatus(StatusActive).
		SetSubscriptionType(SubscriptionTypeSubscription).
		Save(ctx)
	require.NoError(t, err)

	initialExpiry := time.Now().UTC().AddDate(0, 0, 10).Truncate(time.Second)
	_, err = client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(group.ID).
		SetStartsAt(time.Now().UTC().Add(-time.Hour)).
		SetExpiresAt(initialExpiry).
		SetStatus(SubscriptionStatusActive).
		SetAssignedAt(time.Now().UTC()).
		SetNotes("seed").
		Save(ctx)
	require.NoError(t, err)

	groupRepo := &entGroupRepoForPayment{client: client}
	userSubRepo := &entUserSubRepoForPayment{client: client}
	subSvc := NewSubscriptionService(groupRepo, userSubRepo, nil, client, nil)

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	start := make(chan struct{})
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			callCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			_, _, callErr := subSvc.AssignOrExtendSubscription(callCtx, &AssignSubscriptionInput{
				UserID:       user.ID,
				GroupID:      group.ID,
				ValidityDays: 10,
				Notes:        fmt.Sprintf("renew-%d", index),
			})
			errs <- callErr
		}(i)
	}
	close(start)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(8 * time.Second):
		t.Fatal("concurrent renewals timed out")
	}
	close(errs)

	for callErr := range errs {
		require.NoError(t, callErr)
	}

	updated, err := userSubRepo.GetByUserIDAndGroupID(ctx, user.ID, group.ID)
	require.NoError(t, err)
	require.Equal(t, initialExpiry.AddDate(0, 0, 20), updated.ExpiresAt)
}

func TestSubscriptionFulfillmentClaimPreventsDuplicateExtensionOnRetry(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	user, err := client.User.Create().
		SetEmail("fulfillment-claim@example.com").
		SetPasswordHash("hash").
		SetUsername("fulfillment-claim").
		Save(ctx)
	require.NoError(t, err)

	group, err := client.Group.Create().
		SetName("fulfillment-claim-group").
		SetStatus(StatusActive).
		SetSubscriptionType(SubscriptionTypeSubscription).
		Save(ctx)
	require.NoError(t, err)

	initialExpiry := time.Now().UTC().AddDate(0, 0, 5).Truncate(time.Second)
	_, err = client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(group.ID).
		SetStartsAt(time.Now().UTC().Add(-time.Hour)).
		SetExpiresAt(initialExpiry).
		SetStatus(SubscriptionStatusActive).
		SetAssignedAt(time.Now().UTC()).
		SetNotes("seed").
		Save(ctx)
	require.NoError(t, err)

	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(99).
		SetPayAmount(99).
		SetFeeRate(0).
		SetRechargeCode("SUB-CLAIM").
		SetOutTradeNo("sub2_claim_test_order").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade-claim").
		SetOrderType(payment.OrderTypeSubscription).
		SetSubscriptionGroupID(group.ID).
		SetSubscriptionDays(15).
		SetStatus(OrderStatusPaid).
		SetPaidAt(time.Now().UTC()).
		SetExpiresAt(time.Now().UTC().Add(time.Hour)).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		Save(ctx)
	require.NoError(t, err)

	groupRepo := &entGroupRepoForPayment{client: client}
	userSubRepo := &entUserSubRepoForPayment{client: client}
	subSvc := NewSubscriptionService(groupRepo, userSubRepo, nil, client, nil)
	paymentSvc := NewPaymentService(client, payment.NewRegistry(), nil, nil, subSvc, nil, nil, groupRepo, nil)

	require.NoError(t, paymentSvc.ExecuteSubscriptionFulfillment(ctx, order.ID))

	afterFirst, err := userSubRepo.GetByUserIDAndGroupID(ctx, user.ID, group.ID)
	require.NoError(t, err)
	firstExpiry := afterFirst.ExpiresAt

	_, err = client.PaymentAuditLog.Delete().
		Where(
			paymentauditlog.OrderIDEQ(fmt.Sprintf("%d", order.ID)),
			paymentauditlog.ActionEQ("SUBSCRIPTION_SUCCESS"),
		).
		Exec(ctx)
	require.NoError(t, err)

	_, err = client.PaymentOrder.UpdateOneID(order.ID).
		SetStatus(OrderStatusFailed).
		ClearCompletedAt().
		Save(ctx)
	require.NoError(t, err)

	require.NoError(t, paymentSvc.ExecuteSubscriptionFulfillment(ctx, order.ID))

	afterRetry, err := userSubRepo.GetByUserIDAndGroupID(ctx, user.ID, group.ID)
	require.NoError(t, err)
	require.Equal(t, firstExpiry, afterRetry.ExpiresAt)
}
