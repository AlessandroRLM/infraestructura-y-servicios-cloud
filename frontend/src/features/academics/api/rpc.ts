import { createClient, type Transport } from "@connectrpc/connect";
import type { Program } from "@/gen/catalog/v1/catalog_pb";
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
