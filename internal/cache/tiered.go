package cache

import (
	"context"

	"github.com/ecommerce/feature-management/internal/domain"
	"github.com/ecommerce/feature-management/internal/observability"
	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
)

// FlagLoader is called on L1+L2 miss to load from DB.
type FlagLoader func(ctx context.Context, appID, envID uuid.UUID, key string) (*domain.FeatureFlag, error)

// SegmentLoader is called on L1+L2 miss to load from DB.
type SegmentLoader func(ctx context.Context, appID, segID uuid.UUID) (*domain.Segment, error)

type TieredCache struct {
	l1      *L1Cache
	l2      *L2Cache
	metrics *observability.Metrics
	sflag   singleflight.Group
	sseg    singleflight.Group
}

func NewTieredCache(l1 *L1Cache, l2 *L2Cache, metrics *observability.Metrics) *TieredCache {
	return &TieredCache{l1: l1, l2: l2, metrics: metrics}
}

// GetFlag implements L1 -> L2 -> DB with singleflight coalescing.
// Returns the flag, the cache layer it was found in ("l1", "l2", or "db"), and any error.
func (t *TieredCache) GetFlag(ctx context.Context, appID, envID uuid.UUID, flagKey string, loader FlagLoader) (*domain.FeatureFlag, string, error) {
	// L1
	if flag, ok := t.l1.GetFlag(ctx, appID, envID, flagKey); ok {
		if t.metrics != nil {
			t.metrics.CacheHitsTotal.WithLabelValues("l1").Inc()
		}
		return flag, "l1", nil
	}
	if t.metrics != nil {
		t.metrics.CacheMissesTotal.WithLabelValues("l1").Inc()
	}

	// L2
	if flag, ok := t.l2.GetFlag(ctx, appID, envID, flagKey); ok {
		_ = t.l1.SetFlag(ctx, flag)
		if t.metrics != nil {
			t.metrics.CacheHitsTotal.WithLabelValues("l2").Inc()
		}
		return flag, "l2", nil
	}
	if t.metrics != nil {
		t.metrics.CacheMissesTotal.WithLabelValues("l2").Inc()
	}

	// DB (singleflight coalesces concurrent requests for the same key)
	sfKey := FlagKey(appID, envID, flagKey)
	v, err, _ := t.sflag.Do(sfKey, func() (interface{}, error) {
		return loader(ctx, appID, envID, flagKey)
	})
	if err != nil {
		return nil, "db", err
	}
	flag := v.(*domain.FeatureFlag)
	_ = t.l2.SetFlag(ctx, flag)
	_ = t.l1.SetFlag(ctx, flag)
	return flag, "db", nil
}

// SetFlag writes through to both L2 and L1.
func (t *TieredCache) SetFlag(ctx context.Context, flag *domain.FeatureFlag) {
	_ = t.l2.SetFlag(ctx, flag)
	_ = t.l1.SetFlag(ctx, flag)
}

// DeleteFlag evicts from both L1 and L2.
func (t *TieredCache) DeleteFlag(ctx context.Context, appID, envID uuid.UUID, flagKey string) {
	t.l1.DeleteFlag(ctx, appID, envID, flagKey)
	t.l2.DeleteFlag(ctx, appID, envID, flagKey)
	if t.metrics != nil {
		t.metrics.CacheInvalidationsTotal.WithLabelValues("flag_delete").Inc()
	}
}

// GetSegment implements L1 -> L2 -> DB with singleflight coalescing.
func (t *TieredCache) GetSegment(ctx context.Context, appID, segID uuid.UUID, loader SegmentLoader) (*domain.Segment, error) {
	if seg, ok := t.l1.GetSegment(ctx, appID, segID); ok {
		if t.metrics != nil {
			t.metrics.CacheHitsTotal.WithLabelValues("l1").Inc()
		}
		return seg, nil
	}
	if t.metrics != nil {
		t.metrics.CacheMissesTotal.WithLabelValues("l1").Inc()
	}
	if seg, ok := t.l2.GetSegment(ctx, appID, segID); ok {
		_ = t.l1.SetSegment(ctx, seg)
		if t.metrics != nil {
			t.metrics.CacheHitsTotal.WithLabelValues("l2").Inc()
		}
		return seg, nil
	}
	if t.metrics != nil {
		t.metrics.CacheMissesTotal.WithLabelValues("l2").Inc()
	}
	sfKey := SegmentKey(appID, segID)
	v, err, _ := t.sseg.Do(sfKey, func() (interface{}, error) {
		return loader(ctx, appID, segID)
	})
	if err != nil {
		return nil, err
	}
	seg := v.(*domain.Segment)
	_ = t.l2.SetSegment(ctx, seg)
	_ = t.l1.SetSegment(ctx, seg)
	return seg, nil
}

// SetSegment writes through to both L2 and L1.
func (t *TieredCache) SetSegment(ctx context.Context, seg *domain.Segment) {
	_ = t.l2.SetSegment(ctx, seg)
	_ = t.l1.SetSegment(ctx, seg)
}

// DeleteSegment evicts from both L1 and L2.
func (t *TieredCache) DeleteSegment(ctx context.Context, appID, segID uuid.UUID) {
	t.l1.DeleteSegment(ctx, appID, segID)
	t.l2.DeleteSegment(ctx, appID, segID)
	if t.metrics != nil {
		t.metrics.CacheInvalidationsTotal.WithLabelValues("segment_delete").Inc()
	}
}

// L2 exposes the underlying L2 cache for direct access when needed.
func (t *TieredCache) L2() *L2Cache { return t.l2 }
