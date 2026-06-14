import { Pencil, Plus, RefreshCw, Trash2 } from "lucide-react";
import { useState } from "react";
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
import { hasPermission, useSession } from "@/features/auth";
import type { Course } from "@/gen/catalog/v1/catalog_pb";
import { useCourses } from "../hooks/useCourses";
import { CourseDialog } from "./CourseDialog";
import { DeleteCourseDialog } from "./DeleteCourseDialog";

interface CoursesTableProps {
  onCreateClick?: () => void;
}

export function CoursesTable({ onCreateClick }: CoursesTableProps) {
  const session = useSession();
  const canManage = hasPermission(session, "catalog.manage");
  const { courses, isLoading, isError, refetch } = useCourses();

  const [editCourse, setEditCourse] = useState<Course | undefined>(undefined);
  const [deleteCourse, setDeleteCourse] = useState<Course | undefined>(
    undefined,
  );

  if (isLoading) {
    return (
      <div
        role="status"
        className="space-y-2"
        aria-busy="true"
        aria-label="Cargando asignaturas"
      >
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-10 w-full" />
        ))}
      </div>
    );
  }

  if (isError) {
    return (
      <div className="rounded-md border border-destructive/50 p-4" role="alert">
        <p className="text-destructive text-sm font-medium">
          No se pudo cargar la lista de asignaturas.
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
    );
  }

  if (courses.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-4 rounded-md border border-dashed p-12 text-center">
        <p className="text-muted-foreground text-sm">
          Todavía no hay asignaturas
        </p>
        {canManage && (
          <Button onClick={onCreateClick} className="gap-2">
            <Plus className="size-4" aria-hidden />
            Crear asignatura
          </Button>
        )}
      </div>
    );
  }

  return (
    <>
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Código</TableHead>
              <TableHead>Nombre</TableHead>
              <TableHead>Créditos</TableHead>
              <TableHead>Creado</TableHead>
              {canManage && <TableHead className="w-[120px]" />}
            </TableRow>
          </TableHeader>
          <TableBody>
            {courses.map((course) => (
              <TableRow key={course.id}>
                <TableCell className="font-medium">{course.code}</TableCell>
                <TableCell>{course.name}</TableCell>
                <TableCell>{course.credits}</TableCell>
                <TableCell className="text-muted-foreground text-sm">
                  {course.createdAt
                    ? new Date(course.createdAt).toLocaleDateString("es-CL")
                    : "—"}
                </TableCell>
                {canManage && (
                  <TableCell className="flex gap-2 justify-end">
                    <Button
                      variant="ghost"
                      size="sm"
                      className="gap-1"
                      onClick={() => setEditCourse(course)}
                      aria-label={`Editar ${course.code}`}
                    >
                      <Pencil className="size-4" aria-hidden />
                      Editar
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="gap-1 text-destructive hover:text-destructive"
                      onClick={() => setDeleteCourse(course)}
                      aria-label={`Eliminar ${course.code}`}
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

      {editCourse && (
        <CourseDialog
          open={true}
          onOpenChange={(open) => {
            if (!open) setEditCourse(undefined);
          }}
          course={editCourse}
        />
      )}

      {deleteCourse && (
        <DeleteCourseDialog
          open={true}
          onOpenChange={(open) => {
            if (!open) setDeleteCourse(undefined);
          }}
          course={deleteCourse}
        />
      )}
    </>
  );
}
