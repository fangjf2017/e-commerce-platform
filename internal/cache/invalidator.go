package cache

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// InvalidationMessage is the JSON payload published on invalidation channels.
type InvalidationMessage struct {
	FlagKey   string `json:"flag_key"`
	Version   int64  `json:"version"`
	Action    string `json:"action"`
	AppID     string `json:"app_id"`
	EnvID     string `json:"env_id"`
	SegmentID string `json:"segment_id,omitempty"`
}

// Invalidator subscribes to Redis pub/sub invalidation channels and evicts
// stale entries from the tiered cache.
type Invalidator struct {
	client *redis.Client
	cache  *TieredCache
	logger *zap.Logger
}

func NewInvalidator(client *redis.Client, cache *TieredCache, logger *zap.Logger) *Invalidator {
	return &Invalidator{client: client, cache: cache, logger: logger}
}

// Run subscribes to ff:invalidate:* and seg:invalidate:* channels.
// Call this in a goroutine; it blocks until ctx is cancelled.
func (inv *Invalidator) Run(ctx context.Context) {
	pubsub := inv.client.PSubscribe(ctx, "ff:invalidate:*", "seg:invalidate:*")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			inv.handleMessage(ctx, msg)
		}
	}
}

func (inv *Invalidator) handleMessage(ctx context.Context, msg *redis.Message) {
	var payload InvalidationMessage
	if err := json.Unmarshal([]byte(msg.Payload), &payload); err != nil {
		inv.logger.Warn("invalid invalidation message", zap.Error(err))
		return
	}

	if strings.HasPrefix(msg.Channel, "ff:invalidate:") {
		appID, err1 := uuid.Parse(payload.AppID)
		envID, err2 := uuid.Parse(payload.EnvID)
		if err1 != nil || err2 != nil {
			inv.logger.Warn("invalid UUIDs in flag invalidation message",
				zap.String("app_id", payload.AppID),
				zap.String("env_id", payload.EnvID))
			return
		}
		inv.cache.DeleteFlag(ctx, appID, envID, payload.FlagKey)
		inv.logger.Debug("invalidated flag cache",
			zap.String("flag_key", payload.FlagKey),
			zap.String("action", payload.Action))
	} else if strings.HasPrefix(msg.Channel, "seg:invalidate:") {
		if payload.SegmentID == "" {
			inv.logger.Warn("missing segment_id in segment invalidation message")
			return
		}
		appID, err1 := uuid.Parse(payload.AppID)
		segID, err2 := uuid.Parse(payload.SegmentID)
		if err1 != nil || err2 != nil {
			inv.logger.Warn("invalid UUIDs in segment invalidation message",
				zap.String("app_id", payload.AppID),
				zap.String("segment_id", payload.SegmentID))
			return
		}
		inv.cache.DeleteSegment(ctx, appID, segID)
		inv.logger.Debug("invalidated segment cache", zap.String("segment_id", payload.SegmentID))
	}
}

// PublishFlagInvalidation publishes a flag invalidation event to Redis.
func (inv *Invalidator) PublishFlagInvalidation(ctx context.Context, appID, envID uuid.UUID, flagKey, action string, version int64) error {
	payload := InvalidationMessage{
		FlagKey: flagKey,
		Version: version,
		Action:  action,
		AppID:   appID.String(),
		EnvID:   envID.String(),
	}
	data, _ := json.Marshal(payload)
	channel := InvalidationChannel(appID, envID)
	return inv.client.Publish(ctx, channel, data).Err()
}

// PublishSegmentInvalidation publishes a segment invalidation event.
func (inv *Invalidator) PublishSegmentInvalidation(ctx context.Context, appID, segID uuid.UUID) error {
	payload := InvalidationMessage{
		AppID:     appID.String(),
		SegmentID: segID.String(),
	}
	data, _ := json.Marshal(payload)
	channel := SegmentInvalidationChannel(appID)
	return inv.client.Publish(ctx, channel, data).Err()
}
