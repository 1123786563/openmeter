package adapter

import (
	"context"
	"fmt"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/aiusagebatch"
	"github.com/openmeterio/openmeter/openmeter/ent/db/aiusagewatermark"
)

// AdvanceWatermark advances the continuous watermark for (namespace, subject_id).
//
// Watermark semantics:
//   - covered_seq tracks the highest contiguous seq with no gaps.
//   - If seq == covered_seq + 1, the watermark advances to seq and then catches
//     up through any already-stored batches with higher contiguous seqs.
//   - If seq <= covered_seq, the watermark is unchanged (duplicate / late arrival).
//   - If seq > covered_seq + 1, a gap exists; the watermark stays put until the
//     gap is filled. The batch is still persisted (for visibility), but the
//     returned covered_seq lets the caller detect the gap.
//
// customerID is needed only when creating the watermark row for the first time.
// Returns the (possibly unchanged) covered_seq after the operation.
func (t *txAdapter) AdvanceWatermark(ctx context.Context, namespace, subjectID string, seq int64) (int64, error) {
	existing, err := t.db.AIUsageWatermark.Query().
		Where(
			aiusagewatermark.Namespace(namespace),
			aiusagewatermark.SubjectID(subjectID),
		).
		ForUpdate().
		First(ctx)
	if err != nil {
		if !entdb.IsNotFound(err) {
			return 0, fmt.Errorf("query watermark: %w", err)
		}

		// No row yet — try to insert. A concurrent insert will fail with a
		// constraint error on the unique (namespace, subject_id) index; in
		// that case we re-query and proceed to the update path.
		newCovered := int64(0)
		if seq == 1 {
			newCovered = 1
		}

		_, createErr := t.db.AIUsageWatermark.Create().
			SetNamespace(namespace).
			SetSubjectID(subjectID).
			SetCustomerID(t.customerID).
			SetCoveredSeq(newCovered).
			Save(ctx)
		if createErr == nil {
			if newCovered == 1 {
				return t.catchUpWatermark(ctx, namespace, subjectID, 1)
			}
			return newCovered, nil
		}
		if !entdb.IsConstraintError(createErr) {
			return 0, fmt.Errorf("create watermark: %w", createErr)
		}

		// Lost the race — re-query with lock.
		existing, err = t.db.AIUsageWatermark.Query().
			Where(
				aiusagewatermark.Namespace(namespace),
				aiusagewatermark.SubjectID(subjectID),
			).
			ForUpdate().
			First(ctx)
		if err != nil {
			return 0, fmt.Errorf("re-query watermark after conflict: %w", err)
		}
	}

	current := existing.CoveredSeq

	if seq != current+1 {
		return current, nil
	}

	if _, uerr := t.db.AIUsageWatermark.UpdateOne(existing).
		SetCoveredSeq(seq).
		Save(ctx); uerr != nil {
		return 0, fmt.Errorf("advance watermark to %d: %w", seq, uerr)
	}

	return t.catchUpWatermark(ctx, namespace, subjectID, seq)
}

// catchUpWatermark scans for stored batches with contiguous seqs above the
// current covered_seq and advances the watermark as far as possible.
func (t *txAdapter) catchUpWatermark(ctx context.Context, namespace, subjectID string, current int64) (int64, error) {
	covered := current

	for {
		exists, err := t.db.AIUsageBatch.Query().
			Where(
				aiusagebatch.Namespace(namespace),
				aiusagebatch.SubjectID(subjectID),
				aiusagebatch.TenantSeq(covered+1),
			).
			Exist(ctx)
		if err != nil {
			return covered, fmt.Errorf("check next batch seq %d: %w", covered+1, err)
		}
		if !exists {
			break
		}
		covered++
	}

	if covered > current {
		_, err := t.db.AIUsageWatermark.Update().
			Where(
				aiusagewatermark.Namespace(namespace),
				aiusagewatermark.SubjectID(subjectID),
			).
			SetCoveredSeq(covered).
			Save(ctx)
		if err != nil {
			return current, fmt.Errorf("catch-up watermark to %d: %w", covered, err)
		}
	}

	return covered, nil
}
