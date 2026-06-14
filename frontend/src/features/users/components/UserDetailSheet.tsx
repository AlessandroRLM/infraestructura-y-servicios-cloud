import { useTransport } from "@connectrpc/connect-query";
import { RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Skeleton } from "@/components/ui/skeleton";
import { createRpcUsersDetailSource } from "../api/rpc";
import { useUserDetail } from "../hooks/useUserDetail";
import { UserStatusBadge } from "./UserStatusBadge";

interface UserDetailSheetProps {
  userId: string | undefined;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function UserDetailSheet({
  userId,
  open,
  onOpenChange,
}: UserDetailSheetProps) {
  const transport = useTransport();
  const source = createRpcUsersDetailSource(transport);

  const { iam, profile, student, teacher, quals, isNotFound } = useUserDetail(
    userId ?? "",
    source,
  );

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-full sm:max-w-md overflow-y-auto">
        <SheetHeader>
          <SheetTitle>Detalle de usuario</SheetTitle>
          <SheetDescription className="sr-only">
            Detalle del usuario
          </SheetDescription>
        </SheetHeader>

        {!userId ? null : isNotFound ? (
          <div className="flex flex-col gap-4 p-4">
            <p className="text-muted-foreground text-sm">
              El usuario no existe o fue eliminado.
            </p>
          </div>
        ) : (
          <div className="flex flex-col gap-6 p-4">
            <section className="flex flex-col gap-3">
              <h2 className="font-semibold text-sm">Información de cuenta</h2>
              {iam.isLoading ? (
                <div className="flex flex-col gap-2">
                  <Skeleton className="h-4 w-full" />
                  <Skeleton className="h-4 w-3/4" />
                  <Skeleton className="h-4 w-1/2" />
                </div>
              ) : iam.isError ? (
                <div
                  className="rounded-md border border-destructive/50 p-3"
                  role="alert"
                >
                  <p className="text-destructive text-sm">
                    No se pudo cargar la información del usuario.
                  </p>
                  <Button
                    variant="outline"
                    size="sm"
                    className="mt-2 gap-2"
                    onClick={() => iam.refetch()}
                  >
                    <RefreshCw className="size-4" aria-hidden />
                    Reintentar
                  </Button>
                </div>
              ) : iam.data?.user ? (
                <dl className="flex flex-col gap-2 text-sm">
                  <div>
                    <dt className="text-muted-foreground">Email</dt>
                    <dd>{iam.data.user.email}</dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">Nombre</dt>
                    <dd>
                      {iam.data.user.displayName?.length
                        ? iam.data.user.displayName
                        : iam.data.user.email}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">Roles</dt>
                    <dd>{iam.data.user.roles.join(", ") || "—"}</dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">Estado</dt>
                    <dd>
                      <UserStatusBadge status={iam.data.user.status} />
                    </dd>
                  </div>
                </dl>
              ) : null}
            </section>

            <Separator />

            <section className="flex flex-col gap-3">
              <h2 className="font-semibold text-sm">Perfil</h2>
              {profile.isLoading ? (
                <div className="flex flex-col gap-2">
                  <Skeleton className="h-4 w-full" />
                  <Skeleton className="h-4 w-3/4" />
                </div>
              ) : profile.isError ? (
                <div
                  className="rounded-md border border-destructive/50 p-3"
                  role="alert"
                >
                  <p className="text-destructive text-sm">
                    No se pudo cargar el perfil.
                  </p>
                  <Button
                    variant="outline"
                    size="sm"
                    className="mt-2 gap-2"
                    onClick={() => profile.refetch()}
                  >
                    <RefreshCw className="size-4" aria-hidden />
                    Reintentar
                  </Button>
                </div>
              ) : profile.data ? (
                <dl className="flex flex-col gap-2 text-sm">
                  <div>
                    <dt className="text-muted-foreground">Nombres</dt>
                    <dd>{profile.data.givenNames || "—"}</dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">Apellido paterno</dt>
                    <dd>{profile.data.lastNamePaternal || "—"}</dd>
                  </div>
                  {profile.data.phone && (
                    <div>
                      <dt className="text-muted-foreground">Teléfono</dt>
                      <dd>{profile.data.phone}</dd>
                    </div>
                  )}
                </dl>
              ) : null}
            </section>

            {iam.data?.user?.roles.includes("student") && (
              <>
                <Separator />
                <section className="flex flex-col gap-3">
                  <h2 className="font-semibold text-sm">
                    Información académica
                  </h2>
                  {student.isLoading ? (
                    <Skeleton className="h-4 w-1/2" />
                  ) : student.isError ? (
                    <div
                      className="rounded-md border border-destructive/50 p-3"
                      role="alert"
                    >
                      <p className="text-destructive text-sm">
                        No se pudo cargar la información académica.
                      </p>
                      <Button
                        variant="outline"
                        size="sm"
                        className="mt-2 gap-2"
                        onClick={() => student.refetch()}
                      >
                        <RefreshCw className="size-4" aria-hidden />
                        Reintentar
                      </Button>
                    </div>
                  ) : student.data ? (
                    <dl className="flex flex-col gap-2 text-sm">
                      <div>
                        <dt className="text-muted-foreground">
                          Año de ingreso
                        </dt>
                        <dd>{student.data.admissionYear}</dd>
                      </div>
                    </dl>
                  ) : null}
                </section>
              </>
            )}

            {iam.data?.user?.roles.includes("teacher") && (
              <>
                <Separator />
                <section className="flex flex-col gap-3">
                  <h2 className="font-semibold text-sm">Información docente</h2>
                  {teacher.isLoading ? (
                    <div className="flex flex-col gap-2">
                      <Skeleton className="h-4 w-full" />
                      <Skeleton className="h-4 w-3/4" />
                    </div>
                  ) : teacher.isError ? (
                    <div
                      className="rounded-md border border-destructive/50 p-3"
                      role="alert"
                    >
                      <p className="text-destructive text-sm">
                        No se pudo cargar la información docente.
                      </p>
                      <Button
                        variant="outline"
                        size="sm"
                        className="mt-2 gap-2"
                        onClick={() => teacher.refetch()}
                      >
                        <RefreshCw className="size-4" aria-hidden />
                        Reintentar
                      </Button>
                    </div>
                  ) : teacher.data ? (
                    <dl className="flex flex-col gap-2 text-sm">
                      <div>
                        <dt className="text-muted-foreground">Departamento</dt>
                        <dd>{teacher.data.department || "—"}</dd>
                      </div>
                      <div>
                        <dt className="text-muted-foreground">Título</dt>
                        <dd>{teacher.data.title || "—"}</dd>
                      </div>
                    </dl>
                  ) : null}

                  {!teacher.isLoading &&
                    !teacher.isError &&
                    quals.data &&
                    quals.data.length > 0 && (
                      <div className="flex flex-col gap-2">
                        <h3 className="text-sm font-medium text-muted-foreground">
                          Calificaciones
                        </h3>
                        <ul className="flex flex-col gap-1 text-sm">
                          {quals.data.map((q) => (
                            <li key={q.id}>
                              {q.degree} ({q.year})
                            </li>
                          ))}
                        </ul>
                      </div>
                    )}
                </section>
              </>
            )}
          </div>
        )}
      </SheetContent>
    </Sheet>
  );
}
