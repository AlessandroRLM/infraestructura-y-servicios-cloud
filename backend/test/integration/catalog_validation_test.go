package integration_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	catalogv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/catalog/v1"
)

// TestCatalog_Validation_EmptyProgramCode verifies that empty code returns CodeInvalidArgument.
func TestCatalog_Validation_EmptyProgramCode(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-val-empty-code@catalog.test")
	client := newCatalogClient(nil)

	req := connect.NewRequest(&catalogv1.CreateProgramRequest{Code: "", Name: "Valid Name"})
	req.Header().Set("Cookie", "sid="+adminSID)

	_, err := client.CreateProgram(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestCatalog_Validation_EmptyCourseName verifies that empty name returns CodeInvalidArgument.
func TestCatalog_Validation_EmptyCourseName(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-val-empty-name@catalog.test")
	client := newCatalogClient(nil)

	req := connect.NewRequest(&catalogv1.CreateCourseRequest{Code: "CRS-VAL-1", Name: "", Credits: 3})
	req.Header().Set("Cookie", "sid="+adminSID)

	_, err := client.CreateCourse(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestCatalog_Validation_ZeroCredits verifies that credits=0 returns CodeInvalidArgument.
func TestCatalog_Validation_ZeroCredits(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-val-zero-cred@catalog.test")
	client := newCatalogClient(nil)

	req := connect.NewRequest(&catalogv1.CreateCourseRequest{Code: "CRS-VAL-2", Name: "Course", Credits: 0})
	req.Header().Set("Cookie", "sid="+adminSID)

	_, err := client.CreateCourse(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestCatalog_Validation_NegativeCredits verifies that credits=-1 returns CodeInvalidArgument.
func TestCatalog_Validation_NegativeCredits(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-val-neg-cred@catalog.test")
	client := newCatalogClient(nil)

	req := connect.NewRequest(&catalogv1.CreateCourseRequest{Code: "CRS-VAL-3", Name: "Course", Credits: -1})
	req.Header().Set("Cookie", "sid="+adminSID)

	_, err := client.CreateCourse(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestCatalog_Validation_ProgramQuota_ZeroCapacity verifies capacity=0 returns CodeInvalidArgument.
func TestCatalog_Validation_ProgramQuota_ZeroCapacity(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-val-quota-cap@catalog.test")
	client := newCatalogClient(nil)

	req := connect.NewRequest(&catalogv1.CreateProgramQuotaRequest{
		ProgramId:      uuid.New().String(),
		Year:           2025,
		AdmissionQuota: 0,
	})
	req.Header().Set("Cookie", "sid="+adminSID)

	_, err := client.CreateProgramQuota(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestCatalog_Validation_AcademicPeriod_InvalidTermZero verifies term=0 returns CodeInvalidArgument.
func TestCatalog_Validation_AcademicPeriod_InvalidTermZero(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-val-term-zero@catalog.test")
	client := newCatalogClient(nil)

	req := connect.NewRequest(&catalogv1.CreateAcademicPeriodRequest{
		Year: 2025, Term: 0, StartDate: "2025-03-01", EndDate: "2025-07-31",
	})
	req.Header().Set("Cookie", "sid="+adminSID)

	_, err := client.CreateAcademicPeriod(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestCatalog_Validation_AcademicPeriod_InvalidTermThree verifies term=3 returns CodeInvalidArgument.
func TestCatalog_Validation_AcademicPeriod_InvalidTermThree(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-val-term-three@catalog.test")
	client := newCatalogClient(nil)

	req := connect.NewRequest(&catalogv1.CreateAcademicPeriodRequest{
		Year: 2025, Term: 3, StartDate: "2025-03-01", EndDate: "2025-07-31",
	})
	req.Header().Set("Cookie", "sid="+adminSID)

	_, err := client.CreateAcademicPeriod(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestCatalog_Validation_AcademicPeriod_ValidTerms verifies term=1 and term=2 are accepted.
func TestCatalog_Validation_AcademicPeriod_ValidTerms(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-val-terms-ok@catalog.test")
	client := newCatalogClient(nil)

	for _, term := range []int32{1, 2} {
		req := connect.NewRequest(&catalogv1.CreateAcademicPeriodRequest{
			Year:      4000 + int32(term),
			Term:      term,
			StartDate: "4001-03-01",
			EndDate:   "4001-07-31",
		})
		req.Header().Set("Cookie", "sid="+adminSID)
		resp, err := client.CreateAcademicPeriod(ctx, req)
		if err != nil {
			t.Errorf("CreateAcademicPeriod(term=%d): unexpected error: %v", term, err)
			continue
		}
		t.Cleanup(func() { cleanupAcademicPeriod(t, resp.Msg.GetId()) })
	}
}

// TestCatalog_Validation_AcademicPeriod_EqualDates verifies start=end returns CodeInvalidArgument.
func TestCatalog_Validation_AcademicPeriod_EqualDates(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-val-eq-dates@catalog.test")
	client := newCatalogClient(nil)

	req := connect.NewRequest(&catalogv1.CreateAcademicPeriodRequest{
		Year: 2025, Term: 1, StartDate: "2025-03-01", EndDate: "2025-03-01",
	})
	req.Header().Set("Cookie", "sid="+adminSID)

	_, err := client.CreateAcademicPeriod(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestCatalog_Validation_AcademicPeriod_StartAfterEnd verifies start>end returns CodeInvalidArgument.
func TestCatalog_Validation_AcademicPeriod_StartAfterEnd(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "catalog-val-start-after@catalog.test")
	client := newCatalogClient(nil)

	req := connect.NewRequest(&catalogv1.CreateAcademicPeriodRequest{
		Year: 2025, Term: 1, StartDate: "2025-08-01", EndDate: "2025-03-01",
	})
	req.Header().Set("Cookie", "sid="+adminSID)

	_, err := client.CreateAcademicPeriod(ctx, req)
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}
