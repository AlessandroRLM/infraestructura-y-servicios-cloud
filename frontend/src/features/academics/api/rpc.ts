import { createClient, type Transport } from "@connectrpc/connect";
import type {
  Course,
  Program,
  ProgramCourse,
} from "@/gen/catalog/v1/catalog_pb";
import { CatalogService } from "@/gen/catalog/v1/catalog_pb";

export interface ProgramsSource {
  listPrograms(): Promise<Program[]>;
}

export function createRpcProgramsSource(transport: Transport): ProgramsSource {
  const client = createClient(CatalogService, transport);
  return {
    async listPrograms() {
      const res = await client.listPrograms({});
      return res.programs;
    },
  };
}

export interface CoursesSource {
  listCourses(): Promise<Course[]>;
}

export function createRpcCoursesSource(transport: Transport): CoursesSource {
  const client = createClient(CatalogService, transport);
  return {
    async listCourses() {
      const res = await client.listCourses({});
      return res.courses;
    },
  };
}

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
