package reports

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	reportsv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/reports/v1"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/auth"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/authz"
	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/reports/reportsdb"
)

const (
	// actaCap is the maximum number of StudentGradeRows returned by GetSectionGradeReport.
	// The query uses LIMIT actaCap+1; if actaCap+1 rows arrive, Truncated is set.
	actaCap = 500
	// occupancyCap is the maximum number of rows for GetSectionOccupancyReport.
	occupancyCap = 1000
	// programCap is the maximum number of rows for GetProgramSummaryReport.
	programCap = 200
	// fichaCap is the maximum number of rows for GetStudentRecordReport.
	fichaCap = 1000
)

// Service implements the reports domain use cases: four read RPCs with
// Redis cache-aside, per-RPC authz guard, and row grouping for the grade acta.
type Service struct {
	repo  Repository
	cache Cache
	ttl   time.Duration
}

// NewService constructs a Service with the provided dependencies.
// ttl sets the Redis TTL for all cached reports and must be > 0.
func NewService(repo Repository, cache Cache, ttl time.Duration) *Service {
	return &Service{repo: repo, cache: cache, ttl: ttl}
}

// GetSectionGradeReport returns the grade acta for a section.
// Admin callers (PermCatalogManage) receive all rows.
// Teacher callers receive only rows for sections they own; out-of-scope → ErrNotFound.
func (s *Service) GetSectionGradeReport(ctx context.Context, sectionID uuid.UUID) (*reportsv1.GetSectionGradeReportResponse, error) {
	isAdmin := callerIsAdmin(ctx)

	// Cache lookup (fail-open: redis errors treated as miss).
	cacheKey := buildSectionGradeKey(sectionID)
	var target reportsv1.GetSectionGradeReportResponse
	if resp, ok := s.cacheGet(ctx, cacheKey, &target); ok {
		return resp.(*reportsv1.GetSectionGradeReportResponse), nil
	}

	// Existence check.
	exists, err := s.repo.SectionExists(ctx, sectionID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}

	var resp *reportsv1.GetSectionGradeReportResponse

	if isAdmin {
		rows, err := s.repo.ActaForSectionAdmin(ctx, sectionID)
		if err != nil {
			return nil, err
		}
		resp = buildSectionGradeResponse(sectionID, rows, actaCap)
	} else {
		// Teacher path: must extract caller ID.
		callerID, ok := auth.UserIDFromContext(ctx)
		if !ok {
			return nil, ErrNotFound
		}
		rows, err := s.repo.ActaForSectionByTeacher(ctx, sectionID, callerID)
		if err != nil {
			return nil, err
		}
		resp = buildSectionGradeResponseFromTeacher(sectionID, rows, actaCap)
	}

	s.cacheSet(ctx, cacheKey, resp)
	return resp, nil
}

// GetSectionOccupancyReport returns occupancy data for all sections in an academic period.
// Admin-only (PermCatalogManage). Teachers receive ErrPermissionDenied immediately — no cache or DB access.
func (s *Service) GetSectionOccupancyReport(ctx context.Context, periodID uuid.UUID) (*reportsv1.GetSectionOccupancyReportResponse, error) {
	// Admin guard BEFORE cache lookup.
	if !callerIsAdmin(ctx) {
		return nil, ErrPermissionDenied
	}

	cacheKey := buildOccupancyKey(periodID)
	var target reportsv1.GetSectionOccupancyReportResponse
	if resp, ok := s.cacheGet(ctx, cacheKey, &target); ok {
		return resp.(*reportsv1.GetSectionOccupancyReportResponse), nil
	}

	exists, err := s.repo.PeriodExists(ctx, periodID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}

	rows, err := s.repo.OccupancyForPeriod(ctx, periodID)
	if err != nil {
		return nil, err
	}
	resp := buildOccupancyResponse(periodID, rows, occupancyCap)
	s.cacheSet(ctx, cacheKey, resp)
	return resp, nil
}

// GetProgramSummaryReport returns quota and enrollment counts for a program in a given year.
// Admin-only (PermCatalogManage). Teachers receive ErrPermissionDenied.
func (s *Service) GetProgramSummaryReport(ctx context.Context, programID uuid.UUID, year int32) (*reportsv1.GetProgramSummaryReportResponse, error) {
	if !callerIsAdmin(ctx) {
		return nil, ErrPermissionDenied
	}

	cacheKey := buildProgramSummaryKey(programID, int(year))
	var target reportsv1.GetProgramSummaryReportResponse
	if resp, ok := s.cacheGet(ctx, cacheKey, &target); ok {
		return resp.(*reportsv1.GetProgramSummaryReportResponse), nil
	}

	exists, err := s.repo.ProgramExists(ctx, programID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}

	rows, err := s.repo.ProgramSummary(ctx, programID, year)
	if err != nil {
		return nil, err
	}
	resp := buildProgramSummaryResponse(programID, year, rows, programCap)
	s.cacheSet(ctx, cacheKey, resp)
	return resp, nil
}

// GetStudentRecordReport returns the complete academic record for a student.
// Admin-only (PermCatalogManage). Teachers receive ErrPermissionDenied.
func (s *Service) GetStudentRecordReport(ctx context.Context, studentID uuid.UUID) (*reportsv1.GetStudentRecordReportResponse, error) {
	if !callerIsAdmin(ctx) {
		return nil, ErrPermissionDenied
	}

	cacheKey := buildStudentRecordKey(studentID)
	var target reportsv1.GetStudentRecordReportResponse
	if resp, ok := s.cacheGet(ctx, cacheKey, &target); ok {
		return resp.(*reportsv1.GetStudentRecordReportResponse), nil
	}

	exists, err := s.repo.StudentExists(ctx, studentID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}

	rows, err := s.repo.FichaForStudent(ctx, studentID)
	if err != nil {
		return nil, err
	}
	resp := buildStudentRecordResponse(studentID, rows, fichaCap)
	s.cacheSet(ctx, cacheKey, resp)
	return resp, nil
}

// --- Cache helpers ---

// cacheGet retrieves and deserializes a cached proto message.
// Returns (msg, true) on hit. Returns (nil, false) on miss OR Redis error (fail-open).
// target must be the zero value of the target type.
func (s *Service) cacheGet(ctx context.Context, key string, target proto.Message) (proto.Message, bool) {
	data, found, err := s.cache.Get(ctx, key)
	if err != nil {
		logCacheGetError(ctx, key, err)
		return nil, false
	}
	if !found {
		return nil, false
	}
	if err := protojson.Unmarshal(data, target); err != nil {
		// Corrupt cache entry — treat as miss so we recompute.
		logCacheGetError(ctx, key, fmt.Errorf("protojson unmarshal: %w", err))
		return nil, false
	}
	return target, true
}

// cacheSet serializes a proto message and stores it with the service TTL.
// Redis errors are swallowed after logging (best-effort).
func (s *Service) cacheSet(ctx context.Context, key string, msg proto.Message) {
	data, err := protoMarshal(msg)
	if err != nil {
		logCacheSetError(ctx, key, fmt.Errorf("protojson marshal: %w", err))
		return
	}
	if err := s.cache.Set(ctx, key, data, s.ttl); err != nil {
		logCacheSetError(ctx, key, err)
	}
}

// protoMarshal serializes a proto message to JSON bytes using protojson.
// protojson is required (not encoding/json) because proto structs have unexported fields.
func protoMarshal(msg proto.Message) ([]byte, error) {
	return protojson.Marshal(msg)
}

// --- callerIsAdmin ---

// callerIsAdmin returns true when the authenticated caller holds PermCatalogManage,
// which identifies admin-level access in the reports domain.
// Teachers hold PermReportsRead but NOT PermCatalogManage.
func callerIsAdmin(ctx context.Context) bool {
	perms, ok := authz.PermissionsFromContext(ctx)
	if !ok {
		return false
	}
	return perms.Has(authz.PermCatalogManage)
}

// --- Response builders ---

// buildSectionGradeResponse groups flat ActaForSectionAdminRow results into
// StudentGradeRows with nested PartialGrades. Applies LIMIT cap+1 truncation detection.
func buildSectionGradeResponse(sectionID uuid.UUID, rows []reportsdb.ActaForSectionAdminRow, cap int) *reportsv1.GetSectionGradeReportResponse {
	truncated := len(rows) > cap
	if truncated {
		rows = rows[:cap]
	}

	// Group by student UUID (preserving insertion order via slice of keys).
	type studentEntry struct {
		row    reportsdb.ActaForSectionAdminRow
		grades []*reportsv1.PartialGrade
	}
	studentMap := make(map[uuid.UUID]*studentEntry)
	var studentOrder []uuid.UUID

	for _, r := range rows {
		studentID := uuid.UUID(r.StudentID.Bytes)
		entry, exists := studentMap[studentID]
		if !exists {
			entry = &studentEntry{row: r}
			studentMap[studentID] = entry
			studentOrder = append(studentOrder, studentID)
		}

		partial := &reportsv1.PartialGrade{
			EvaluationId: uuid.UUID(r.EvaluationID.Bytes).String(),
			Position:     r.Position.Int32,
		}
		if r.GradeValue.Valid {
			partial.Value = numericToString(r.GradeValue)
		}
		entry.grades = append(entry.grades, partial)
	}

	gradeRows := make([]*reportsv1.StudentGradeRow, 0, len(studentOrder))
	for _, sid := range studentOrder {
		entry := studentMap[sid]
		r := entry.row

		row := &reportsv1.StudentGradeRow{
			StudentId:        sid.String(),
			GivenNames:       r.GivenNames,
			LastNamePaternal: r.LastNamePaternal,
			PartialGrades:    entry.grades,
		}
		if r.LastNameMaternal.Valid {
			row.LastNameMaternal = r.LastNameMaternal.String
		}
		if r.FinalGrade.Valid {
			row.FinalGrade = numericToString(r.FinalGrade)
			row.Outcome = gradeOutcome(row.FinalGrade)
		}
		gradeRows = append(gradeRows, row)
	}

	return &reportsv1.GetSectionGradeReportResponse{
		SectionId:   sectionID.String(),
		GeneratedAt: generatedAt(),
		Truncated:   truncated,
		Rows:        gradeRows,
	}
}

// buildSectionGradeResponseFromTeacher groups flat ActaForSectionByTeacherRow results.
func buildSectionGradeResponseFromTeacher(sectionID uuid.UUID, rows []reportsdb.ActaForSectionByTeacherRow, cap int) *reportsv1.GetSectionGradeReportResponse {
	truncated := len(rows) > cap
	if truncated {
		rows = rows[:cap]
	}

	type studentEntry struct {
		row    reportsdb.ActaForSectionByTeacherRow
		grades []*reportsv1.PartialGrade
	}
	studentMap := make(map[uuid.UUID]*studentEntry)
	var studentOrder []uuid.UUID

	for _, r := range rows {
		studentID := uuid.UUID(r.StudentID.Bytes)
		entry, exists := studentMap[studentID]
		if !exists {
			entry = &studentEntry{row: r}
			studentMap[studentID] = entry
			studentOrder = append(studentOrder, studentID)
		}

		partial := &reportsv1.PartialGrade{
			EvaluationId: uuid.UUID(r.EvaluationID.Bytes).String(),
			Position:     r.Position.Int32,
		}
		if r.GradeValue.Valid {
			partial.Value = numericToString(r.GradeValue)
		}
		entry.grades = append(entry.grades, partial)
	}

	gradeRows := make([]*reportsv1.StudentGradeRow, 0, len(studentOrder))
	for _, sid := range studentOrder {
		entry := studentMap[sid]
		r := entry.row

		row := &reportsv1.StudentGradeRow{
			StudentId:        sid.String(),
			GivenNames:       r.GivenNames,
			LastNamePaternal: r.LastNamePaternal,
			PartialGrades:    entry.grades,
		}
		if r.LastNameMaternal.Valid {
			row.LastNameMaternal = r.LastNameMaternal.String
		}
		if r.FinalGrade.Valid {
			row.FinalGrade = numericToString(r.FinalGrade)
			row.Outcome = gradeOutcome(row.FinalGrade)
		}
		gradeRows = append(gradeRows, row)
	}

	return &reportsv1.GetSectionGradeReportResponse{
		SectionId:   sectionID.String(),
		GeneratedAt: generatedAt(),
		Truncated:   truncated,
		Rows:        gradeRows,
	}
}

// buildOccupancyResponse converts OccupancyForPeriodRow slice to proto response.
func buildOccupancyResponse(periodID uuid.UUID, rows []reportsdb.OccupancyForPeriodRow, cap int) *reportsv1.GetSectionOccupancyReportResponse {
	truncated := len(rows) > cap
	if truncated {
		rows = rows[:cap]
	}

	protoRows := make([]*reportsv1.SectionOccupancyRow, 0, len(rows))
	for _, r := range rows {
		sectionID := uuid.UUID(r.SectionID.Bytes)
		seatCount := r.ActiveSeatCount // DB returns int64; proto uses int32
		if seatCount > math.MaxInt32 {
			seatCount = math.MaxInt32
		}
		active := int32(seatCount)
		capacity := r.Capacity

		row := &reportsv1.SectionOccupancyRow{
			SectionId:       sectionID.String(),
			Capacity:        capacity,
			ActiveSeatCount: active,
		}
		if r.CourseName.Valid {
			row.CourseName = r.CourseName.String
		}
		// fill_percentage: guard against division by zero.
		if capacity > 0 {
			row.FillPercentage = fmt.Sprintf("%.2f", float64(active)/float64(capacity)*100)
		} else {
			row.FillPercentage = "0.00"
		}
		protoRows = append(protoRows, row)
	}

	return &reportsv1.GetSectionOccupancyReportResponse{
		AcademicPeriodId: periodID.String(),
		GeneratedAt:      generatedAt(),
		Truncated:        truncated,
		Rows:             protoRows,
	}
}

// buildProgramSummaryResponse converts ProgramSummaryRow slice to proto response.
func buildProgramSummaryResponse(programID uuid.UUID, year int32, rows []reportsdb.ProgramSummaryRow, cap int) *reportsv1.GetProgramSummaryReportResponse {
	truncated := len(rows) > cap
	if truncated {
		rows = rows[:cap]
	}

	protoRows := make([]*reportsv1.ProgramEnrollmentRow, 0, len(rows))
	for _, r := range rows {
		quotaID := uuid.UUID(r.QuotaID.Bytes)
		capacity := r.QuotaCapacity
		enrolled := r.EnrolledCount
		available := capacity - enrolled
		if available < 0 {
			available = 0
		}

		row := &reportsv1.ProgramEnrollmentRow{
			QuotaId:        quotaID.String(),
			QuotaCapacity:  capacity,
			EnrolledCount:  enrolled,
			AvailableSeats: available,
		}
		if capacity > 0 {
			row.FillPercentage = fmt.Sprintf("%.2f", float64(enrolled)/float64(capacity)*100)
		} else {
			row.FillPercentage = "0.00"
		}
		protoRows = append(protoRows, row)
	}

	return &reportsv1.GetProgramSummaryReportResponse{
		ProgramId:   programID.String(),
		Year:        year,
		GeneratedAt: generatedAt(),
		Truncated:   truncated,
		Rows:        protoRows,
	}
}

// buildStudentRecordResponse converts FichaForStudentRow slice to proto response.
// Note: AcademicRecordRow in the proto does not have PartialGrades — grade info
// is stored directly as final_grade/outcome fields. Individual evaluation grades
// are intentionally not exposed in the ficha RPC.
func buildStudentRecordResponse(studentID uuid.UUID, rows []reportsdb.FichaForStudentRow, cap int) *reportsv1.GetStudentRecordReportResponse {
	truncated := len(rows) > cap
	if truncated {
		rows = rows[:cap]
	}

	protoRows := make([]*reportsv1.AcademicRecordRow, 0, len(rows))
	for _, r := range rows {
		sectionID := uuid.UUID(r.SectionID.Bytes)
		periodID := uuid.UUID(r.AcademicPeriodID.Bytes)

		row := &reportsv1.AcademicRecordRow{
			AcademicPeriodId:   periodID.String(),
			AcademicPeriodName: fmt.Sprintf("%v", r.AcademicPeriodName),
			SectionId:          sectionID.String(),
			CourseName:         r.CourseName,
			EnrollmentStatus:   r.EnrollmentStatus,
		}

		if r.FinalGrade.Valid {
			row.FinalGrade = numericToString(r.FinalGrade)
			row.Outcome = gradeOutcome(row.FinalGrade)
		}
		protoRows = append(protoRows, row)
	}

	return &reportsv1.GetStudentRecordReportResponse{
		StudentId:   studentID.String(),
		GeneratedAt: generatedAt(),
		Truncated:   truncated,
		Rows:        protoRows,
	}
}

// --- Conversion helpers ---

// numericToString converts a pgtype.Numeric to its string representation.
// Returns empty string if the value is not valid.
func numericToString(n pgtype.Numeric) string {
	if !n.Valid {
		return ""
	}
	text, err := n.Value()
	if err != nil {
		return ""
	}
	if text == nil {
		return ""
	}
	return fmt.Sprintf("%v", text)
}

// gradeOutcome maps a grade string to "passed" / "failed" using the 4.0 threshold.
// An unparseable grade is mapped to "failed" (safe default).
func gradeOutcome(grade string) string {
	// Parse as a Rat to avoid float drift at boundary.
	r, ok := new(big.Rat).SetString(grade)
	if !ok {
		return "failed"
	}
	threshold := new(big.Rat).SetFrac(big.NewInt(4), big.NewInt(1))
	if r.Cmp(threshold) >= 0 {
		return "passed"
	}
	return "failed"
}

// generatedAt returns the current UTC time formatted as ISO-8601.
func generatedAt() string {
	return time.Now().UTC().Format(time.RFC3339)
}
