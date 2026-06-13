// Package authz contains shared authorization vocabulary: Permission typed constants,
// PermissionSet, the Policy interface, and context helpers. It is a pure-types leaf —
// it does not import internal/auth or internal/rbac, so both of those packages can
// import it without creating a cycle.
package authz

// Permission is a typed string that identifies a single action capability.
// Using a distinct type prevents plain string literals from being assigned
// to a Permission parameter without an explicit conversion, catching typos early.
type Permission string

// The 14 permission codes that correspond to the operations matrix defined in the
// system architecture. These constants are the single source of truth; seed migrations
// insert the same literal strings, and a test asserts parity between this slice and
// the seeded rows.
const (
	PermUsersManage              Permission = "users.manage"
	PermCatalogManage            Permission = "catalog.manage"
	PermEnrollmentManage         Permission = "enrollment.manage"
	PermSectionsEnroll           Permission = "sections.enroll"
	PermEnrollmentViewOwn        Permission = "enrollment.view_own"
	PermGradesWrite              Permission = "grades.write"
	PermGradesRead               Permission = "grades.read"
	PermGradesViewOwn            Permission = "grades.view_own"
	PermReportsRead              Permission = "reports.read"
	PermAuditRead                Permission = "audit.read"
	PermGradesOverride           Permission = "grades.override"
	PermProfileViewOwn           Permission = "profile.view_own"
	PermSectionEnrollmentViewOwn Permission = "section_enrollment.view_own"
	PermProfileEditOwn           Permission = "profile.edit_own"
)

// AllPermissions lists every defined permission in the order they appear above.
// It is used by the seed parity test to verify that typed constants match the
// set of permission codes inserted by the seed migrations.
var AllPermissions = []Permission{
	PermUsersManage,
	PermCatalogManage,
	PermEnrollmentManage,
	PermSectionsEnroll,
	PermEnrollmentViewOwn,
	PermGradesWrite,
	PermGradesRead,
	PermGradesViewOwn,
	PermReportsRead,
	PermAuditRead,
	PermGradesOverride,
	PermProfileViewOwn,
	PermSectionEnrollmentViewOwn,
	PermProfileEditOwn,
}
