package migrate_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDropRefundSnapshotVersionMigration(t *testing.T) {
	runner{
		stops: stops{
			{
				version:   20260811000100,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					_, err := db.Exec(`ALTER TABLE refund_requests ADD COLUMN snapshot_version character varying`)
					require.NoError(t, err)
				},
			},
			{
				version:   20260811000200,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					var count int
					err := db.QueryRow(`
						SELECT count(*)
						FROM information_schema.columns
						WHERE table_schema = current_schema()
						  AND table_name = 'refund_requests'
						  AND column_name = 'snapshot_version'
					`).Scan(&count)
					require.NoError(t, err)
					require.Zero(t, count)
				},
			},
			{
				version:   20260811000200,
				direction: directionDown,
				action: func(t *testing.T, db *sql.DB) {
					var count int
					err := db.QueryRow(`
						SELECT count(*)
						FROM information_schema.columns
						WHERE table_schema = current_schema()
						  AND table_name = 'refund_requests'
						  AND column_name = 'snapshot_version'
					`).Scan(&count)
					require.NoError(t, err)
					require.Equal(t, 1, count)
				},
			},
		},
	}.Test(t)
}
