package integration_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	reportsv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/reports/v1"
)

// TestReports_Cache_SectionGradeReport_MissPopulatesCache verifies that a cache miss
// results in the key being populated in Redis after a successful DB fetch.
func TestReports_Cache_SectionGradeReport_MissPopulatesCache(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "reports-cache-miss@reports.test", "admin")

	// Seed a real section.
	_, courseID, programCleanup := seedProgramWithCourse(t)
	t.Cleanup(programCleanup)
	periodID, _, periodCleanup := seedAcademicPeriodWithWindow(t, false, false)
	t.Cleanup(periodCleanup)
	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 30)
	t.Cleanup(sectionCleanup)

	// Delete any pre-existing cache key to ensure a miss.
	cacheKey := "report:section_grades:" + sectionID
	testRedisClient.Del(ctx, cacheKey)

	// Make the request.
	client := newReportsClient(nil)
	req := connect.NewRequest(&reportsv1.GetSectionGradeReportRequest{
		SectionId: sectionID,
	})
	req.Header().Set("Cookie", "sid="+adminSID)

	_, err := client.GetSectionGradeReport(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Key must now exist in Redis.
	exists := testRedisClient.Exists(ctx, cacheKey).Val()
	if exists != 1 {
		t.Errorf("expected cache key %q to exist after miss+DB, got exists=%d", cacheKey, exists)
	}
}

// TestReports_Cache_SectionGradeReport_HitDoesNotCallDB verifies that a subsequent
// request for the same section is served from cache (no DB query executed).
// We verify this by mutating the DB row after the first fetch and confirming the
// second response still returns the cached (original) data.
func TestReports_Cache_SectionGradeReport_HitSkipsDB(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "reports-cache-hit@reports.test", "admin")

	_, courseID, programCleanup := seedProgramWithCourse(t)
	t.Cleanup(programCleanup)
	periodID, _, periodCleanup := seedAcademicPeriodWithWindow(t, false, false)
	t.Cleanup(periodCleanup)
	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 30)
	t.Cleanup(sectionCleanup)

	cacheKey := "report:section_grades:" + sectionID
	testRedisClient.Del(ctx, cacheKey)

	client := newReportsClient(nil)
	req1 := connect.NewRequest(&reportsv1.GetSectionGradeReportRequest{SectionId: sectionID})
	req1.Header().Set("Cookie", "sid="+adminSID)

	// First request: cache miss → populates cache.
	resp1, err := client.GetSectionGradeReport(ctx, req1)
	if err != nil {
		t.Fatalf("first request unexpected error: %v", err)
	}

	// Verify key exists.
	if testRedisClient.Exists(ctx, cacheKey).Val() != 1 {
		t.Fatalf("cache key not set after first request")
	}

	// Mutate the DB so any second DB hit would differ (section capacity changed to detect cache skip).
	_, _ = pgxPool.Exec(ctx, `UPDATE sections SET capacity = capacity + 999 WHERE id = $1::uuid`, sectionID)

	// Second request: must be served from cache (DB mutation invisible).
	req2 := connect.NewRequest(&reportsv1.GetSectionGradeReportRequest{SectionId: sectionID})
	req2.Header().Set("Cookie", "sid="+adminSID)
	resp2, err := client.GetSectionGradeReport(ctx, req2)
	if err != nil {
		t.Fatalf("second request unexpected error: %v", err)
	}
	if resp2.Msg.SectionId != sectionID {
		t.Errorf("second response SectionId=%s, want %s", resp2.Msg.SectionId, sectionID)
	}
	// generated_at must be identical — cache hit returns the same frozen payload.
	if resp2.Msg.GetGeneratedAt() != resp1.Msg.GetGeneratedAt() {
		t.Errorf("cache hit: GeneratedAt changed: first=%q second=%q", resp1.Msg.GetGeneratedAt(), resp2.Msg.GetGeneratedAt())
	}
	// Full payload identity: proto.Equal catches any field divergence.
	if !proto.Equal(resp1.Msg, resp2.Msg) {
		t.Error("cache hit: proto.Equal(resp1, resp2) = false — cache did not return identical payload")
	}
}

// TestReports_Cache_TTL verifies that the Redis key has a TTL close to ReportsCacheTTL
// after a cache population.
func TestReports_Cache_TTL(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "reports-cache-ttl@reports.test", "admin")

	_, courseID, programCleanup := seedProgramWithCourse(t)
	t.Cleanup(programCleanup)
	periodID, _, periodCleanup := seedAcademicPeriodWithWindow(t, false, false)
	t.Cleanup(periodCleanup)
	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 30)
	t.Cleanup(sectionCleanup)

	cacheKey := "report:section_grades:" + sectionID
	testRedisClient.Del(ctx, cacheKey)

	client := newReportsClient(nil)
	req := connect.NewRequest(&reportsv1.GetSectionGradeReportRequest{SectionId: sectionID})
	req.Header().Set("Cookie", "sid="+adminSID)

	_, err := client.GetSectionGradeReport(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ttl := testRedisClient.TTL(ctx, cacheKey).Val()
	if ttl <= 0 {
		t.Fatalf("cache key has no TTL (key missing or no expiry), TTL=%v", ttl)
	}

	wantTTL := sharedCfg.ReportsCacheTTL
	tolerance := 5 * time.Second
	if ttl < wantTTL-tolerance {
		t.Errorf("cache TTL = %v, want close to %v (within %v)", ttl, wantTTL, tolerance)
	}
}

// TestReports_Cache_UUIDCanonical verifies that cache keys use lowercase UUIDs.
// Redis keys are case-sensitive; all UUIDs must be normalized to lowercase.
func TestReports_Cache_UUIDCanonical(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "reports-cache-uuid@reports.test", "admin")

	_, courseID, programCleanup := seedProgramWithCourse(t)
	t.Cleanup(programCleanup)
	periodID, _, periodCleanup := seedAcademicPeriodWithWindow(t, false, false)
	t.Cleanup(periodCleanup)
	sectionID, sectionCleanup := seedSection(t, courseID, periodID, 30)
	t.Cleanup(sectionCleanup)

	// sectionID from the DB is already lowercase UUID.
	// The key builder uses uuid.UUID.String() which always returns lowercase.
	expectedKey := "report:section_grades:" + sectionID
	testRedisClient.Del(ctx, expectedKey)

	client := newReportsClient(nil)
	req := connect.NewRequest(&reportsv1.GetSectionGradeReportRequest{SectionId: sectionID})
	req.Header().Set("Cookie", "sid="+adminSID)

	_, err := client.GetSectionGradeReport(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if testRedisClient.Exists(ctx, expectedKey).Val() != 1 {
		t.Errorf("canonical (lowercase) cache key %q does not exist after request", expectedKey)
	}
}

// TestReports_Cache_OccupancyReport_MissPopulatesCache verifies the occupancy report
// also populates Redis on a cache miss.
func TestReports_Cache_OccupancyReport_MissPopulatesCache(t *testing.T) {
	ctx := context.Background()
	_, adminSID := seedUserWithSession(t, "reports-cache-occupancy@reports.test", "admin")

	_, _, programCleanup := seedProgramWithCourse(t)
	t.Cleanup(programCleanup)
	periodID, _, periodCleanup := seedAcademicPeriodWithWindow(t, false, false)
	t.Cleanup(periodCleanup)

	cacheKey := "report:section_occupancy:" + periodID
	testRedisClient.Del(ctx, cacheKey)

	client := newReportsClient(nil)
	req := connect.NewRequest(&reportsv1.GetSectionOccupancyReportRequest{
		AcademicPeriodId: periodID,
	})
	req.Header().Set("Cookie", "sid="+adminSID)

	_, err := client.GetSectionOccupancyReport(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if testRedisClient.Exists(ctx, cacheKey).Val() != 1 {
		t.Errorf("occupancy cache key %q not set after miss", cacheKey)
	}
}
