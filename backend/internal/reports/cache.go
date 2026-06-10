package reports

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Cache is the consumer-side caching seam for the reports slice.
// Implementations must treat a key-miss as (nil, false, nil) — never an error.
// Redis errors from Get are treated as a cache miss (fail-open); Redis errors from
// Set are swallowed silently after logging.
//
// proto messages are serialized with protojson (not encoding/json) because generated
// structs carry unexported fields that encoding/json cannot round-trip correctly.
type Cache interface {
	// Get retrieves the cached bytes for key. Returns (nil, false, nil) on a cache miss.
	// Returns (nil, false, err) on a Redis error (caller should treat as miss and log).
	Get(ctx context.Context, key string) (data []byte, found bool, err error)
	// Set stores data under key with the given TTL. A non-nil error should be logged and swallowed.
	Set(ctx context.Context, key string, data []byte, ttl time.Duration) error
}

// redisCache implements Cache over a *redis.Client.
type redisCache struct {
	client *redis.Client
}

// Compile-time proof that *redisCache satisfies Cache.
var _ Cache = (*redisCache)(nil)

// NewRedisCache constructs a Cache backed by the provided Redis client.
// The ttl parameter is accepted for API symmetry but each Set call receives its own ttl.
func NewRedisCache(client *redis.Client) Cache {
	return &redisCache{client: client}
}

// Get retrieves bytes stored at key.
// redis.Nil (key not found) is mapped to (nil, false, nil) — a cache miss, not an error.
func (c *redisCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	val, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, err
	}
	return val, true, nil
}

// Set stores data at key with the given TTL.
func (c *redisCache) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	return c.client.Set(ctx, key, data, ttl).Err()
}

// --- Cache key builders ---

// buildSectionGradeKey returns the canonical cache key for a section grade report.
// Format: report:section_grades:<section_uuid_lowercase>
func buildSectionGradeKey(sectionID uuid.UUID) string {
	return fmt.Sprintf("report:section_grades:%s", sectionID.String())
}

// buildOccupancyKey returns the canonical cache key for a section occupancy report.
// Format: report:section_occupancy:<period_uuid_lowercase>
func buildOccupancyKey(periodID uuid.UUID) string {
	return fmt.Sprintf("report:section_occupancy:%s", periodID.String())
}

// buildProgramSummaryKey returns the canonical cache key for a program enrollment summary.
// Format: report:program_enrollment:<program_uuid_lowercase>:<year_decimal>
func buildProgramSummaryKey(programID uuid.UUID, year int) string {
	return fmt.Sprintf("report:program_enrollment:%s:%d", programID.String(), year)
}

// buildStudentRecordKey returns the canonical cache key for a student academic record.
// Format: report:student_record:<student_uuid_lowercase>
func buildStudentRecordKey(studentID uuid.UUID) string {
	return fmt.Sprintf("report:student_record:%s", studentID.String())
}

// logCacheGetError logs a Redis Get failure at Warn level (fail-open: not returned to caller).
func logCacheGetError(ctx context.Context, key string, err error) {
	slog.WarnContext(ctx, "reports cache get failed", "key", key, "err", err)
}

// logCacheSetError logs a Redis Set failure at Warn level (best-effort: not returned to caller).
func logCacheSetError(ctx context.Context, key string, err error) {
	slog.WarnContext(ctx, "reports cache set failed", "key", key, "err", err)
}

// logInternalError logs an unexpected internal error at Error level.
func logInternalError(ctx context.Context, err error) {
	slog.ErrorContext(ctx, "reports: internal error", "err", err)
}
