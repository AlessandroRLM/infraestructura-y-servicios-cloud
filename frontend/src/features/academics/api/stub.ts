import type { ProgramCoursesSource } from "./rpc";

export function stubProgramCoursesSource(
  overrides?: Partial<ProgramCoursesSource>,
): ProgramCoursesSource {
  return {
    listProgramCourses: async () => [],
    ...overrides,
  };
}
