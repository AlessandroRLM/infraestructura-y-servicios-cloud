import { Loader2, RefreshCw } from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useDebounce } from "@/core/hooks";
import { Route } from "@/routes/_authenticated/users";
import { SEARCH_DEBOUNCE_MS } from "../constants";
import { useUsersList } from "../hooks/useUsersList";
import { UserStatusBadge } from "./UserStatusBadge";

interface UsersTableProps {
  onRowClick: (userId: string) => void;
}

export function UsersTable({ onRowClick }: UsersTableProps) {
  const { q } = Route.useSearch();
  const navigate = Route.useNavigate();

  const [inputValue, setInputValue] = useState(q);
  const debouncedQ = useDebounce(inputValue, SEARCH_DEBOUNCE_MS);

  useEffect(() => {
    if (debouncedQ !== q) {
      navigate({ search: { q: debouncedQ } });
    }
  }, [debouncedQ, q, navigate]);

  const {
    users,
    isLoading,
    isError,
    refetch,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    isFetchNextPageError,
  } = useUsersList(q);

  useEffect(() => {
    if (isFetchNextPageError) {
      toast.error("No se pudieron cargar más usuarios.");
    }
  }, [isFetchNextPageError]);

  return (
    <div className="flex flex-col gap-4">
      <Input
        placeholder="Buscar por email o nombre..."
        value={inputValue}
        onChange={(e) => setInputValue(e.target.value)}
        className="max-w-sm"
      />

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
                  <TableCell>{user.roles.join(", ")}</TableCell>
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
            onClick={() => fetchNextPage()}
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
