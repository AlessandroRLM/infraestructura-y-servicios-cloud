import { createClient, type Transport } from "@connectrpc/connect";
import type { ProgramCourse } from "@/gen/catalog/v1/catalog_pb";
import { CatalogService } from "@/gen/catalog/v1/catalog_pb";

export interface ProgramCoursesSource {
  listProgramCourses(programId: string): Promise<ProgramCourse[]>;
}

export function createRpcProgramCoursesSource(
  transport: Transport,
): ProgramCoursesSource {
  const client = createClient(CatalogService, transport);
  return {
    async listProgramCourses(programId) {
      const res = await client.listProgramCourses({ programId });
      return res.programCourses;
    },
  };
}
