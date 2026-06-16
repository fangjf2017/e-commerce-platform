package evaluator

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/spaolacci/murmur3"
)

const bucketCount = 10000

// ComputeBucket returns a deterministic bucket [0, 9999] for (flagKey, entityID, salt).
// The same inputs always produce the same bucket, ensuring sticky assignment.
func ComputeBucket(flagKey, entityID string, salt uuid.UUID) int {
	input := fmt.Sprintf("%s:%s:%s", flagKey, entityID, salt.String())
	h := murmur3.Sum32([]byte(input))
	return int(h % bucketCount)
}

// IsInRollout returns true if bucket falls within [0, rolloutPct*100).
// rolloutPct is 0-100; bucket is 0-9999.
func IsInRollout(bucket, rolloutPct int) bool {
	if rolloutPct <= 0 {
		return false
	}
	if rolloutPct >= 100 {
		return true
	}
	return bucket < rolloutPct*100
}

// SelectVariant returns the variant key whose allocation range contains the bucket.
// allocations must be ordered and non-overlapping.
func SelectVariant(bucket int, allocations []variantRange) string {
	for _, a := range allocations {
		if bucket >= a.from && bucket < a.to {
			return a.variantKey
		}
	}
	return ""
}

type variantRange struct {
	variantKey string
	from       int
	to         int
}
