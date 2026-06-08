package integration_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	catalogv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/catalog/v1"
	profilesv1 "github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/gen/profiles/v1"
)

// cleanupSection removes a section row (and its section_teachers) by id.
func cleanupSection(t *testing.T, id string) {
	t.Helper()
	ctx := context.Background()
	_, _ = pgxPool.Exec(ctx, `DELETE FROM section_teachers WHERE section_id = $1`, id)
	_, _ = pgxPool.Exec(ctx, `DELETE FROM sections WHERE id = $1`, id)
}

// seedTeacherProfile creates a user with role "teacher", inserts a teacher_profiles row,
// and returns (userID string, sessionID string).
func seedTeacherProfile(t *testing.T, email string) (string, string) {
	t.Helper()
	adminSID := catalogSeedAdminSession(t, "sect-admin-for-teacher-seed-"+email)
	teacherID := seedUserWithRole(t, email, "teacher")
	teacherSID := seedSessionInRedis(t, teacherID, time.Hour)

	client := newProfilesClient(nil)
	ctx := context.Background()

	dept := "Test Department"
	_, err := client.UpsertTeacherProfile(ctx, withSID(connect.NewRequest(&profilesv1.UpsertTeacherProfileRequest{
		UserId:     teacherID.String(),
		Department: &dept,
	}), adminSID))
	if err != nil {
		t.Fatalf("seedTeacherProfile: UpsertTeacherProfile(%s): %v", email, err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pgxPool.Exec(ctx, `DELETE FROM section_teachers WHERE teacher_id = $1`, teacherID)
		_, _ = pgxPool.Exec(ctx, `DELETE FROM teacher_qualifications WHERE teacher_id = $1`, teacherID)
		_, _ = pgxPool.Exec(ctx, `DELETE FROM teacher_profiles WHERE user_id = $1`, teacherID)
	})
	return teacherID.String(), teacherSID
}

// seedSectionFixture creates the prerequisite course and academic period for a section,
// then creates the section itself. Returns (sectionID, courseID, periodID).
func seedSectionFixture(t *testing.T, adminSID string) (sectionID, courseID, periodID string) {
	t.Helper()
	ctx := context.Background()
	client := newCatalogClient(nil)

	cResp, err := client.CreateCourse(ctx, withSID(connect.NewRequest(&catalogv1.CreateCourseRequest{
		Code:    "SECT-CRS-" + uuid.New().String()[:8],
		Name:    "Section Course",
		Credits: 4,
	}), adminSID))
	if err != nil {
		t.Fatalf("seedSectionFixture: CreateCourse: %v", err)
	}
	courseID = cResp.Msg.GetId()
	t.Cleanup(func() { cleanupCourse(t, courseID) })

	pResp, err := client.CreateAcademicPeriod(ctx, withSID(connect.NewRequest(&catalogv1.CreateAcademicPeriodRequest{
		Year:      9000 + int32(time.Now().UnixNano()%1000),
		Term:      1,
		StartDate: "9000-03-01",
		EndDate:   "9000-07-31",
	}), adminSID))
	if err != nil {
		// Year collision: use a different year range.
		pResp2, err2 := client.CreateAcademicPeriod(ctx, withSID(connect.NewRequest(&catalogv1.CreateAcademicPeriodRequest{
			Year:      8000 + int32(time.Now().UnixNano()%1000),
			Term:      2,
			StartDate: "8000-08-01",
			EndDate:   "8000-12-31",
		}), adminSID))
		if err2 != nil {
			t.Fatalf("seedSectionFixture: CreateAcademicPeriod (retry): %v", err2)
		}
		periodID = pResp2.Msg.GetId()
	} else {
		periodID = pResp.Msg.GetId()
	}
	t.Cleanup(func() { cleanupAcademicPeriod(t, periodID) })

	sResp, err := client.CreateSection(ctx, withSID(connect.NewRequest(&catalogv1.CreateSectionRequest{
		CourseId:         courseID,
		AcademicPeriodId: periodID,
		SeatCapacity:     30,
	}), adminSID))
	if err != nil {
		t.Fatalf("seedSectionFixture: CreateSection: %v", err)
	}
	sectionID = sResp.Msg.GetId()
	t.Cleanup(func() { cleanupSection(t, sectionID) })

	return sectionID, courseID, periodID
}

// ── Section CRUD ──────────────────────────────────────────────────────────────

// TestSection_FullLifecycle verifies Create→Get→Update→List→Delete→Get(NotFound)→List(excluded).
func TestSection_FullLifecycle(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "sect-lifecycle@catalog.test")
	client := newCatalogClient(nil)

	// Fixtures
	cResp, err := client.CreateCourse(ctx, withSID(connect.NewRequest(&catalogv1.CreateCourseRequest{
		Code:    "SECT-LIFE-CRS-" + uuid.New().String()[:8],
		Name:    "Life Course",
		Credits: 3,
	}), adminSID))
	if err != nil {
		t.Fatalf("CreateCourse: %v", err)
	}
	courseID := cResp.Msg.GetId()
	t.Cleanup(func() { cleanupCourse(t, courseID) })

	apResp, err := client.CreateAcademicPeriod(ctx, withSID(connect.NewRequest(&catalogv1.CreateAcademicPeriodRequest{
		Year:      7100,
		Term:      1,
		StartDate: "7100-03-01",
		EndDate:   "7100-07-31",
	}), adminSID))
	if err != nil {
		t.Fatalf("CreateAcademicPeriod: %v", err)
	}
	periodID := apResp.Msg.GetId()
	t.Cleanup(func() { cleanupAcademicPeriod(t, periodID) })

	// Create
	sResp, err := client.CreateSection(ctx, withSID(connect.NewRequest(&catalogv1.CreateSectionRequest{
		CourseId:         courseID,
		AcademicPeriodId: periodID,
		SeatCapacity:     25,
	}), adminSID))
	if err != nil {
		t.Fatalf("CreateSection: %v", err)
	}
	id := sResp.Msg.GetId()
	t.Cleanup(func() { cleanupSection(t, id) })
	if sResp.Msg.GetSeatCapacity() != 25 {
		t.Errorf("CreateSection seat_capacity = %d, want 25", sResp.Msg.GetSeatCapacity())
	}

	// Get
	getResp, err := client.GetSection(ctx, withSID(connect.NewRequest(&catalogv1.GetSectionRequest{Id: id}), adminSID))
	if err != nil {
		t.Errorf("GetSection after create: %v", err)
	}
	if getResp.Msg.GetCourseId() != courseID {
		t.Errorf("GetSection course_id = %q, want %q", getResp.Msg.GetCourseId(), courseID)
	}

	// Update
	updateResp, err := client.UpdateSection(ctx, withSID(connect.NewRequest(&catalogv1.UpdateSectionRequest{
		Id:           id,
		SeatCapacity: 40,
	}), adminSID))
	if err != nil {
		t.Errorf("UpdateSection: %v", err)
	}
	if updateResp.Msg.GetSeatCapacity() != 40 {
		t.Errorf("UpdateSection seat_capacity = %d, want 40", updateResp.Msg.GetSeatCapacity())
	}

	// List — must include
	listResp, err := client.ListSections(ctx, withSID(connect.NewRequest(&catalogv1.ListSectionsRequest{}), adminSID))
	if err != nil {
		t.Fatalf("ListSections: %v", err)
	}
	found := false
	for _, s := range listResp.Msg.GetSections() {
		if s.GetId() == id {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ListSections: section %s not found", id)
	}

	// Delete
	if _, err := client.DeleteSection(ctx, withSID(connect.NewRequest(&catalogv1.DeleteSectionRequest{Id: id}), adminSID)); err != nil {
		t.Errorf("DeleteSection: %v", err)
	}

	// Get after delete — must be NotFound
	_, err = client.GetSection(ctx, withSID(connect.NewRequest(&catalogv1.GetSectionRequest{Id: id}), adminSID))
	assertConnectCode(t, err, connect.CodeNotFound)

	// List after delete — must exclude
	listAfter, err := client.ListSections(ctx, withSID(connect.NewRequest(&catalogv1.ListSectionsRequest{}), adminSID))
	if err != nil {
		t.Fatalf("ListSections after delete: %v", err)
	}
	for _, s := range listAfter.Msg.GetSections() {
		if s.GetId() == id {
			t.Errorf("ListSections after delete: soft-deleted section %s still appears", id)
		}
	}
}

// TestSection_Audit_CreatedByUpdatedBySet verifies audit columns for sections.
func TestSection_Audit_CreatedByUpdatedBySet(t *testing.T) {
	ctx := context.Background()
	adminID := seedUserWithRole(t, "sect-audit@catalog.test", "admin")
	adminSID := seedSessionInRedis(t, adminID, time.Hour)
	client := newCatalogClient(nil)

	cResp, err := client.CreateCourse(ctx, withSID(connect.NewRequest(&catalogv1.CreateCourseRequest{
		Code:    "SECT-AUD-CRS-" + uuid.New().String()[:8],
		Name:    "Audit Course",
		Credits: 2,
	}), adminSID))
	if err != nil {
		t.Fatalf("CreateCourse: %v", err)
	}
	courseID := cResp.Msg.GetId()
	t.Cleanup(func() { cleanupCourse(t, courseID) })

	apResp, err := client.CreateAcademicPeriod(ctx, withSID(connect.NewRequest(&catalogv1.CreateAcademicPeriodRequest{
		Year:      7200,
		Term:      1,
		StartDate: "7200-03-01",
		EndDate:   "7200-07-31",
	}), adminSID))
	if err != nil {
		t.Fatalf("CreateAcademicPeriod: %v", err)
	}
	periodID := apResp.Msg.GetId()
	t.Cleanup(func() { cleanupAcademicPeriod(t, periodID) })

	sResp, err := client.CreateSection(ctx, withSID(connect.NewRequest(&catalogv1.CreateSectionRequest{
		CourseId:         courseID,
		AcademicPeriodId: periodID,
		SeatCapacity:     20,
	}), adminSID))
	if err != nil {
		t.Fatalf("CreateSection: %v", err)
	}
	id := sResp.Msg.GetId()
	t.Cleanup(func() { cleanupSection(t, id) })

	var createdBy, updatedBy string
	if err := pgxPool.QueryRow(ctx,
		`SELECT created_by::text, updated_by::text FROM sections WHERE id = $1`, id,
	).Scan(&createdBy, &updatedBy); err != nil {
		t.Fatalf("SELECT audit cols from sections: %v", err)
	}
	if createdBy != adminID.String() {
		t.Errorf("sections.created_by = %q, want %q", createdBy, adminID.String())
	}
	if updatedBy != adminID.String() {
		t.Errorf("sections.updated_by = %q, want %q", updatedBy, adminID.String())
	}
}

// ── Section validation ────────────────────────────────────────────────────────

// TestSection_Validation_ZeroCapacity verifies that seat_capacity=0 returns CodeInvalidArgument.
func TestSection_Validation_ZeroCapacity(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "sect-val-zero-cap@catalog.test")
	client := newCatalogClient(nil)

	_, err := client.CreateSection(ctx, withSID(connect.NewRequest(&catalogv1.CreateSectionRequest{
		CourseId:         uuid.New().String(),
		AcademicPeriodId: uuid.New().String(),
		SeatCapacity:     0,
	}), adminSID))
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestSection_Validation_NegativeCapacity verifies that seat_capacity<0 returns CodeInvalidArgument.
func TestSection_Validation_NegativeCapacity(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "sect-val-neg-cap@catalog.test")
	client := newCatalogClient(nil)

	_, err := client.CreateSection(ctx, withSID(connect.NewRequest(&catalogv1.CreateSectionRequest{
		CourseId:         uuid.New().String(),
		AcademicPeriodId: uuid.New().String(),
		SeatCapacity:     -5,
	}), adminSID))
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestSection_UpdateValidation_ZeroCapacity verifies that updating with seat_capacity=0 fails.
func TestSection_UpdateValidation_ZeroCapacity(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "sect-upd-val@catalog.test")
	client := newCatalogClient(nil)

	sectionID, _, _ := seedSectionFixture(t, adminSID)

	_, err := client.UpdateSection(ctx, withSID(connect.NewRequest(&catalogv1.UpdateSectionRequest{
		Id:           sectionID,
		SeatCapacity: 0,
	}), adminSID))
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// ── Section FK errors ─────────────────────────────────────────────────────────

// TestSection_BadFK_Course_InvalidArgument verifies that a non-existent course_id returns CodeInvalidArgument.
func TestSection_BadFK_Course_InvalidArgument(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "sect-fk-course@catalog.test")
	client := newCatalogClient(nil)

	apResp, err := client.CreateAcademicPeriod(ctx, withSID(connect.NewRequest(&catalogv1.CreateAcademicPeriodRequest{
		Year: 7300, Term: 1, StartDate: "7300-03-01", EndDate: "7300-07-31",
	}), adminSID))
	if err != nil {
		t.Fatalf("CreateAcademicPeriod: %v", err)
	}
	periodID := apResp.Msg.GetId()
	t.Cleanup(func() { cleanupAcademicPeriod(t, periodID) })

	_, err = client.CreateSection(ctx, withSID(connect.NewRequest(&catalogv1.CreateSectionRequest{
		CourseId:         uuid.New().String(), // non-existent
		AcademicPeriodId: periodID,
		SeatCapacity:     10,
	}), adminSID))
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestSection_BadFK_AcademicPeriod_InvalidArgument verifies that a non-existent academic_period_id returns CodeInvalidArgument.
func TestSection_BadFK_AcademicPeriod_InvalidArgument(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "sect-fk-period@catalog.test")
	client := newCatalogClient(nil)

	cResp, err := client.CreateCourse(ctx, withSID(connect.NewRequest(&catalogv1.CreateCourseRequest{
		Code: "SECT-FK-CRS-" + uuid.New().String()[:8], Name: "FK Course", Credits: 3,
	}), adminSID))
	if err != nil {
		t.Fatalf("CreateCourse: %v", err)
	}
	courseID := cResp.Msg.GetId()
	t.Cleanup(func() { cleanupCourse(t, courseID) })

	_, err = client.CreateSection(ctx, withSID(connect.NewRequest(&catalogv1.CreateSectionRequest{
		CourseId:         courseID,
		AcademicPeriodId: uuid.New().String(), // non-existent
		SeatCapacity:     10,
	}), adminSID))
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestSection_Delete_AbsentID_NotFound verifies CodeNotFound for absent section.
func TestSection_Delete_AbsentID_NotFound(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "sect-del-absent@catalog.test")
	client := newCatalogClient(nil)

	_, err := client.DeleteSection(ctx, withSID(connect.NewRequest(&catalogv1.DeleteSectionRequest{
		Id: uuid.New().String(),
	}), adminSID))
	assertConnectCode(t, err, connect.CodeNotFound)
}

// ── Dependent-blocking: DeleteCourse and DeleteAcademicPeriod ────────────────

// TestSection_DeleteCourse_BlockedByLiveSection verifies that DeleteCourse fails when a live section references it.
func TestSection_DeleteCourse_BlockedByLiveSection(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "sect-del-course-blocked@catalog.test")
	client := newCatalogClient(nil)

	_, courseID, _ := seedSectionFixture(t, adminSID)

	// Delete course — must be blocked by the live section.
	_, err := client.DeleteCourse(ctx, withSID(connect.NewRequest(&catalogv1.DeleteCourseRequest{Id: courseID}), adminSID))
	assertConnectCode(t, err, connect.CodeFailedPrecondition)

	// Course row must still be live.
	var deletedAt *string
	if err := pgxPool.QueryRow(ctx, `SELECT deleted_at::text FROM courses WHERE id = $1`, courseID).Scan(&deletedAt); err != nil {
		t.Fatalf("SELECT deleted_at from courses: %v", err)
	}
	if deletedAt != nil {
		t.Errorf("DeleteCourse (blocked): course row was soft-deleted, expected live")
	}
}

// TestSection_DeleteCourse_AllowedAfterSectionDeleted verifies that DeleteCourse succeeds
// once the blocking section has been soft-deleted.
func TestSection_DeleteCourse_AllowedAfterSectionDeleted(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "sect-del-course-ok@catalog.test")
	client := newCatalogClient(nil)

	sectionID, courseID, _ := seedSectionFixture(t, adminSID)

	// Soft-delete the section first.
	if _, err := client.DeleteSection(ctx, withSID(connect.NewRequest(&catalogv1.DeleteSectionRequest{Id: sectionID}), adminSID)); err != nil {
		t.Fatalf("DeleteSection (prereq): %v", err)
	}

	// Delete course — must now succeed (soft-deleted section does not block).
	if _, err := client.DeleteCourse(ctx, withSID(connect.NewRequest(&catalogv1.DeleteCourseRequest{Id: courseID}), adminSID)); err != nil {
		t.Errorf("DeleteCourse after section deleted: %v", err)
	}
}

// TestSection_DeleteAcademicPeriod_BlockedByLiveSection verifies CodeFailedPrecondition.
func TestSection_DeleteAcademicPeriod_BlockedByLiveSection(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "sect-del-ap-blocked@catalog.test")
	client := newCatalogClient(nil)

	_, _, periodID := seedSectionFixture(t, adminSID)

	_, err := client.DeleteAcademicPeriod(ctx, withSID(connect.NewRequest(&catalogv1.DeleteAcademicPeriodRequest{Id: periodID}), adminSID))
	assertConnectCode(t, err, connect.CodeFailedPrecondition)
}

// TestSection_DeleteAcademicPeriod_AllowedAfterSectionDeleted verifies that deleting
// the section unblocks the academic period deletion.
func TestSection_DeleteAcademicPeriod_AllowedAfterSectionDeleted(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "sect-del-ap-ok@catalog.test")
	client := newCatalogClient(nil)

	sectionID, _, periodID := seedSectionFixture(t, adminSID)

	if _, err := client.DeleteSection(ctx, withSID(connect.NewRequest(&catalogv1.DeleteSectionRequest{Id: sectionID}), adminSID)); err != nil {
		t.Fatalf("DeleteSection (prereq): %v", err)
	}

	if _, err := client.DeleteAcademicPeriod(ctx, withSID(connect.NewRequest(&catalogv1.DeleteAcademicPeriodRequest{Id: periodID}), adminSID)); err != nil {
		t.Errorf("DeleteAcademicPeriod after section deleted: %v", err)
	}
}

// ── DeleteSection blocked by section_teachers ─────────────────────────────────

// TestSection_DeleteSection_BlockedByTeacherAssignment verifies CodeFailedPrecondition.
func TestSection_DeleteSection_BlockedByTeacherAssignment(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "sect-del-blocked-by-teacher@catalog.test")
	client := newCatalogClient(nil)

	sectionID, _, _ := seedSectionFixture(t, adminSID)
	teacherID, _ := seedTeacherProfile(t, "sect-teacher-block@catalog.test")

	// Assign teacher.
	if _, err := client.AssignTeacherToSection(ctx, withSID(connect.NewRequest(&catalogv1.AssignTeacherToSectionRequest{
		SectionId: sectionID,
		TeacherId: teacherID,
	}), adminSID)); err != nil {
		t.Fatalf("AssignTeacherToSection: %v", err)
	}

	// Delete section — must be blocked.
	_, err := client.DeleteSection(ctx, withSID(connect.NewRequest(&catalogv1.DeleteSectionRequest{Id: sectionID}), adminSID))
	assertConnectCode(t, err, connect.CodeFailedPrecondition)
}

// TestSection_DeleteSection_AllowedAfterTeacherRemoved verifies that removing the teacher
// assignment allows the section to be soft-deleted.
func TestSection_DeleteSection_AllowedAfterTeacherRemoved(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "sect-del-unblocked@catalog.test")
	client := newCatalogClient(nil)

	sectionID, _, _ := seedSectionFixture(t, adminSID)
	teacherID, _ := seedTeacherProfile(t, "sect-teacher-unblock@catalog.test")

	if _, err := client.AssignTeacherToSection(ctx, withSID(connect.NewRequest(&catalogv1.AssignTeacherToSectionRequest{
		SectionId: sectionID,
		TeacherId: teacherID,
	}), adminSID)); err != nil {
		t.Fatalf("AssignTeacherToSection: %v", err)
	}

	if _, err := client.RemoveTeacherFromSection(ctx, withSID(connect.NewRequest(&catalogv1.RemoveTeacherFromSectionRequest{
		SectionId: sectionID,
		TeacherId: teacherID,
	}), adminSID)); err != nil {
		t.Fatalf("RemoveTeacherFromSection: %v", err)
	}

	if _, err := client.DeleteSection(ctx, withSID(connect.NewRequest(&catalogv1.DeleteSectionRequest{Id: sectionID}), adminSID)); err != nil {
		t.Errorf("DeleteSection after teacher removed: %v", err)
	}
}

// ── Section-teacher association ───────────────────────────────────────────────

// TestSectionTeacher_AssignListRemove verifies the full assign→list→remove roundtrip.
func TestSectionTeacher_AssignListRemove(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "sect-teacher-crud@catalog.test")
	client := newCatalogClient(nil)

	sectionID, _, _ := seedSectionFixture(t, adminSID)
	teacherID, _ := seedTeacherProfile(t, "sect-teacher-crud-t@catalog.test")

	// Assign
	assignResp, err := client.AssignTeacherToSection(ctx, withSID(connect.NewRequest(&catalogv1.AssignTeacherToSectionRequest{
		SectionId: sectionID,
		TeacherId: teacherID,
	}), adminSID))
	if err != nil {
		t.Fatalf("AssignTeacherToSection: %v", err)
	}
	if assignResp.Msg.GetTeacherId() != teacherID {
		t.Errorf("AssignTeacherToSection teacher_id = %q, want %q", assignResp.Msg.GetTeacherId(), teacherID)
	}

	// Verify row in DB.
	var count int
	if err := pgxPool.QueryRow(ctx,
		`SELECT count(*) FROM section_teachers WHERE section_id = $1 AND teacher_id = $2`,
		sectionID, teacherID,
	).Scan(&count); err != nil {
		t.Fatalf("count section_teachers: %v", err)
	}
	if count != 1 {
		t.Errorf("section_teachers count = %d, want 1", count)
	}

	// List
	listResp, err := client.ListSectionTeachers(ctx, withSID(connect.NewRequest(&catalogv1.ListSectionTeachersRequest{
		SectionId: sectionID,
	}), adminSID))
	if err != nil {
		t.Fatalf("ListSectionTeachers: %v", err)
	}
	found := false
	for _, st := range listResp.Msg.GetSectionTeachers() {
		if st.GetTeacherId() == teacherID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ListSectionTeachers: teacher %s not found", teacherID)
	}

	// Remove
	if _, err := client.RemoveTeacherFromSection(ctx, withSID(connect.NewRequest(&catalogv1.RemoveTeacherFromSectionRequest{
		SectionId: sectionID,
		TeacherId: teacherID,
	}), adminSID)); err != nil {
		t.Fatalf("RemoveTeacherFromSection: %v", err)
	}

	// Verify row gone.
	if err := pgxPool.QueryRow(ctx,
		`SELECT count(*) FROM section_teachers WHERE section_id = $1 AND teacher_id = $2`,
		sectionID, teacherID,
	).Scan(&count); err != nil {
		t.Fatalf("count section_teachers after remove: %v", err)
	}
	if count != 0 {
		t.Errorf("section_teachers count after remove = %d, want 0", count)
	}
}

// TestSectionTeacher_ListIsolation verifies ListSectionTeachers returns only teachers for the requested section.
func TestSectionTeacher_ListIsolation(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "sect-teacher-isolate@catalog.test")
	client := newCatalogClient(nil)

	s1ID, _, _ := seedSectionFixture(t, adminSID)
	s2ID, _, _ := seedSectionFixture(t, adminSID)
	t1ID, _ := seedTeacherProfile(t, "sect-teacher-iso-t1@catalog.test")
	t2ID, _ := seedTeacherProfile(t, "sect-teacher-iso-t2@catalog.test")
	t3ID, _ := seedTeacherProfile(t, "sect-teacher-iso-t3@catalog.test")

	// Assign t1 and t2 to s1; t3 to s2.
	for _, tid := range []string{t1ID, t2ID} {
		if _, err := client.AssignTeacherToSection(ctx, withSID(connect.NewRequest(&catalogv1.AssignTeacherToSectionRequest{
			SectionId: s1ID, TeacherId: tid,
		}), adminSID)); err != nil {
			t.Fatalf("AssignTeacherToSection s1/%s: %v", tid, err)
		}
	}
	if _, err := client.AssignTeacherToSection(ctx, withSID(connect.NewRequest(&catalogv1.AssignTeacherToSectionRequest{
		SectionId: s2ID, TeacherId: t3ID,
	}), adminSID)); err != nil {
		t.Fatalf("AssignTeacherToSection s2/t3: %v", err)
	}

	// List s1 — must contain t1 and t2 only.
	listResp, err := client.ListSectionTeachers(ctx, withSID(connect.NewRequest(&catalogv1.ListSectionTeachersRequest{
		SectionId: s1ID,
	}), adminSID))
	if err != nil {
		t.Fatalf("ListSectionTeachers(s1): %v", err)
	}
	for _, st := range listResp.Msg.GetSectionTeachers() {
		if st.GetTeacherId() == t3ID {
			t.Errorf("ListSectionTeachers(s1): contains t3 from s2 — must not")
		}
	}
	teachersInS1 := listResp.Msg.GetSectionTeachers()
	if len(teachersInS1) < 2 {
		t.Errorf("ListSectionTeachers(s1): got %d, want ≥2", len(teachersInS1))
	}
}

// TestSectionTeacher_DuplicateAssign_AlreadyExists verifies duplicate assignment returns CodeAlreadyExists.
func TestSectionTeacher_DuplicateAssign_AlreadyExists(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "sect-teacher-dup@catalog.test")
	client := newCatalogClient(nil)

	sectionID, _, _ := seedSectionFixture(t, adminSID)
	teacherID, _ := seedTeacherProfile(t, "sect-teacher-dup-t@catalog.test")

	// First assign — OK.
	if _, err := client.AssignTeacherToSection(ctx, withSID(connect.NewRequest(&catalogv1.AssignTeacherToSectionRequest{
		SectionId: sectionID, TeacherId: teacherID,
	}), adminSID)); err != nil {
		t.Fatalf("AssignTeacherToSection (first): %v", err)
	}

	// Second assign — AlreadyExists.
	_, err := client.AssignTeacherToSection(ctx, withSID(connect.NewRequest(&catalogv1.AssignTeacherToSectionRequest{
		SectionId: sectionID, TeacherId: teacherID,
	}), adminSID))
	assertConnectCode(t, err, connect.CodeAlreadyExists)
}

// TestSectionTeacher_BadFK_Teacher_InvalidArgument verifies assigning a non-existent teacher returns CodeInvalidArgument.
func TestSectionTeacher_BadFK_Teacher_InvalidArgument(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "sect-teacher-fk-teacher@catalog.test")
	client := newCatalogClient(nil)

	sectionID, _, _ := seedSectionFixture(t, adminSID)

	_, err := client.AssignTeacherToSection(ctx, withSID(connect.NewRequest(&catalogv1.AssignTeacherToSectionRequest{
		SectionId: sectionID,
		TeacherId: uuid.New().String(), // no teacher_profiles row
	}), adminSID))
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestSectionTeacher_BadFK_Section_InvalidArgument verifies assigning to non-existent section returns CodeInvalidArgument.
func TestSectionTeacher_BadFK_Section_InvalidArgument(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "sect-teacher-fk-section@catalog.test")
	client := newCatalogClient(nil)

	teacherID, _ := seedTeacherProfile(t, "sect-teacher-fk-s-t@catalog.test")

	_, err := client.AssignTeacherToSection(ctx, withSID(connect.NewRequest(&catalogv1.AssignTeacherToSectionRequest{
		SectionId: uuid.New().String(), // non-existent
		TeacherId: teacherID,
	}), adminSID))
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestSectionTeacher_Remove_AbsentAssociation_NotFound verifies CodeNotFound on absent remove.
func TestSectionTeacher_Remove_AbsentAssociation_NotFound(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "sect-teacher-remove-absent@catalog.test")
	client := newCatalogClient(nil)

	_, err := client.RemoveTeacherFromSection(ctx, withSID(connect.NewRequest(&catalogv1.RemoveTeacherFromSectionRequest{
		SectionId: uuid.New().String(),
		TeacherId: uuid.New().String(),
	}), adminSID))
	assertConnectCode(t, err, connect.CodeNotFound)
}

// ── Section authz ─────────────────────────────────────────────────────────────

// ── ListSections filtering ────────────────────────────────────────────────────

// TestListSections_FilterByCourseID verifies that course_id filter returns only sections for that course.
func TestListSections_FilterByCourseID(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "sect-filter-course@catalog.test")
	client := newCatalogClient(nil)

	// Create two courses and one section for each.
	s1ID, c1ID, _ := seedSectionFixture(t, adminSID)
	s2ID, c2ID, _ := seedSectionFixture(t, adminSID)

	// Filter by c1ID — must include s1, must exclude s2.
	resp, err := client.ListSections(ctx, withSID(connect.NewRequest(&catalogv1.ListSectionsRequest{
		CourseId: &c1ID,
	}), adminSID))
	if err != nil {
		t.Fatalf("ListSections(course_id=%s): %v", c1ID, err)
	}

	foundS1, foundS2 := false, false
	for _, s := range resp.Msg.GetSections() {
		switch s.GetId() {
		case s1ID:
			foundS1 = true
		case s2ID:
			foundS2 = true
		}
	}
	if !foundS1 {
		t.Errorf("ListSections by course_id: section %s (for c1) not found", s1ID)
	}
	if foundS2 {
		t.Errorf("ListSections by course_id: section %s (for c2) incorrectly included when filtering by c1", s2ID)
	}

	// Verify all returned sections belong to c1ID.
	for _, s := range resp.Msg.GetSections() {
		if s.GetCourseId() != c1ID {
			t.Errorf("ListSections by course_id: section %s has course_id %q, want %q", s.GetId(), s.GetCourseId(), c1ID)
		}
	}
	_ = c2ID
}

// TestListSections_FilterByAcademicPeriodID verifies that academic_period_id filter returns only sections for that period.
func TestListSections_FilterByAcademicPeriodID(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "sect-filter-period@catalog.test")
	client := newCatalogClient(nil)

	s1ID, _, p1ID := seedSectionFixture(t, adminSID)
	s2ID, _, p2ID := seedSectionFixture(t, adminSID)

	// Filter by p1ID — must include s1, must exclude s2.
	resp, err := client.ListSections(ctx, withSID(connect.NewRequest(&catalogv1.ListSectionsRequest{
		AcademicPeriodId: &p1ID,
	}), adminSID))
	if err != nil {
		t.Fatalf("ListSections(academic_period_id=%s): %v", p1ID, err)
	}

	foundS1, foundS2 := false, false
	for _, s := range resp.Msg.GetSections() {
		switch s.GetId() {
		case s1ID:
			foundS1 = true
		case s2ID:
			foundS2 = true
		}
	}
	if !foundS1 {
		t.Errorf("ListSections by academic_period_id: section %s (for p1) not found", s1ID)
	}
	if foundS2 {
		t.Errorf("ListSections by academic_period_id: section %s (for p2) incorrectly included when filtering by p1", s2ID)
	}

	// Verify all returned sections belong to p1ID.
	for _, s := range resp.Msg.GetSections() {
		if s.GetAcademicPeriodId() != p1ID {
			t.Errorf("ListSections by academic_period_id: section %s has academic_period_id %q, want %q", s.GetId(), s.GetAcademicPeriodId(), p1ID)
		}
	}
	_ = p2ID
}

// TestListSections_NoFilter_ReturnsAll verifies that omitting all filters returns all live sections
// (at least those seeded by this test).
func TestListSections_NoFilter_ReturnsAll(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "sect-filter-all@catalog.test")
	client := newCatalogClient(nil)

	s1ID, _, _ := seedSectionFixture(t, adminSID)
	s2ID, _, _ := seedSectionFixture(t, adminSID)

	resp, err := client.ListSections(ctx, withSID(connect.NewRequest(&catalogv1.ListSectionsRequest{}), adminSID))
	if err != nil {
		t.Fatalf("ListSections (no filter): %v", err)
	}

	seen := make(map[string]bool)
	for _, s := range resp.Msg.GetSections() {
		seen[s.GetId()] = true
	}
	if !seen[s1ID] {
		t.Errorf("ListSections (no filter): section %s missing", s1ID)
	}
	if !seen[s2ID] {
		t.Errorf("ListSections (no filter): section %s missing", s2ID)
	}
}

// TestListSections_MalformedCourseID_InvalidArgument verifies that a malformed course_id UUID
// returns CodeInvalidArgument.
func TestListSections_MalformedCourseID_InvalidArgument(t *testing.T) {
	ctx := context.Background()
	adminSID := catalogSeedAdminSession(t, "sect-filter-bad-uuid@catalog.test")
	client := newCatalogClient(nil)

	badUUID := "not-a-uuid"
	_, err := client.ListSections(ctx, withSID(connect.NewRequest(&catalogv1.ListSectionsRequest{
		CourseId: &badUUID,
	}), adminSID))
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestSection_NonAdmin_PermissionDenied verifies non-admin receives CodePermissionDenied on section procedures.
func TestSection_NonAdmin_PermissionDenied(t *testing.T) {
	ctx := context.Background()
	_, studentSID := seedUserWithSession(t, "sect-nonadmin@catalog.test", "student")
	client := newCatalogClient(nil)

	_, err := client.ListSections(ctx, withSID(connect.NewRequest(&catalogv1.ListSectionsRequest{}), studentSID))
	assertConnectCode(t, err, connect.CodePermissionDenied)

	_, err = client.AssignTeacherToSection(ctx, withSID(connect.NewRequest(&catalogv1.AssignTeacherToSectionRequest{
		SectionId: uuid.New().String(),
		TeacherId: uuid.New().String(),
	}), studentSID))
	assertConnectCode(t, err, connect.CodePermissionDenied)
}
