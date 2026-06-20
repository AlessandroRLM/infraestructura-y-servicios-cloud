import { useMutation } from "@connectrpc/connect-query";
import type { Grade } from "@/gen/grades/v1/grades_pb";
import { GradesService } from "@/gen/grades/v1/grades_pb";

export interface OverrideGradeParams {
  evaluationId: string;
  sectionEnrollmentId: string;
  /** Decimal string in [1.0, 7.0], e.g. "5.5". */
  value: string;
  /**
   * Expected version for optimistic locking.
   * Pass undefined for first-time override creation; pass the current version when correcting.
   */
  expectedVersion?: number | undefined;
}

export interface UseOverrideGradeResult {
  /** Issues an OverrideGrade RPC for a single cell. Throws on error. */
  override: (params: OverrideGradeParams) => Promise<Grade>;
  /** True while any mutation is in flight. */
  isPending: boolean;
}

/**
 * Issues OverrideGrade calls — for callers who hold grades.override.
 * No section_teachers check on the backend; full audit trail is recorded.
 * Each call is independent; the caller fans out per edited cell.
 *
 * @returns override function + isPending flag.
 */
export function useOverrideGrade(): UseOverrideGradeResult {
  const mutation = useMutation(GradesService.method.overrideGrade);

  const override = async (params: OverrideGradeParams): Promise<Grade> => {
    const response = await mutation.mutateAsync({
      evaluationId: params.evaluationId,
      sectionEnrollmentId: params.sectionEnrollmentId,
      value: params.value,
      expectedVersion: params.expectedVersion,
    });
    if (!response.grade) {
      throw new Error("OverrideGrade response missing grade");
    }
    return response.grade;
  };

  return {
    override,
    isPending: mutation.isPending,
  };
}
