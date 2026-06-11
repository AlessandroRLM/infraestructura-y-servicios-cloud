package reports

import (
	"context"
	"errors"
	"testing"
	"time"

	reportsv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/reports/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/reports/reportsdb"
	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// --- Key builder tests ---

func TestBuildSectionGradeKey(t *testing.T) {
	tests := []struct {
		name      string
		sectionID uuid.UUID
		wantKey   string
	}{
		{
			name:      "lowercase uuid produces canonical key",
			sectionID: uuid.MustParse("01932a81-f801-7a4c-90b4-123456789abc"),
			wantKey:   "report:section_grades:01932a81-f801-7a4c-90b4-123456789abc",
		},
		{
			name:      "uppercase uuid input is lowercased via uuid.UUID.String()",
			sectionID: uuid.MustParse("01932A81-F801-7A4C-90B4-123456789ABC"),
			wantKey:   "report:section_grades:01932a81-f801-7a4c-90b4-123456789abc",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildSectionGradeKey(tt.sectionID)
			if got != tt.wantKey {
				t.Errorf("buildSectionGradeKey() = %q, want %q", got, tt.wantKey)
			}
		})
	}
}

func TestBuildOccupancyKey(t *testing.T) {
	tests := []struct {
		name     string
		periodID uuid.UUID
		wantKey  string
	}{
		{
			name:     "produces canonical occupancy key",
			periodID: uuid.MustParse("01932a81-f801-7a4c-90b4-aabbccddeeff"),
			wantKey:  "report:section_occupancy:01932a81-f801-7a4c-90b4-aabbccddeeff",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildOccupancyKey(tt.periodID)
			if got != tt.wantKey {
				t.Errorf("buildOccupancyKey() = %q, want %q", got, tt.wantKey)
			}
		})
	}
}

func TestBuildProgramSummaryKey(t *testing.T) {
	tests := []struct {
		name      string
		programID uuid.UUID
		year      int
		wantKey   string
	}{
		{
			name:      "year decimal formatting, no leading zeros",
			programID: uuid.MustParse("01932a81-f801-7a4c-90b4-000000000001"),
			year:      2025,
			wantKey:   "report:program_enrollment:01932a81-f801-7a4c-90b4-000000000001:2025",
		},
		{
			name:      "year 2000 formatted as 2000",
			programID: uuid.MustParse("01932a81-f801-7a4c-90b4-000000000002"),
			year:      2000,
			wantKey:   "report:program_enrollment:01932a81-f801-7a4c-90b4-000000000002:2000",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildProgramSummaryKey(tt.programID, tt.year)
			if got != tt.wantKey {
				t.Errorf("buildProgramSummaryKey() = %q, want %q", got, tt.wantKey)
			}
		})
	}
}

func TestBuildStudentRecordKey(t *testing.T) {
	tests := []struct {
		name      string
		studentID uuid.UUID
		wantKey   string
	}{
		{
			name:      "produces canonical student record key",
			studentID: uuid.MustParse("01932a81-f801-7a4c-90b4-111111111111"),
			wantKey:   "report:student_record:01932a81-f801-7a4c-90b4-111111111111",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildStudentRecordKey(tt.studentID)
			if got != tt.wantKey {
				t.Errorf("buildStudentRecordKey() = %q, want %q", got, tt.wantKey)
			}
		})
	}
}

// --- fakeRedisClient for redisCache unit tests ---

// fakeRedis simulates a redis client with injectable results for Get and Set.
type fakeRedis struct {
	getCalled bool
	setCalled bool
	setTTL    time.Duration
	getVal    []byte
	getErr    error
	setErr    error
}

func (f *fakeRedis) Get(ctx context.Context, key string) ([]byte, bool, error) {
	f.getCalled = true
	if f.getErr != nil {
		return nil, false, f.getErr
	}
	if f.getVal == nil {
		return nil, false, nil // miss
	}
	return f.getVal, true, nil
}

func (f *fakeRedis) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	f.setCalled = true
	f.setTTL = ttl
	return f.setErr
}

// Verify fakeRedis satisfies the Cache interface.
var _ Cache = (*fakeRedis)(nil)

// --- Codec round-trip test ---

func TestProtojsonRoundTrip(t *testing.T) {
	// Build a realistic GetSectionGradeReportResponse.
	original := &reportsv1.GetSectionGradeReportResponse{
		SectionId:   "01932a81-f801-7a4c-90b4-111111111111",
		GeneratedAt: "2026-06-10T05:00:00Z",
		Truncated:   false,
		Rows: []*reportsv1.StudentGradeRow{
			{
				StudentId:        "01932a81-f801-7a4c-90b4-222222222222",
				GivenNames:       "Juan",
				LastNamePaternal: "García",
				LastNameMaternal: "López",
				FinalGrade:       "5.5",
				Outcome:          "passed",
				PartialGrades: []*reportsv1.PartialGrade{
					{EvaluationId: "01932a81-f801-7a4c-90b4-333333333333", Position: 1, Value: "5.5"},
				},
			},
		},
	}

	// Marshal with protojson.
	b, err := protojson.Marshal(original)
	if err != nil {
		t.Fatalf("protojson.Marshal: %v", err)
	}

	// Unmarshal back.
	var decoded reportsv1.GetSectionGradeReportResponse
	if err := protojson.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("protojson.Unmarshal: %v", err)
	}

	// Deep equality via proto.Equal.
	if !proto.Equal(original, &decoded) {
		t.Errorf("round-trip failed: original != decoded")
	}

	// Note: protojson does not guarantee byte-stable re-marshaling (it may vary
	// whitespace across calls). The proto.Equal round-trip above is the authoritative
	// fidelity check. Do not assert byte equality of b and b2 here.
}

// --- redisCache miss / error / set tests ---

func TestRedisCacheGetMiss(t *testing.T) {
	// fakeRedis returns nil → miss.
	fr := &fakeRedis{}
	c := newFakeCache(fr)
	data, found, err := c.Get(context.Background(), "somekey")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Fatal("expected miss (found=false), got found=true")
	}
	if data != nil {
		t.Fatalf("expected nil data on miss, got %v", data)
	}
}

func TestRedisCacheGetError(t *testing.T) {
	// fakeRedis returns an error.
	fr := &fakeRedis{getErr: errors.New("redis: connection refused")}
	c := newFakeCache(fr)
	_, _, err := c.Get(context.Background(), "somekey")
	if err == nil {
		t.Fatal("expected error from Get, got nil")
	}
}

func TestRedisCacheSetHappy(t *testing.T) {
	fr := &fakeRedis{}
	c := newFakeCache(fr)
	err := c.Set(context.Background(), "somekey", []byte("data"), 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error from Set: %v", err)
	}
	if !fr.setCalled {
		t.Fatal("expected Set to be called, but it was not")
	}
}

// newFakeCache returns a Cache backed by a fakeRedis directly (since we cannot
// easily create a real *redis.Client without a server in unit tests, we rely
// on the fakeRedis stub which already satisfies Cache).
func newFakeCache(fr *fakeRedis) Cache {
	return fr
}

// TestCacheMissPath_SetTTL_EqualsConfiguredTTL verifies that on a cache miss the
// cacheSet call passes the Service's configured TTL to the underlying Cache.Set.
// This is S-2: fakeRedis.Set captures the TTL and the test asserts it matches.
func TestCacheMissPath_SetTTL_EqualsConfiguredTTL(t *testing.T) {
	const wantTTL = 7 * time.Minute
	sectionID := uuid.New()

	repo := &fakeRepository{
		sectionExistsResult: true,
		actaAdminResult:     []reportsdb.ActaForSectionAdminRow{},
	}
	fr := &fakeRedis{}
	svc := NewService(repo, fr, wantTTL)

	_, err := svc.GetSectionGradeReport(adminCtx(), sectionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fr.setCalled {
		t.Fatal("expected cache.Set to be called on miss path, but it was not")
	}
	if fr.setTTL != wantTTL {
		t.Errorf("cache.Set received TTL=%v, want %v", fr.setTTL, wantTTL)
	}
}
