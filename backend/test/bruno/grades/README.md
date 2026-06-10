# grades/

Placeholder — the grades slice (proto/grades/v1/grades.proto) is not yet merged into
the main branch. Bruno request files for GradesService RPCs will be added here once
the slice lands.

When the slice is available, add requests for:
- RecordGrade (admin/teacher, enrollment.manage or grades.record)
- GetGrade / ListGrades (admin)
- Student self-read equivalents if exposed

The section_enrollment flow sets section inscriptions to in_progress; the grades slice
transitions them to passed or failed. The connection point is the SectionEnrollment.id
field captured in variables se_a_final_id / se_b_id.
