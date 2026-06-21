import { Button } from "@/components/ui/button";
import { useUsersList } from "@/features/users";

interface AuditFiltersBarProps {
  actorId: string;
  from: string;
  to: string;
  onActorChange: (actorId: string) => void;
  onFromChange: (from: string) => void;
  onToChange: (to: string) => void;
}

/**
 * Filter bar for the audit log feed.
 *
 * Provides date range (createdFrom / createdTo) and an actor dropdown.
 * Date values are stored as YYYY-MM-DD in the URL; the parent is responsible
 * for converting to RFC3339 before passing to the RPC.
 */
export function AuditFiltersBar({
  actorId,
  from,
  to,
  onActorChange,
  onFromChange,
  onToChange,
}: AuditFiltersBarProps) {
  const { users } = useUsersList("", 200);

  const hasFilters = actorId !== "" || from !== "" || to !== "";

  return (
    <div className="flex flex-wrap items-center gap-3">
      <div className="flex items-center gap-2">
        <label
          htmlFor="audit-from"
          className="text-muted-foreground text-sm whitespace-nowrap"
        >
          Desde
        </label>
        <input
          id="audit-from"
          type="date"
          value={from}
          onChange={(e) => onFromChange(e.target.value)}
          className="h-9 rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
        />
      </div>

      <div className="flex items-center gap-2">
        <label
          htmlFor="audit-to"
          className="text-muted-foreground text-sm whitespace-nowrap"
        >
          Hasta
        </label>
        <input
          id="audit-to"
          type="date"
          value={to}
          onChange={(e) => onToChange(e.target.value)}
          className="h-9 rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
        />
      </div>

      <div className="flex items-center gap-2">
        <label
          htmlFor="audit-actor"
          className="text-muted-foreground text-sm whitespace-nowrap"
        >
          Actor
        </label>
        <select
          id="audit-actor"
          value={actorId}
          onChange={(e) => onActorChange(e.target.value)}
          className="h-9 rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
        >
          <option value="">Todos</option>
          {users.map((user) => (
            <option key={user.id} value={user.id}>
              {user.displayName?.length ? user.displayName : user.email}
            </option>
          ))}
        </select>
      </div>

      {hasFilters && (
        <Button
          variant="ghost"
          size="sm"
          onClick={() => {
            onActorChange("");
            onFromChange("");
            onToChange("");
          }}
        >
          Limpiar filtros
        </Button>
      )}
    </div>
  );
}
