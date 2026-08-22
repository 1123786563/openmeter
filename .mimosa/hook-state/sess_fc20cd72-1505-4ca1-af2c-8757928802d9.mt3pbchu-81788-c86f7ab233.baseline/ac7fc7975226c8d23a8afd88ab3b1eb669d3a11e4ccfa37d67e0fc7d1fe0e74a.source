package refund

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRetiredAIUsageSnapshotSurfaceIsRemoved(t *testing.T) {
	_, hasSnapshots := reflect.TypeOf(Config{}).FieldByName("Snapshots")
	require.False(t, hasSnapshots, "refund Config must not expose Runtime Authorization snapshots")

	_, hasSnapshotVersion := reflect.TypeOf(RefundRequest{}).FieldByName("SnapshotVersion")
	require.False(t, hasSnapshotVersion, "refund requests must not persist Runtime Authorization snapshot versions")
}
