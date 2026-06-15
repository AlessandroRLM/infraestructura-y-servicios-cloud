import type { Enrollment } from "@/gen/enrollment/v1/enrollment_pb";
import type { GradePeriod, OwnGrade } from "@/gen/grades/v1/grades_pb";
import type { OwnGradesSource } from "./rpc";

/**
 * Creates a test stub for {@link OwnGradesSource}.
 * Each handler defaults to returning an empty result; override only what the
 * test needs.
 */
export function makeOwnGradesStub(
  overrides: Partial<OwnGradesSource> = {},
): OwnGradesSource {
  return {
    async listOwnGrades() {
      return { grades: [] as OwnGrade[], nextPageToken: "" };
    },
    async listOwnGradePeriods() {
      return [] as GradePeriod[];
    },
    async listOwnEnrollments() {
      return [] as Enrollment[];
    },
    ...overrides,
  };
}
