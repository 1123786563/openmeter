package commerce

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSourcePriority verifies the fixed consumption burn order: plan < gift
// < recharge < enterprise_receivable. Unknown sources get the lowest priority
// (100) so they are never consumed before known sources.
func TestSourcePriority(t *testing.T) {
	tests := []struct {
		source BucketSource
		want   int
	}{
		{BucketSourcePlan, 10},
		{BucketSourceGift, 20},
		{BucketSourceRecharge, 30},
		{BucketSourceEnterpriseReceivable, 40},
		{"unknown", 100},
		{"", 100},
	}
	for _, tc := range tests {
		t.Run(string(tc.source), func(t *testing.T) {
			assert.Equal(t, tc.want, SourcePriority(tc.source))
		})
	}
}

// TestSourcePriorityOrder confirms the burn ordering by comparing all pairs.
func TestSourcePriorityOrder(t *testing.T) {
	sources := []BucketSource{
		BucketSourcePlan,
		BucketSourceGift,
		BucketSourceRecharge,
		BucketSourceEnterpriseReceivable,
	}
	for i := 0; i < len(sources)-1; i++ {
		assert.Less(t, SourcePriority(sources[i]), SourcePriority(sources[i+1]),
			"%s must burn before %s", sources[i], sources[i+1])
	}
}
