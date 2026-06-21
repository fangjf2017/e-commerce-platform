package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/ecommerce/feature-management/internal/domain"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const l2FlagTTL = 5 * time.Minute
const l2SegmentTTL = 10 * time.Minute

type L2Cache struct {
	client *redis.Client
}

func NewL2Cache(client *redis.Client) *L2Cache {
	return &L2Cache{client: client}
}

func (c *L2Cache) GetFlag(ctx context.Context, appID, envID uuid.UUID, flagKey string) (*domain.FeatureFlag, bool) {
	key := FlagKey(appID, envID, flagKey)
	val, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false
		}
		return nil, false
	}
	var flag domain.FeatureFlag
	if err := json.Unmarshal(val, &flag); err != nil {
		return nil, false
	}
	return &flag, true
}

func (c *L2Cache) SetFlag(ctx context.Context, flag *domain.FeatureFlag) error {
	key := FlagKey(flag.ApplicationID, flag.EnvironmentID, flag.Key)
	data, err := json.Marshal(flag)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, l2FlagTTL).Err()
}

func (c *L2Cache) DeleteFlag(ctx context.Context, appID, envID uuid.UUID, flagKey string) {
	c.client.Del(ctx, FlagKey(appID, envID, flagKey))
}

func (c *L2Cache) GetSegment(ctx context.Context, appID, segID uuid.UUID) (*domain.Segment, bool) {
	key := SegmentKey(appID, segID)
	val, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}
	var seg domain.Segment
	if err := json.Unmarshal(val, &seg); err != nil {
		return nil, false
	}
	return &seg, true
}

func (c *L2Cache) SetSegment(ctx context.Context, seg *domain.Segment) error {
	key := SegmentKey(seg.ApplicationID, seg.ID)
	data, err := json.Marshal(seg)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, l2SegmentTTL).Err()
}

func (c *L2Cache) DeleteSegment(ctx context.Context, appID, segID uuid.UUID) {
	c.client.Del(ctx, SegmentKey(appID, segID))
}

func (c *L2Cache) PublishInvalidation(ctx context.Context, channel string, payload []byte) error {
	return c.client.Publish(ctx, channel, payload).Err()
}

func (c *L2Cache) Client() *redis.Client {
	return c.client
}
