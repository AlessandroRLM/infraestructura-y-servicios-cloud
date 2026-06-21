import { Loader2, RefreshCw } from "lucide-react";
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
import { PageSizeSelector, SearchInput } from "@/core/components";
import { type Role, roleLabel } from "@/features/auth";
import { Route } from "@/routes/_authenticated/admin/users";
import { SEARCH_DEBOUNCE_MS } from "../constants";
import { useUsersList } from "../hooks/useUsersList";
import { UserStatusBadge } from "./UserStatusBadge";

interface UsersTableProps {
  onRowClick: (userId: string) => void;
}

export function UsersTable({ onRowClick }: UsersTableProps) {
  const { q, pageSize } = Route.useSearch();
  const navigate = Route.useNavigate();

  const {
    users,
    isLoading,
    isError,
    refetch,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useUsersList(q, pageSize);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-3">
        <SearchInput
          value={q}
          onChange={(v) => navigate({ search: (prev) => ({ ...prev, q: v }) })}
          debounceMs={SEARCH_DEBOUNCE_MS}
          placeholder="Buscar por email o nombre..."
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
          aria-busy="true"
          aria-label="Cargando usuarios"
          className="flex flex-col gap-2"
        >
          {Array.from({ length: 5 }).map((_, i) => (
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
            No se pudo cargar la lista de usuarios.
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

      {!isLoading && !isError && users.length === 0 && (
        <div className="flex flex-col items-center justify-center gap-4 rounded-md border border-dashed p-12 text-center">
          <p className="text-muted-foreground text-sm">
            No se encontraron usuarios.
          </p>
        </div>
      )}

      {!isLoading && !isError && users.length > 0 && (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Email</TableHead>
                <TableHead>Nombre</TableHead>
                <TableHead>Roles</TableHead>
                <TableHead>Estado</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {users.map((user) => (
                <TableRow
                  key={user.id}
                  onClick={() => onRowClick(user.id)}
                  className="cursor-pointer"
                >
                  <TableCell>{user.email}</TableCell>
                  <TableCell>
                    {user.displayName?.length ? user.displayName : user.email}
                  </TableCell>
                  <TableCell>
                    {user.roles
                      .map((role) => roleLabel(role as Role))
                      .join(", ")}
                  </TableCell>
                  <TableCell>
                    <UserStatusBadge status={user.status} />
                  </TableCell>
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
                toast.error("No se pudieron cargar más usuarios.");
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
    </div>
  );
}
