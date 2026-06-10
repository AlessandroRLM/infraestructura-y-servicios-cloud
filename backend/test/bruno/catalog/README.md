# catalog/

Bruno smoke tests for the `CatalogService` Connect-RPC slice.

## Flow overview

The folder is self-contained: it sets up its own admin and student sessions, creates all
required catalog entities using run-scoped unique codes (prefixed `cat_`), and tears down
throwaway entities. It does not rely on the `grades/` or `section_enrollment/` folders
having run first.

### Setup chain (01–06)

| # | Step | Captures |
|---|------|----------|
| 01–03 | Admin password reset + login | `admin_sid`; seeds `cat_prog_code`, `cat_course_code`, `cat_prog2_code`, `cat_course2_code` |
| 04–06 | Student A password reset + login | `cat_student_a_sid` (used for permission denial test 53) |

### Happy path — Program CRUD (07–12)

| # | Step | Captures / Key assertions |
|---|------|---------------------------|
| 07 | CreateProgram | `cat_prog_id`; asserts `code`, `name` |
| 08 | GetProgram | asserts `id` round-trips |
| 09 | UpdateProgram | asserts `name` updated |
| 10 | ListPrograms | asserts `programs` is array; created program is findable |
| 11 | CreateProgram (throwaway) | `cat_prog2_id` |
| 12 | DeleteProgram (throwaway) | 200; `cat_prog_id` preserved for dependents |

### Happy path — Course CRUD (13–18)

| # | Step | Captures / Key assertions |
|---|------|---------------------------|
| 13 | CreateCourse | `cat_course_id`; asserts `credits=4` |
| 14 | GetCourse | asserts `credits` round-trips |
| 15 | UpdateCourse | asserts `credits=6`, name updated |
| 16 | ListCourses | asserts `courses` is array; created course is findable |
| 17 | CreateCourse (throwaway) | `cat_course2_id` |
| 18 | DeleteCourse (throwaway) | 200; `cat_course_id` preserved |

### Happy path — AcademicPeriod CRUD (19–24)

| # | Step | Key assertions |
|---|------|----------------|
| 19 | CreateAcademicPeriod (year=2030, term=1) | `cat_period_id`; asserts `year=2030`, `term=1` |
| 20 | GetAcademicPeriod | asserts `id` round-trips |
| 21 | UpdateAcademicPeriod | asserts `endDate` updated |
| 22 | ListAcademicPeriods | asserts `academicPeriods` is array; created period is findable |
| 23 | CreateAcademicPeriod (throwaway, year=2031, term=2) | `cat_period2_id` |
| 24 | DeleteAcademicPeriod (throwaway) | 200; `cat_period_id` preserved |

### Happy path — ProgramQuota CRUD (25–30)

| # | Step | Key assertions |
|---|------|----------------|
| 25 | CreateProgramQuota (year=2030, quota=50) | `cat_quota_id`; asserts `programId`, `admissionQuota=50` |
| 26 | GetProgramQuota | asserts `id`, `year` round-trip |
| 27 | UpdateProgramQuota | asserts `admissionQuota=75` |
| 28 | ListProgramQuotas | asserts `programQuotas` is array; quota is findable |
| 29 | CreateProgramQuota (throwaway, year=2029) | `cat_quota2_id` |
| 30 | DeleteProgramQuota (throwaway) | 200 |

### Happy path — ProgramCourse M:N (31–34)

| # | Step | Key assertions |
|---|------|----------------|
| 31 | AddCourseToProgram | asserts `programId`, `courseId` |
| 32 | ListProgramCourses | asserts `programCourses` is array; association is findable |
| 33 | RemoveCourseFromProgram | 200 (empty response) |
| 34 | AddCourseToProgram (re-add) | asserts `programId`; demonstrates re-insert after hard-delete |

### Happy path — Section CRUD (35–38)

| # | Step | Key assertions |
|---|------|----------------|
| 35 | CreateSection (seatCapacity=30) | `cat_section_id`; asserts `courseId`, `seatCapacity=30` |
| 36 | GetSection | asserts `id`, `seatCapacity` round-trip |
| 37 | UpdateSection | asserts `seatCapacity=45` |
| 38 | ListSections (filtered by courseId) | asserts `sections` is array; created section is findable |

### Happy path — SectionTeacher M:N (39–44)

| # | Step | Key assertions |
|---|------|----------------|
| 39 | UpsertTeacherProfile (bootstrap admin) | FK prerequisite for 40 |
| 40 | AssignTeacherToSection | asserts `sectionId`, `teacherId` |
| 41 | ListSectionTeachers | asserts `sectionTeachers` is array; bootstrap admin is findable |
| 42 | RemoveTeacherFromSection | 200 (empty response) |
| 43 | CreateSection (throwaway, no teacher) | `cat_section2_id` |
| 44 | DeleteSection (throwaway) | 200; no dependents, succeeds |

### Negative cases (45–54)

| # | Scenario | Expected code | Derivation |
|---|----------|---------------|------------|
| 45 | Duplicate program code | `already_exists` | 23505 → ErrAlreadyExists → CodeAlreadyExists |
| 46 | Duplicate course code | `already_exists` | 23505 → ErrAlreadyExists → CodeAlreadyExists |
| 47 | GetProgram nonexistent UUID | `not_found` | pgx.ErrNoRows → ErrNotFound → CodeNotFound |
| 48 | GetProgram malformed UUID | `invalid_argument` | parseUUID failure → CodeInvalidArgument |
| 49 | CreateProgram with blank code | `invalid_argument` | service.go:63-65 code guard → ErrInvalidInput |
| 50 | CreateCourse with credits=0 | `invalid_argument` | service.go:110-112 credits guard → ErrInvalidInput |
| 51 | CreateAcademicPeriod with term=3 | `invalid_argument` | service.go:371-373 term guard → ErrInvalidInput |
| 52 | DeleteProgram with live dependents | `failed_precondition` | ErrHasDependents → CodeFailedPrecondition |
| 53 | Student calls CreateProgram | `permission_denied` | authz interceptor; student lacks catalog.manage |
| 54 | No session calls CreateCourse | `unauthenticated` | session interceptor rejects before authz |

## Assertion gotchas

- **Integer fields**: `credits`, `year`, `term`, `seatCapacity`, `admissionQuota` are JSON
  numbers serialized as integers — safe to assert with `value: "4"` directly.
- **bru `getVar()` in assertions**: `bru.getVar("x"): eq "true"` fails because bru coerces
  the expected `"true"` to boolean `true`, while `getVar` returns a string. Fix: put the
  membership check directly in the `expression` field as a JS expression
  (e.g. `res.body.programs.some(p => p.id === bru.getVar("cat_prog_id"))`).
- **AcademicPeriod uniqueness**: the DB has a unique constraint on `(year, term)`. Requests
  19 and 23 use a sfx-derived year (`3000 + sfx % 1000`) to avoid collisions across runs.
  The year is stored in `cat_period_year` and used in subsequent date fields.
- **Dynamic JSON bodies**: when request fields include computed values (year as int, date
  strings), the before-request script builds the full JSON body and stores it in a var
  (e.g. `cat_period_body`). The `data` field then uses `"{{cat_period_body}}"`.
- **protojson omits empty optional scalars**: `createdBy`, `updatedBy`, `deletedAt` are
  absent (not `""`) when not set. Do not assert their absence directly.
- **Response field names in lowerCamelCase**: `academicPeriods`, `programCourses`,
  `programQuotas`, `sectionTeachers` (repeated fields follow protojson naming).
- **Empty responses**: `DeleteProgramResponse`, `DeleteCourseResponse`,
  `RemoveCourseFromProgramResponse`, `RemoveTeacherFromSectionResponse` are empty messages —
  a 200 status assertion is sufficient.

## Prerequisites

- Stack running: `docker compose up -d` from the repo root.
- `seed.sh` executed and its `export` statements sourced (provides `student_a_email`
  and related vars).

## Run

```bash
eval "$(bash seed.sh | grep '^export')"
bru run catalog/ --env local --sandbox developer --disable-cookies \
  --env-var student_a_id="$BRUNO_STUDENT_A_ID" \
  --env-var student_a_email="$BRUNO_STUDENT_A_EMAIL" \
  --env-var open_period_id="$BRUNO_OPEN_PERIOD_ID"
```

A healthy run reports all 54 requests passed with a non-zero assertion count.

## Permission model

All `CatalogService` RPCs require the `catalog.manage` permission.

| Permission | Role | RPCs |
|---|---|---|
| `catalog.manage` | admin | All 31 RPCs (CreateProgram, UpdateProgram, GetProgram, ListPrograms, DeleteProgram, CreateCourse, UpdateCourse, GetCourse, ListCourses, DeleteCourse, CreateAcademicPeriod, UpdateAcademicPeriod, GetAcademicPeriod, ListAcademicPeriods, DeleteAcademicPeriod, CreateProgramQuota, UpdateProgramQuota, GetProgramQuota, ListProgramQuotas, DeleteProgramQuota, AddCourseToProgram, RemoveCourseFromProgram, ListProgramCourses, CreateSection, UpdateSection, GetSection, ListSections, DeleteSection, AssignTeacherToSection, RemoveTeacherFromSection, ListSectionTeachers) |

Students (e.g. `grades.view_own`) and unauthenticated callers receive `permission_denied`
and `unauthenticated` respectively.
