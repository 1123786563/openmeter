package wechat

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/commerce"
	"github.com/openmeterio/openmeter/openmeter/commerce/payment"
)

func TestPaymentServiceAcceptsCallbackRetryWithFreshTransportSignatureTime(t *testing.T) {
	keys := newTestKeys(t)
	headers, body := encryptedCallback(t, keys.platformPrivate, testAPIv3Key, testNowUnix)
	adapter := newTestAdapter(t, "https://api.mch.weixin.qq.com", keys)
	service, store := newReplayPaymentService(t, adapter)

	first, err := service.HandleCallback(t.Context(), "ns", payment.ProviderWeChat, map[string][]string(headers), body)
	require.NoError(t, err)
	require.False(t, first.AlreadyPaid)

	freshTimestamp := strconv.FormatInt(testNowUnix+60, 10)
	freshHeaders := headers.Clone()
	freshHeaders.Set("Wechatpay-Timestamp", freshTimestamp)
	freshHeaders.Set("Wechatpay-Signature", signWechatMessage(
		t,
		keys.platformPrivate,
		freshTimestamp+"\n"+freshHeaders.Get("Wechatpay-Nonce")+"\n"+string(body)+"\n",
	))

	replayed, err := service.HandleCallback(t.Context(), "ns", payment.ProviderWeChat, map[string][]string(freshHeaders), body)
	require.NoError(t, err)
	require.True(t, replayed.AlreadyPaid)
	require.Equal(t, first.Fact.ID, replayed.Fact.ID)
	require.Len(t, store.facts, 1)
	require.Equal(t, first.Fact.Timestamp, replayed.Fact.Timestamp)
}

func TestPaymentServiceAcceptsRepeatedNonSuccessQueryWithUnchangedBody(t *testing.T) {
	keys := newTestKeys(t)
	const body = `{"appid":"wx-app","mchid":"wx-mch","out_trade_no":"01ORDER","trade_state":"NOTPAY","amount":{"total":10000,"currency":"CNY"}}`
	var nowUnix atomic.Int64
	nowUnix.Store(testNowUnix)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeSignedWechatResponseAt(t, keys.platformPrivate, w, http.StatusOK, body, nowUnix.Load())
	}))
	defer server.Close()

	adapter := newTestAdapter(t, server.URL, keys)
	adapter.now = func() time.Time { return time.Unix(nowUnix.Load(), 0) }
	service, store := newReplayPaymentService(t, adapter)

	first, err := service.ConfirmPayment(t.Context(), "ns", "attempt-1")
	require.NoError(t, err)
	require.False(t, first.Fact.Success)
	require.True(t, first.Fact.Timestamp.IsZero())

	nowUnix.Store(testNowUnix + 60)
	replayed, err := service.ConfirmPayment(t.Context(), "ns", "attempt-1")
	require.NoError(t, err)
	require.Equal(t, first.Fact.ID, replayed.Fact.ID)
	require.Equal(t, first.Fact.RawHash, replayed.Fact.RawHash)
	require.True(t, replayed.Fact.Timestamp.IsZero())
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
		Provider: payment.ProviderWeChat, ProviderOrderID: "01ORDER", Status: payment.AttemptStatusPending,
		AmountMinor: 10000, Currency: "CNY", ExpectedMerchantID: "wx-mch", ExpectedApplicationID: "wx-app",
	}}
	service, err := payment.New(payment.Config{
		Attempts: store,
		Facts:    store,
		Orders:   replayOrderPort{},
		TxRunner: replayPaidTxRunner{store: store},
		Providers: map[payment.Provider]payment.ProviderAdapter{
			payment.ProviderWeChat: adapter,
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
