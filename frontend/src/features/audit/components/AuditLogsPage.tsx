import { hasPermission, useSession } from "@/features/auth";
import { Route } from "@/routes/_authenticated/admin/audit";
import { useAuditActorNames } from "../hooks/useAuditActorNames";
import { useRecentAuditLogs } from "../hooks/useRecentAuditLogs";
import { AuditFiltersBar } from "./AuditFiltersBar";
import { AuditLogsTable } from "./AuditLogsTable";

/** Converts a YYYY-MM-DD date string to a start-of-day RFC3339 UTC timestamp. */
function toCreatedFrom(date: string): string {
  return date ? `${date}T00:00:00Z` : "";
}

/** Converts a YYYY-MM-DD date string to an end-of-day RFC3339 UTC timestamp. */
function toCreatedTo(date: string): string {
  return date ? `${date}T23:59:59Z` : "";
}

export function AuditLogsPage() {
  const session = useSession();

  if (!hasPermission(session, "audit.read")) {
    return null;
  }

  return <AuditLogsPageContent />;
}

function AuditLogsPageContent() {
  const { actorId, from, to } = Route.useSearch();
  const navigate = Route.useNavigate();

  const {
    logs,
    isLoading,
    isError,
    refetch,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useRecentAuditLogs({
    actorId: actorId || undefined,
    createdFrom: toCreatedFrom(from),
    createdTo: toCreatedTo(to),
    pageSize: 20,
  });

  const actorIds = [...new Set(logs.map((log) => log.actorId).filter(Boolean))];
  const actorNames = useAuditActorNames(actorIds);

  return (
    <div className="flex flex-col gap-6">
      <h1 className="font-semibold text-2xl tracking-tight">Auditoría</h1>

      <AuditFiltersBar
        actorId={actorId}
        from={from}
        to={to}
        onActorChange={(v) =>
          navigate({ search: (prev) => ({ ...prev, actorId: v }) })
        }
        onFromChange={(v) =>
          navigate({ search: (prev) => ({ ...prev, from: v }) })
        }
        onToChange={(v) => navigate({ search: (prev) => ({ ...prev, to: v }) })}
      />

      <AuditLogsTable
        logs={logs}
        isLoading={isLoading}
        isError={isError}
        hasNextPage={hasNextPage}
        isFetchingNextPage={isFetchingNextPage}
        actorNames={actorNames}
        onRefetch={refetch}
        onLoadMore={async () => {
          await fetchNextPage({ throwOnError: true });
        }}
      />
    </div>
  );
}
