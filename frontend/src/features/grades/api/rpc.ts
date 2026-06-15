import { createClient, type Transport } from "@connectrpc/connect";
import type { Enrollment } from "@/gen/enrollment/v1/enrollment_pb";
import { EnrollmentService } from "@/gen/enrollment/v1/enrollment_pb";
import type { GradePeriod, OwnGrade } from "@/gen/grades/v1/grades_pb";
import { GradesService } from "@/gen/grades/v1/grades_pb";

/** Parameters for listing a student's own grades with optional server-side filters. */
export interface ListOwnGradesParams {
  pageSize: number;
  pageToken: string;
  /** UUID string; omit or empty string to return all periods. */
  academicPeriodId?: string;
  /** UUID string; omit or empty string to return all programs. */
  programId?: string;
}

/**
 * Source interface for student-facing grades data.
 * Injected into hooks so tests can substitute a stub without touching the transport layer.
 */
export interface OwnGradesSource {
  /**
   * Fetches one page of the student's own grades.
   * @returns grades array and the next page token (empty when no further pages exist).
   */
  listOwnGrades(
    params: ListOwnGradesParams,
  ): Promise<{ grades: OwnGrade[]; nextPageToken: string }>;

  /**
   * Fetches the distinct academic periods in which the student has grades.
   * Ordered most-recent first.
   */
  listOwnGradePeriods(): Promise<GradePeriod[]>;

  /**
   * Fetches the student's own enrollments for carrera filter options.
   * Uses AIP-158 keyset — pageSize 200 to cover the realistic max in one call.
   */
  listOwnEnrollments(): Promise<Enrollment[]>;
}

/**
 * Creates a live RPC-backed implementation of {@link OwnGradesSource} using
 * the provided Connect transport.
 */
export function createRpcOwnGradesSource(
  transport: Transport,
): OwnGradesSource {
  const gradesClient = createClient(GradesService, transport);
  const enrollmentClient = createClient(EnrollmentService, transport);

  return {
    async listOwnGrades({ pageSize, pageToken, academicPeriodId, programId }) {
      const res = await gradesClient.listOwnGrades({
        pageSize,
        pageToken,
        academicPeriodId: academicPeriodId || undefined,
        programId: programId || undefined,
      });
      return { grades: res.grades, nextPageToken: res.nextPageToken };
    },

    async listOwnGradePeriods() {
      const res = await gradesClient.listOwnGradePeriods({});
      return res.periods;
    },

    async listOwnEnrollments() {
      const res = await enrollmentClient.listOwnEnrollments({
        pageSize: 200,
        pageToken: "",
      });
      return res.enrollments;
    },
  };
}
