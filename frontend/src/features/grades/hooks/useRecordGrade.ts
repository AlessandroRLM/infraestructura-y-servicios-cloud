import { useMutation } from "@connectrpc/connect-query";
import type { Grade } from "@/gen/grades/v1/grades_pb";
import { GradesService } from "@/gen/grades/v1/grades_pb";

export interface RecordGradeParams {
  evaluationId: string;
  sectionEnrollmentId: string;
  /** Decimal string in [1.0, 7.0], e.g. "5.5". */
  value: string;
  /**
   * Expected version for optimistic locking.
   * Pass undefined for first-time grade creation; pass the current version when correcting.
   */
  expectedVersion?: number | undefined;
}

export interface UseRecordGradeResult {
  /** Issues a RecordGrade RPC for a single cell. Throws on error. */
  record: (params: RecordGradeParams) => Promise<Grade>;
  /** True while any mutation is in flight. */
  isPending: boolean;
}

/**
 * Issues RecordGrade calls — for callers who hold grades.write but NOT grades.override.
 * Each call is independent; the caller fans out per edited cell and handles partial failure.
 *
 * @returns record function + isPending flag.
 */
export function useRecordGrade(): UseRecordGradeResult {
  const mutation = useMutation(GradesService.method.recordGrade);

  const record = async (params: RecordGradeParams): Promise<Grade> => {
    const response = await mutation.mutateAsync({
      evaluationId: params.evaluationId,
      sectionEnrollmentId: params.sectionEnrollmentId,
      value: params.value,
      expectedVersion: params.expectedVersion,
    });
    if (!response.grade) {
      throw new Error("RecordGrade response missing grade");
    }
    return response.grade;
  };

  return {
    record,
    isPending: mutation.isPending,
  };
}
