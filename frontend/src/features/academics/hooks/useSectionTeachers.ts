import { useQuery } from "@connectrpc/connect-query";
import { CatalogService } from "@/gen/catalog/v1/catalog_pb";

/**
 * Fetches the list of teachers assigned to a section via ListSectionTeachers.
 */
export function useSectionTeachers(sectionId: string) {
  const result = useQuery(CatalogService.method.listSectionTeachers, {
    sectionId,
  });

  return {
    sectionTeachers: result.data?.sectionTeachers ?? [],
    isLoading: result.isLoading,
    isError: result.isError,
    refetch: result.refetch,
  };
}
