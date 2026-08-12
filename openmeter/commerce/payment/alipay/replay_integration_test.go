package alipay

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/commerce"
	"github.com/openmeterio/openmeter/openmeter/commerce/payment"
)

func TestPaymentServiceAcceptsCallbackReplayWithChangedDeliveryTimeAndTimezone(t *testing.T) {
	keys := newTestKeys(t)
	adapter := newTestAdapter(t, "https://openapi.alipay.com/gateway.do", keys)
	adapter.now = func() time.Time { return time.Now().In(time.UTC) }
	service, store := newReplayPaymentService(t, adapter)

	firstBody := buildAlipayCallback(t, keys.alipayPrivate, map[string]string{
		"notify_id":   "stable-notify-id",
		"notify_time": "2027-01-15 09:00:00",
		"gmt_payment": "2027-01-15 08:01:02",
	})
	first, err := service.HandleCallback(t.Context(), "ns", payment.ProviderAlipay, nil, firstBody)
	require.NoError(t, err)
	require.False(t, first.AlreadyPaid)

	adapter.now = func() time.Time { return time.Now().In(time.FixedZone("delivery-west", -7*60*60)) }
	replayedBody := buildAlipayCallback(t, keys.alipayPrivate, map[string]string{
		"notify_id":   "stable-notify-id",
		"notify_time": "2027-01-16 10:00:00",
		"gmt_payment": "2027-01-15 08:01:02",
	})
	replayed, err := service.HandleCallback(t.Context(), "ns", payment.ProviderAlipay, nil, replayedBody)
	require.NoError(t, err)
	require.True(t, replayed.AlreadyPaid)
	require.Equal(t, first.Fact.ID, replayed.Fact.ID)
	require.Equal(t, first.Fact.Timestamp, replayed.Fact.Timestamp)
	require.Len(t, store.facts, 1)
}

type replayPaymentStore struct {
	attempt payment.PaymentAttempt
	facts   []payment.PaymentFactRecord
}

func newReplayPaymentService(t *testing.T, adapter *Adapter) (payment.Service, *replayPaymentStore) {
	t.Helper()
	store := &replayPaymentStore{attempt: payment.PaymentAttempt{
		ID: "attempt-1", Namespace: "ns", OrderID: "order-1", CustomerID: "customer-1",
		Provider: payment.ProviderAlipay, ProviderOrderID: "01ORDER", Status: payment.AttemptStatusPending,
		AmountMinor: 10000, Currency: "CNY", ExpectedMerchantID: "ali-seller", ExpectedApplicationID: "ali-app",
	}}
	service, err := payment.New(payment.Config{
		Attempts: store,
		Facts:    store,
		Orders:   replayOrderPort{},
		TxRunner: replayPaidTxRunner{store: store},
		Providers: map[payment.Provider]payment.ProviderAdapter{
			payment.ProviderAlipay: adapter,
		},
		Logger: slog.New(slog.DiscardHandler),
	})
	require.NoError(t, err)
	return service, store
}

func (s *replayPaymentStore) CreateAttempt(context.Context, payment.PaymentAttempt) (*payment.PaymentAttempt, bool, error) {
	copy := s.attempt
	return &copy, false, nil
}

func (s *replayPaymentStore) GetAttempt(_ context.Context, namespace, id string) (*payment.PaymentAttempt, error) {
	if s.attempt.Namespace != namespace || s.attempt.ID != id {
		return nil, payment.ErrPaymentAttemptNotFound
	}
	copy := s.attempt
	return &copy, nil
}

func (s *replayPaymentStore) GetAttemptByIdempotencyKey(context.Context, string, string, string) (*payment.PaymentAttempt, error) {
	return nil, payment.ErrPaymentAttemptNotFound
}

func (s *replayPaymentStore) GetAttemptByProviderOrder(_ context.Context, namespace string, provider payment.Provider, providerOrderID string) (*payment.PaymentAttempt, error) {
	if s.attempt.Namespace != namespace || s.attempt.Provider != provider || s.attempt.ProviderOrderID != providerOrderID {
		return nil, payment.ErrPaymentAttemptNotFound
	}
	copy := s.attempt
	return &copy, nil
}

func (s *replayPaymentStore) UpdateAttemptStatus(_ context.Context, namespace, id string, expectedFrom, to payment.AttemptStatus) (*payment.PaymentAttempt, error) {
	if s.attempt.Namespace != namespace || s.attempt.ID != id || s.attempt.Status != expectedFrom {
		return nil, errors.New("invalid attempt transition")
	}
	s.attempt.Status = to
	copy := s.attempt
	return &copy, nil
}

func (s *replayPaymentStore) SetProviderIDs(context.Context, string, string, string, string, string) (*payment.PaymentAttempt, error) {
	copy := s.attempt
	return &copy, nil
}

func (s *replayPaymentStore) InsertFact(_ context.Context, fact payment.PaymentFactRecord) (*payment.PaymentFactRecord, bool, error) {
	for i := range s.facts {
		if fact.RawHash != "" && s.facts[i].RawHash == fact.RawHash {
			copy := s.facts[i]
			return &copy, false, nil
		}
	}
	s.facts = append(s.facts, fact)
	copy := fact
	return &copy, true, nil
}

func (s *replayPaymentStore) GetFactByRawHash(_ context.Context, namespace, rawHash string) (*payment.PaymentFactRecord, error) {
	for i := range s.facts {
		if s.facts[i].Namespace == namespace && s.facts[i].RawHash == rawHash {
			copy := s.facts[i]
			return &copy, nil
		}
	}
	return nil, commerce.ErrPaymentFactNotFound
}

func (s *replayPaymentStore) GetFactsByProviderOrder(_ context.Context, namespace string, provider payment.Provider, providerOrderID string) ([]payment.PaymentFactRecord, error) {
	var result []payment.PaymentFactRecord
	for i := range s.facts {
		if s.facts[i].Namespace == namespace && s.facts[i].Provider == provider && s.facts[i].ProviderOrderID == providerOrderID {
			result = append(result, s.facts[i])
		}
	}
	return result, nil
}

func (s *replayPaymentStore) GetFactByProviderEvent(_ context.Context, namespace string, provider payment.Provider, providerEventID string) (*payment.PaymentFactRecord, error) {
	for i := range s.facts {
		if s.facts[i].Namespace == namespace && s.facts[i].Provider == provider && s.facts[i].ProviderEventID == providerEventID {
			copy := s.facts[i]
			return &copy, nil
		}
	}
	return nil, commerce.ErrPaymentFactNotFound
}

type replayPaidTxRunner struct{ store *replayPaymentStore }

func (r replayPaidTxRunner) RunPaidTransition(ctx context.Context, input payment.PaidTransitionInput) (payment.PaidTransitionResult, error) {
	fact, inserted, err := r.store.InsertFact(ctx, input.Fact)
	if err != nil {
		return payment.PaidTransitionResult{}, err
	}
	alreadyPaid := r.store.attempt.Status == payment.AttemptStatusSucceeded
	if inserted {
		r.store.attempt.Status = payment.AttemptStatusSucceeded
	}
	return payment.PaidTransitionResult{Fact: fact, AlreadyPaid: alreadyPaid}, nil
}

type replayOrderPort struct{}

func (replayOrderPort) UpdateOrderStatus(context.Context, string, string, commerce.OrderStatus, commerce.OrderStatus) (*commerce.Order, error) {
	return &commerce.Order{}, nil
}

func (replayOrderPort) GetOrder(context.Context, string, string) (*commerce.Order, error) {
	return &commerce.Order{}, nil
}
