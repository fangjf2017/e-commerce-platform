package cache

import (
	"fmt"

	"github.com/google/uuid"
)

func FlagKey(appID, envID uuid.UUID, flagKey string) string {
	return fmt.Sprintf("ff:%s:%s:%s", appID, envID, flagKey)
}

func AllFlagsKey(appID, envID uuid.UUID) string {
	return fmt.Sprintf("ff:%s:%s:*", appID, envID)
}

func SegmentKey(appID, segID uuid.UUID) string {
	return fmt.Sprintf("seg:%s:%s", appID, segID)
}

func InvalidationChannel(appID, envID uuid.UUID) string {
	return fmt.Sprintf("ff:invalidate:%s:%s", appID, envID)
}

func SegmentInvalidationChannel(appID uuid.UUID) string {
	return fmt.Sprintf("seg:invalidate:%s", appID)
}
