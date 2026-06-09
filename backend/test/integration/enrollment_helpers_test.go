package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// seedProgramWithQuota inserts a program and a program_quotas row for the given
// capacity and year. Returns the program UUID string and a cleanup func that
// deletes the quota then the program (FK-safe order).
func seedProgramWithQuota(t *testing.T, capacity int32, year int32) (string, func()) {
	t.Helper()
	ctx := context.Background()

	var programID uuid.UUID
	err := pgxPool.QueryRow(ctx,
		`INSERT INTO programs (code, name) VALUES ($1, $2) RETURNING id`,
		"PROG-"+uniqueSuffix(t), "Enrollment Test Program",
	).Scan(&programID)
	if err != nil {
		t.Fatalf("seedProgramWithQuota: insert program: %v", err)
	}

	_, err = pgxPool.Exec(ctx,
		`INSERT INTO program_quotas (program_id, year, capacity) VALUES ($1, $2, $3)`,
		programID, year, capacity,
	)
	if err != nil {
		// Roll back the program we just inserted.
		_, _ = pgxPool.Exec(ctx, `DELETE FROM programs WHERE id = $1`, programID)
		t.Fatalf("seedProgramWithQuota: insert quota: %v", err)
	}

	cleanup := func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM program_quotas WHERE program_id = $1 AND year = $2`, programID, year)
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM programs WHERE id = $1`, programID)
	}
	return programID.String(), cleanup
}

// seedBareProgram inserts a program with no quota row.
// Returns the program UUID string. Registers t.Cleanup to delete the row.
func seedBareProgram(t *testing.T) string {
	t.Helper()
	ctx := context.Background()

	var programID uuid.UUID
	err := pgxPool.QueryRow(ctx,
		`INSERT INTO programs (code, name) VALUES ($1, $2) RETURNING id`,
		"BARE-"+uniqueSuffix(t), "Bare Program (no quota)",
	).Scan(&programID)
	if err != nil {
		t.Fatalf("seedBareProgram: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM programs WHERE id = $1`, programID)
	})
	return programID.String()
}

// seedStudentProfile inserts a student_profiles row for the given user ID.
// Registers t.Cleanup to delete the row in FK-safe order.
func seedStudentProfile(t *testing.T, userID uuid.UUID, admissionYear int32) {
	t.Helper()
	ctx := context.Background()

	_, err := pgxPool.Exec(ctx,
		`INSERT INTO student_profiles (user_id, admission_year) VALUES ($1, $2) ON CONFLICT (user_id) DO NOTHING`,
		userID, admissionYear,
	)
	if err != nil {
		t.Fatalf("seedStudentProfile: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pgxPool.Exec(context.Background(), `DELETE FROM student_profiles WHERE user_id = $1`, userID)
	})
}
