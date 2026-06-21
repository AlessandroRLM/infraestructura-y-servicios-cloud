import { Loader2, Pencil, Plus, RefreshCw, Trash2, Users } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  academicPeriodLabel,
  useAcademicPeriods,
  useCourses,
  useSections,
} from "@/core/catalog";
import { PageSizeSelector, SearchInput } from "@/core/components";
import { hasPermission, useSession } from "@/features/auth";
import type { Section } from "@/gen/catalog/v1/catalog_pb";
import { Route } from "@/routes/_authenticated/admin/academics";
import { SEARCH_DEBOUNCE_MS } from "../constants";
import { DeleteSectionDialog } from "./DeleteSectionDialog";
import { SectionDialog } from "./SectionDialog";
import { SectionTeachersSheet } from "./SectionTeachersSheet";

interface SectionsTableProps {
  onCreateClick?: () => void;
}

export function SectionsTable({ onCreateClick }: SectionsTableProps) {
  const session = useSession();
  const canManage = hasPermission(session, "catalog.manage");

  const { q, pageSize } = Route.useSearch();
  const navigate = Route.useNavigate();

  const {
    sections,
    isLoading,
    isError,
    refetch,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useSections({}, pageSize);

  // Resolve course/period IDs to human-readable labels; fall back to the raw
  // ID when the catalog lists are still loading or the entry is missing.
  const { courses } = useCourses("", 100);
  const { periods } = useAcademicPeriods();
  const courseLabel = (id: string) => {
    const course = courses.find((c) => c.id === id);
    return course ? `${course.code} — ${course.name}` : id;
  };
  const periodLabel = (id: string) => {
    const period = periods.find((p) => p.id === id);
    return period ? academicPeriodLabel(period.year, period.term) : id;
  };

  const [editSection, setEditSection] = useState<Section | undefined>(
    undefined,
  );
  const [deleteSection, setDeleteSection] = useState<Section | undefined>(
    undefined,
  );
  const [teachersSection, setTeachersSection] = useState<Section | undefined>(
    undefined,
  );

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-3">
        <SearchInput
          value={q}
          onChange={(v) => navigate({ search: (prev) => ({ ...prev, q: v }) })}
          debounceMs={SEARCH_DEBOUNCE_MS}
          placeholder="Buscar secciones..."
          className="max-w-sm"
        />
        <PageSizeSelector
          value={pageSize}
          onChange={(n) =>
            navigate({ search: (prev) => ({ ...prev, pageSize: n }) })
          }
        />
      </div>

      {isLoading && (
        <div
          role="status"
          className="space-y-2"
          aria-busy="true"
          aria-label="Cargando secciones"
        >
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-10 w-full" />
          ))}
        </div>
      )}

      {!isLoading && isError && (
        <div
          className="rounded-md border border-destructive/50 p-4"
          role="alert"
        >
          <p className="text-destructive text-sm font-medium">
            No se pudo cargar la lista de secciones.
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-3 gap-2"
            onClick={() => refetch()}
          >
            <RefreshCw className="size-4" aria-hidden />
            Reintentar
          </Button>
        </div>
      )}

      {!isLoading && !isError && sections.length === 0 && (
        <div className="flex flex-col items-center justify-center gap-4 rounded-md border border-dashed p-12 text-center">
          <p className="text-muted-foreground text-sm">
            Todavía no hay secciones
          </p>
          {canManage && (
            <Button onClick={onCreateClick} className="gap-2">
              <Plus className="size-4" aria-hidden />
              Crear sección
            </Button>
          )}
        </div>
      )}

      {!isLoading && !isError && sections.length > 0 && (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Asignatura</TableHead>
                <TableHead>Período</TableHead>
                <TableHead>Capacidad</TableHead>
                <TableHead>Creado</TableHead>
                {canManage && <TableHead className="w-[200px]" />}
              </TableRow>
            </TableHeader>
            <TableBody>
              {sections.map((section) => (
                <TableRow key={section.id}>
                  <TableCell className="font-medium">
                    {courseLabel(section.courseId)}
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    {periodLabel(section.academicPeriodId)}
                  </TableCell>
                  <TableCell>{section.seatCapacity}</TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    {section.createdAt
                      ? new Date(section.createdAt).toLocaleDateString("es-CL")
                      : "—"}
                  </TableCell>
                  {canManage && (
                    <TableCell className="flex gap-2 justify-end">
                      <Button
                        variant="ghost"
                        size="sm"
                        className="gap-1"
                        onClick={() => setTeachersSection(section)}
                        aria-label={`Docentes sección ${section.id}`}
                      >
                        <Users className="size-4" aria-hidden />
                        Docentes
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="gap-1"
                        onClick={() => setEditSection(section)}
                        aria-label={`Editar sección ${section.id}`}
                      >
                        <Pencil className="size-4" aria-hidden />
                        Editar
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="gap-1 text-destructive hover:text-destructive"
                        onClick={() => setDeleteSection(section)}
                        aria-label={`Eliminar sección ${section.id}`}
                      >
                        <Trash2 className="size-4" aria-hidden />
                        Eliminar
                      </Button>
                    </TableCell>
                  )}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {hasNextPage && (
        <div className="flex justify-center">
          <Button
            variant="outline"
            onClick={async () => {
              try {
                await fetchNextPage({ throwOnError: true });
              } catch {
                toast.error("No se pudieron cargar más secciones.");
              }
            }}
            disabled={isFetchingNextPage}
            className="gap-2"
          >
            {isFetchingNextPage && (
              <Loader2
                data-icon="inline-start"
                className="animate-spin"
                aria-hidden
              />
            )}
            Cargar más
          </Button>
        </div>
      )}

      {editSection && (
        <SectionDialog
          open={true}
          onOpenChange={(open) => {
            if (!open) setEditSection(undefined);
          }}
          section={editSection}
        />
      )}

      {deleteSection && (
        <DeleteSectionDialog
          open={true}
          onOpenChange={(open) => {
            if (!open) setDeleteSection(undefined);
          }}
          section={deleteSection}
        />
      )}

      <SectionTeachersSheet
        section={teachersSection}
        open={teachersSection !== undefined}
        onOpenChange={(open) => {
          if (!open) setTeachersSection(undefined);
        }}
      />
    </div>
  );
}
