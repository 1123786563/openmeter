package worker

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/creditreservationoutbox"
)

// EntRepository is the production storage implementation for the credit usage
// relay. Each transition is owner-conditional so a lease reclaimed by another
// worker cannot be acknowledged by its former owner.
type EntRepository struct{ db *entdb.Client }

func NewEntRepository(db *entdb.Client) (*EntRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("credit reservation worker: ent client is required")
	}
	return &EntRepository{db: db}, nil
}

func (r *EntRepository) Claim(ctx context.Context, owner string, limit int, lease time.Duration) ([]OutboxRow, error) {
	if limit <= 0 {
		return nil, nil
	}
	now := time.Now().UTC()
	until := now.Add(lease)
	tx, err := r.db.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("start outbox claim transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	rows, err := tx.CreditReservationOutbox.Query().Where(
		creditreservationoutbox.PublishedEQ(false), creditreservationoutbox.DeadLetteredEQ(false),
		creditreservationoutbox.Or(creditreservationoutbox.LeasedUntilIsNil(), creditreservationoutbox.LeasedUntilLTE(now)),
	).Order(entdb.Asc(creditreservationoutbox.FieldCreatedAt)).Limit(limit).ForUpdate(sql.WithLockAction(sql.SkipLocked)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query credit usage outbox: %w", err)
	}
	out := make([]OutboxRow, 0, len(rows))
	for _, row := range rows {
		if err := tx.CreditReservationOutbox.UpdateOneID(row.ID).SetOwner(owner).SetLeasedUntil(until).AddClaimCount(1).Exec(ctx); err != nil {
			return nil, fmt.Errorf("claim credit usage outbox %s: %w", row.ID, err)
		}
		out = append(out, OutboxRow{ID: row.ID, Namespace: row.Namespace, EventType: row.EventType, Subject: row.AggregateID, OccurredAt: row.CreatedAt, Payload: row.Payload, Owner: owner, ClaimCount: row.ClaimCount + 1})
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit outbox claim: %w", err)
	}
	return out, nil
}

func (r *EntRepository) MarkPublished(ctx context.Context, owner, id string) error {
	now := time.Now().UTC()
	affected, err := r.db.CreditReservationOutbox.Update().Where(creditreservationoutbox.IDEQ(id), creditreservationoutbox.OwnerEQ(owner), creditreservationoutbox.PublishedEQ(false)).SetPublished(true).SetPublishedAt(now).SetOwner("").ClearLeasedUntil().Save(ctx)
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf("outbox %s lease lost before publish acknowledgement", id)
	}
	return nil
}
func (r *EntRepository) Release(ctx context.Context, owner, id string) error {
	_, err := r.db.CreditReservationOutbox.Update().Where(creditreservationoutbox.IDEQ(id), creditreservationoutbox.OwnerEQ(owner), creditreservationoutbox.PublishedEQ(false)).SetOwner("").ClearLeasedUntil().Save(ctx)
	return err
}
func (r *EntRepository) MarkDeadLetter(ctx context.Context, owner, id, reason string) error {
	affected, err := r.db.CreditReservationOutbox.Update().Where(creditreservationoutbox.IDEQ(id), creditreservationoutbox.OwnerEQ(owner), creditreservationoutbox.PublishedEQ(false)).SetDeadLettered(true).SetDeadLetterReason(reason).SetOwner("").ClearLeasedUntil().Save(ctx)
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf("outbox %s lease lost before dead letter", id)
	}
	return nil
}
