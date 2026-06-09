package integration_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	section_enrollmentv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/section_enrollment/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/section_enrollment/v1/section_enrollmentv1connect"
)

// seAcademicPeriodYearCounter generates unique year values for academic period seeds.
// Starts at 3000 to avoid conflicts with other test helpers that use 2099.
var seAcademicPeriodYearCounter atomic.Int32

func init() {
	seAcademicPeriodYearCounter.Store(3000)
}

// newSectionEnrollmentClient returns a Connect SectionEnrollmentService client.
func newSectionEnrollmentClient(jar http.CookieJar) section_enrollmentv1connect.SectionEnrollmentServiceClient {
	return section_enrollmentv1connect.NewSectionEnrollmentServiceClient(&http.Client{Jar: jar}, baseURL)
}

// seedAcademicPeriodWithWindow inserts an academic_period with the given enrollment window.
// Pass windowOpen=true and the helper sets starts=now-1h/ends=now+1h.
// Pass windowOpen=false for starts=now+1h/ends=now+2h (future, closed).
// Pass nullWindow=true for NULL columns (fail-closed).
// Returns the period id UUID and a cleanup func.
func seedAcademicPeriodWithWindow(t *testing.T, windowOpen, nullWindow bool) (string, func()) {
	t.Helper()
	ctx := context.Background()

	// Use a unique year per call to avoid UNIQUE(year,term) conflicts across parallel tests.
	year := seAcademicPeriodYearCounter.Add(1)

	now := time.Now().UTC()
	var periodID uuid.UUID
	var err error

	if nullWindow {
		err = pgxPool.QueryRow(ctx,
			`INSERT INTO academic_periods (year, term, start_date, end_date)
			 VALUES ($1, $2, $3, $4) RETURNING id`,
			year, 1,
			now.Format("2006-01-02"),
			now.AddDate(0, 6, 0).Format("2006-01-02"),
		).Scan(&periodID)
	} else if windowOpen {
		err = pgxPool.QueryRow(ctx,
			`INSERT INTO academic_periods (year, term, start_date, end_date, enrollment_starts_at, enrollment_ends_at)
			 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
			year, 1,
			now.Format("2006-01-02"),
			now.AddDate(0, 6, 0).Format("2006-01-02"),
			now.Add(-time.Hour),
			now.Add(time.Hour),
		).Scan(&periodID)
	} else {
		// Window in the past — closed.
		err = pgxPool.QueryRow(ctx,
			`INSERT INTO academic_periods (year, term, start_date, end_date, enrollment_starts_at, enrollment_ends_at)
			 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
			year, 1,
			now.Format("2006-01-02"),
			now.AddDate(0, 6, 0).Format("2006-01-02"),
			now.Add(-2*time.Hour),
			now.Add(-time.Hour),
		).Scan(&periodID)
	}
	if err != nil {
		t.Fatalf("seedAcademicPeriodWithWindow: %v", err)
	}

	cleanup := func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM academic_periods WHERE id = $1`, periodID)
	}
	return periodID.String(), cleanup
}

// seedProgramWithCourse creates a program, a course, links them via program_courses,
// and returns (programID, courseID, cleanup).
func seedProgramWithCourse(t *testing.T) (string, string, func()) {
	t.Helper()
	ctx := context.Background()

	var programID, courseID uuid.UUID
	if err := pgxPool.QueryRow(ctx,
		`INSERT INTO programs (code, name) VALUES ($1, $2) RETURNING id`,
		"SE-PROG-"+uniqueSuffix(t), "SE Test Program",
	).Scan(&programID); err != nil {
		t.Fatalf("seedProgramWithCourse: insert program: %v", err)
	}

	if err := pgxPool.QueryRow(ctx,
		`INSERT INTO courses (code, name, credits) VALUES ($1, $2, $3) RETURNING id`,
		"SE-CRS-"+uniqueSuffix(t), "SE Test Course", 3,
	).Scan(&courseID); err != nil {
		_, _ = pgxPool.Exec(ctx, `DELETE FROM programs WHERE id = $1`, programID)
		t.Fatalf("seedProgramWithCourse: insert course: %v", err)
	}

	if _, err := pgxPool.Exec(ctx,
		`INSERT INTO program_courses (program_id, course_id) VALUES ($1, $2)`,
		programID, courseID,
	); err != nil {
		_, _ = pgxPool.Exec(ctx, `DELETE FROM courses WHERE id = $1`, courseID)
		_, _ = pgxPool.Exec(ctx, `DELETE FROM programs WHERE id = $1`, programID)
		t.Fatalf("seedProgramWithCourse: link course: %v", err)
	}

	cleanup := func() {
		c := context.Background()
		_, _ = pgxPool.Exec(c, `DELETE FROM program_courses WHERE program_id = $1 AND course_id = $2`, programID, courseID)
		_, _ = pgxPool.Exec(c, `DELETE FROM courses WHERE id = $1`, courseID)
		_, _ = pgxPool.Exec(c, `DELETE FROM programs WHERE id = $1`, programID)
	}
	return programID.String(), courseID.String(), cleanup
}

// seedSection creates a section for the given courseID and academicPeriodID with the given capacity.
// Returns the section UUID and a cleanup func.
func seedSection(t *testing.T, courseID, periodID string, capacity int32) (string, func()) {
	t.Helper()
	ctx := context.Background()

	var sectionID uuid.UUID
	if err := pgxPool.QueryRow(ctx,
		`INSERT INTO sections (course_id, academic_period_id, capacity) VALUES ($1, $2, $3) RETURNING id`,
		courseID, periodID, capacity,
	).Scan(&sectionID); err != nil {
		t.Fatalf("seedSection: %v", err)
	}

	cleanup := func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM sections WHERE id = $1`, sectionID)
	}
	return sectionID.String(), cleanup
}

// seedPaidEnrollment creates an enrollment for the given student+program and marks it paid.
// Returns the enrollment UUID and a cleanup func.
func seedPaidEnrollment(t *testing.T, studentID, programID string, year int32) (string, func()) {
	t.Helper()
	ctx := context.Background()

	var enrollmentID uuid.UUID
	if err := pgxPool.QueryRow(ctx,
		`INSERT INTO enrollments (student_id, program_id, year, status) VALUES ($1, $2, $3, 'paid') RETURNING id`,
		studentID, programID, year,
	).Scan(&enrollmentID); err != nil {
		t.Fatalf("seedPaidEnrollment: %v", err)
	}

	cleanup := func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM enrollments WHERE id = $1`, enrollmentID)
	}
	return enrollmentID.String(), cleanup
}

// seedPendingEnrollment creates an enrollment with status=pending (not paid).
func seedPendingEnrollment(t *testing.T, studentID, programID string, year int32) (string, func()) {
	t.Helper()
	ctx := context.Background()

	var enrollmentID uuid.UUID
	if err := pgxPool.QueryRow(ctx,
		`INSERT INTO enrollments (student_id, program_id, year, status) VALUES ($1, $2, $3, 'pending') RETURNING id`,
		studentID, programID, year,
	).Scan(&enrollmentID); err != nil {
		t.Fatalf("seedPendingEnrollment: %v", err)
	}

	cleanup := func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM enrollments WHERE id = $1`, enrollmentID)
	}
	return enrollmentID.String(), cleanup
}

// cleanupSectionEnrollment deletes a section_enrollment row by id string.
func cleanupSectionEnrollment(t *testing.T, id string) {
	t.Helper()
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM section_enrollments WHERE id = $1`, id)
	})
}

// cleanupAllSectionEnrollmentsForSection deletes all section_enrollment rows for a section.
func cleanupAllSectionEnrollmentsForSection(t *testing.T, sectionID string) {
	t.Helper()
	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM section_enrollments WHERE section_id = $1`, sectionID)
	})
}

// seEnrollAdmin calls admin EnrollSection and returns the proto.
func seEnrollAdmin(ctx context.Context, client section_enrollmentv1connect.SectionEnrollmentServiceClient, adminSID, enrollmentID, sectionID string) (*section_enrollmentv1.SectionEnrollment, error) {
	req := connect.NewRequest(&section_enrollmentv1.EnrollSectionRequest{
		EnrollmentId: enrollmentID,
		SectionId:    sectionID,
	})
	req.Header().Set("Cookie", "sid="+adminSID)
	resp, err := client.EnrollSection(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

// seEnrollOwn calls student EnrollOwnSection and returns the proto.
func seEnrollOwn(ctx context.Context, client section_enrollmentv1connect.SectionEnrollmentServiceClient, studentSID, sectionID string) (*section_enrollmentv1.SectionEnrollment, error) {
	req := connect.NewRequest(&section_enrollmentv1.EnrollOwnSectionRequest{
		SectionId: sectionID,
	})
	req.Header().Set("Cookie", "sid="+studentSID)
	resp, err := client.EnrollOwnSection(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

// activeSeatCount returns the current active (non-withdrawn, non-deleted) seat count for a section.
func activeSeatCount(t *testing.T, sectionID string) int {
	t.Helper()
	var n int
	if err := pgxPool.QueryRow(context.Background(),
		`SELECT count(*) FROM section_enrollments WHERE section_id = $1 AND status <> 'withdrawn' AND deleted_at IS NULL`,
		sectionID,
	).Scan(&n); err != nil {
		t.Fatalf("activeSeatCount: %v", err)
	}
	return n
}
