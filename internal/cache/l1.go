package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/ecommerce/feature-management/internal/domain"
	"github.com/google/uuid"
)

const l1TTL = 30 * time.Second

type L1Cache struct {
	cache *ristretto.Cache[string, []byte]
}

func NewL1Cache(c *ristretto.Cache[string, []byte]) *L1Cache {
	return &L1Cache{cache: c}
}

func (c *L1Cache) GetFlag(ctx context.Context, appID, envID uuid.UUID, flagKey string) (*domain.FeatureFlag, bool) {
	key := FlagKey(appID, envID, flagKey)
	val, ok := c.cache.Get(key)
	if !ok {
		return nil, false
	}
	var flag domain.FeatureFlag
	if err := json.Unmarshal(val, &flag); err != nil {
		return nil, false
	}
	return &flag, true
}

func (c *L1Cache) SetFlag(ctx context.Context, flag *domain.FeatureFlag) error {
	key := FlagKey(flag.ApplicationID, flag.EnvironmentID, flag.Key)
	data, err := json.Marshal(flag)
	if err != nil {
		return err
	}
	c.cache.SetWithTTL(key, data, int64(len(data)), l1TTL)
	return nil
}

func (c *L1Cache) DeleteFlag(ctx context.Context, appID, envID uuid.UUID, flagKey string) {
	c.cache.Del(FlagKey(appID, envID, flagKey))
}

func (c *L1Cache) GetSegment(ctx context.Context, appID, segID uuid.UUID) (*domain.Segment, bool) {
	key := SegmentKey(appID, segID)
	val, ok := c.cache.Get(key)
	if !ok {
		return nil, false
	}
	var seg domain.Segment
	if err := json.Unmarshal(val, &seg); err != nil {
		return nil, false
	}
	return &seg, true
}

func (c *L1Cache) SetSegment(ctx context.Context, seg *domain.Segment) error {
	key := SegmentKey(seg.ApplicationID, seg.ID)
	data, err := json.Marshal(seg)
	if err != nil {
		return err
	}
	c.cache.SetWithTTL(key, data, int64(len(data)), l1TTL)
	return nil
}

func (c *L1Cache) DeleteSegment(ctx context.Context, appID, segID uuid.UUID) {
	c.cache.Del(SegmentKey(appID, segID))
}
