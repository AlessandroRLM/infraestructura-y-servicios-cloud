import type { CoursesSource, ProgramsSource } from "./rpc";

export function stubProgramsSource(
  overrides?: Partial<ProgramsSource>,
): ProgramsSource {
  return {
    listPrograms: async () => [],
    ...overrides,
  };
}

export function stubCoursesSource(
  overrides?: Partial<CoursesSource>,
): CoursesSource {
  return {
    listCourses: async () => [],
    ...overrides,
  };
}
