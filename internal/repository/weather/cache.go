package weather

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"weathertracker/internal/domain"
	"weathertracker/internal/repository"
)

// CachedProvider decorates a WeatherProvider with an in-memory TTL cache. It
// implements the graceful-degradation strategy from the architecture: when the
// upstream provider fails, the last known snapshot is returned instead of an
// error. The snapshot keeps its original FetchedAt, so the service layer can
// flag it as stale ("Устаревшие данные") rather than hiding the outage.
type CachedProvider struct {
	next repository.WeatherProvider
	ttl  time.Duration

	mu    sync.Mutex
	cache map[string]cacheEntry
	now   func() time.Time
}

type cacheEntry struct {
	snapshot  domain.WeatherSnapshot
	expiresAt time.Time
}

// NewCachedProvider wraps next with a TTL cache.
func NewCachedProvider(next repository.WeatherProvider, ttl time.Duration) *CachedProvider {
	return &CachedProvider{
		next:  next,
		ttl:   ttl,
		cache: make(map[string]cacheEntry),
		now:   time.Now,
	}
}

// Fetch returns a fresh cached snapshot when available, otherwise delegates to
// the upstream provider. On upstream failure a stale snapshot is served as a
// fallback; if nothing is cached the wrapped error is returned.
func (p *CachedProvider) Fetch(ctx context.Context, latitude, longitude float64, forecastDays int) (domain.WeatherSnapshot, error) {
	key := cacheKey(latitude, longitude, forecastDays)
	now := p.now()

	p.mu.Lock()
	entry, ok := p.cache[key]
	p.mu.Unlock()

	if ok && now.Before(entry.expiresAt) {
		return entry.snapshot, nil
	}

	snapshot, err := p.next.Fetch(ctx, latitude, longitude, forecastDays)
	if err != nil {
		if ok {
			return entry.snapshot, nil
		}
		return domain.WeatherSnapshot{}, fmt.Errorf("upstream weather unavailable and cache empty: %w", err)
	}

	p.mu.Lock()
	p.cache[key] = cacheEntry{snapshot: snapshot, expiresAt: now.Add(p.ttl)}
	p.mu.Unlock()
	return snapshot, nil
}

// Cache entries are keyed on a ~1.1 km grid because weather is effectively
// identical within that radius, which maximises cache hits across users.
func cacheKey(latitude, longitude float64, forecastDays int) string {
	return fmt.Sprintf("%.2f:%.2f:%d",
		math.Round(latitude*100)/100,
		math.Round(longitude*100)/100,
		forecastDays,
	)
}
