import type { ProgramsSource } from "./rpc";

export function stubProgramsSource(
  overrides?: Partial<ProgramsSource>,
): ProgramsSource {
  return {
    listPrograms: async () => [],
    ...overrides,
  };
}
