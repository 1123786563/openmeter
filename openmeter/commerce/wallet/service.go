// Package wallet implements the read-only Wallet aggregation. The Wallet is a
// pure projection over the immutable Credit Ledger, valid allocations, and
// enterprise receivables — it never holds a mutable second balance.
//
// Bucket priorities are fixed: plan=10, gift=20, recharge=30,
// enterprise_receivable=40. Within the same priority, earliest expiry and then
// earliest creation break ties.
package wallet

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/openmeterio/openmeter/openmeter/commerce"
	"github.com/openmeterio/openmeter/pkg/clock"
)

// DataPort provides the read-model inputs the wallet needs. All data originates
// from the Credit Ledger and related immutable facts.
type DataPort interface {
	GetGrants(ctx context.Context, namespace, customerID string) ([]commerce.AllocationGrant, error)
	GetEnterpriseReceivable(ctx context.Context, namespace, customerID string) (*commerce.EnterpriseReceivable, error)
	GetRecentTransactions(ctx context.Context, namespace, customerID string, limit int) ([]commerce.WalletTransaction, error)
}

// Service is the read-only wallet aggregation interface.
type Service interface {
	GetWallet(ctx context.Context, namespace, customerID string) (*commerce.Wallet, error)
}

// Config wires the wallet service.
type Config struct {
	Port   DataPort
	Logger *slog.Logger
}

type service struct {
	port   DataPort
	logger *slog.Logger
}

// New creates a wallet Service from the given Config.
func New(cfg Config) Service {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &service{port: cfg.Port, logger: logger}
}

// GetWallet computes the full wallet view for a customer. It reads grants,
// enterprise receivables, and recent transactions from the data port and
// projects them into buckets.
func (s *service) GetWallet(ctx context.Context, namespace, customerID string) (*commerce.Wallet, error) {
	grants, err := s.port.GetGrants(ctx, namespace, customerID)
	if err != nil {
		return nil, fmt.Errorf("wallet: get grants: %w", err)
	}

	receivable, err := s.port.GetEnterpriseReceivable(ctx, namespace, customerID)
	if err != nil {
		return nil, fmt.Errorf("wallet: get enterprise receivable: %w", err)
	}

	txns, err := s.port.GetRecentTransactions(ctx, namespace, customerID, 50)
	if err != nil {
		return nil, fmt.Errorf("wallet: get transactions: %w", err)
	}

	buckets := AggregateBuckets(grants, receivable)
	total := int64(0)
	for _, b := range buckets {
		total += b.AvailableCredits
	}

	return &commerce.Wallet{
		CustomerID:            customerID,
		ContractVersion:       commerce.ContractVersion,
		TotalAvailableCredits: total,
		Buckets:               buckets,
		Transactions:          txns,
		RetrievedAt:           clock.Now(),
	}, nil
}

// AggregateBuckets converts allocation grants and an optional enterprise
// receivable into ordered wallet buckets. Buckets are sorted by consumption
// priority (ascending). This is pure computation with no side effects — ideal
// for table-driven testing.
func AggregateBuckets(grants []commerce.AllocationGrant, receivable *commerce.EnterpriseReceivable) []commerce.WalletBucket {
	// Group grants by source and aggregate.
	type bucketAcc struct {
		available  int64
		refundable int64
		expiresAt  *time.Time
	}
	acc := make(map[commerce.BucketSource]*bucketAcc)

	for _, g := range grants {
		if g.Granted-g.Consumed <= 0 {
			continue // skip fully consumed or over-consumed grants
		}
		b := acc[g.Source]
		if b == nil {
			b = &bucketAcc{}
			acc[g.Source] = b
		}
		b.available += g.Granted - g.Consumed
		if g.Source == commerce.BucketSourceRecharge {
			b.refundable += g.Refundable
		}
		// Track earliest expiry within the bucket.
		if g.ExpiresAt != nil {
			if b.expiresAt == nil || g.ExpiresAt.Before(*b.expiresAt) {
				b.expiresAt = g.ExpiresAt
			}
		}
	}

	buckets := make([]commerce.WalletBucket, 0, len(acc)+1)
	for src, b := range acc {
		bk := commerce.WalletBucket{
			Source:           src,
			AvailableCredits: b.available,
			ExpiresAt:        b.expiresAt,
		}
		if src == commerce.BucketSourceRecharge {
			rc := b.refundable
			bk.RefundableCredits = &rc
		}
		buckets = append(buckets, bk)
	}

	// Enterprise receivable bucket: ceiling minus used.
	if receivable != nil && receivable.CeilingCredits > 0 {
		avail := receivable.CeilingCredits - receivable.UsedCredits
		if avail < 0 {
			avail = 0
		}
		buckets = append(buckets, commerce.WalletBucket{
			Source:           commerce.BucketSourceEnterpriseReceivable,
			AvailableCredits: avail,
		})
	}

	// Sort by consumption priority (ascending = burn order).
	sort.Slice(buckets, func(i, j int) bool {
		return commerce.SourcePriority(buckets[i].Source) < commerce.SourcePriority(buckets[j].Source)
	})

	return buckets
}

var _ Service = (*service)(nil)
