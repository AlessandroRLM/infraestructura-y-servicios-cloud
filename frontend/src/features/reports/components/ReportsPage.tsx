import type { ReactNode } from "react";
import { useEffect } from "react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import type { SessionState } from "@/features/auth";
import { hasPermission, useSession } from "@/features/auth";
import { Route } from "@/routes/_authenticated/admin/reports";
import type { ReportTab } from "../schemas/reportsSearch";
import { ProgramSummaryReportView } from "./ProgramSummaryReportView";
import { SectionGradeReportView } from "./SectionGradeReportView";
import { SectionOccupancyReportView } from "./SectionOccupancyReportView";
import { StudentRecordReportView } from "./StudentRecordReportView";

// Tab label map — ordered by display priority.
const TAB_CONFIG: {
  value: ReportTab;
  label: string;
  requiresPermission: "reports.read" | "users.manage";
}[] = [
  {
    value: "section-grade",
    label: "Calificaciones por Sección",
    requiresPermission: "reports.read",
  },
  {
    value: "occupancy",
    label: "Ocupación por Período",
    requiresPermission: "users.manage",
  },
  {
    value: "program-summary",
    label: "Resumen de Programa",
    requiresPermission: "users.manage",
  },
  {
    value: "student-record",
    label: "Expediente de Alumno",
    requiresPermission: "users.manage",
  },
];

/**
 * Returns the subset of tabs the given session may see.
 * Pure function — no hooks, safe to call in both render and effects.
 */
export function getVisibleTabs(session: SessionState) {
  return TAB_CONFIG.filter((t) => hasPermission(session, t.requiresPermission));
}

/**
 * Resolves the active tab accounting for permission gating.
 * If `requestedTab` is visible it is returned unchanged; otherwise the first
 * visible tab is returned. Falls back to `requestedTab` when the session has
 * no visible tabs (permission-empty state — the shell handles that case).
 */
export function resolveActiveTab(
  session: SessionState,
  requestedTab: ReportTab,
): ReportTab {
  const visible = getVisibleTabs(session);
  if (visible.length === 0) return requestedTab;
  return visible.some((t) => t.value === requestedTab)
    ? requestedTab
    : visible[0]!.value;
}

interface ReportsTabShellProps {
  activeTab: ReportTab;
  onTabChange: (tab: ReportTab) => void;
  /**
   * Optional render function for the active tab's content.
   * When provided, called with the active tab value to produce the panel content.
   * Falls back to a placeholder when not provided (used in isolated shell tests).
   */
  renderTabContent?: (tab: ReportTab) => ReactNode;
}

/**
 * Pure tab shell component — receives active tab, handler, and optional content renderer as props.
 * Reads permissions from session context.
 * Exported for isolated unit tests (no router dependency).
 *
 * `activeTab` MUST already be the permission-resolved tab (supplied by
 * ReportsPage via resolveActiveTab). The shell renders it directly so the
 * correct tab is visible on the first render without an extra re-render.
 *
 * `renderTabContent` is optional: tests can omit it to avoid mounting route-
 * dependent view components (SectionGradeReportView, etc.). ReportsPage always
 * provides it.
 */
export function ReportsTabShell({
  activeTab,
  onTabChange,
  renderTabContent,
}: ReportsTabShellProps) {
  const session = useSession();

  // Compute visible tabs from session permissions.
  const visibleTabs = getVisibleTabs(session);

  // RF-1.4: zero usable tabs → permission-empty state.
  if (visibleTabs.length === 0) {
    return (
      <div className="flex flex-col gap-1">
        <h1 className="font-semibold text-2xl tracking-tight">Reportes</h1>
        <p className="text-muted-foreground text-sm">
          No tienes permisos para acceder a los reportes.
        </p>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-1">
        <h1 className="font-semibold text-2xl tracking-tight">Reportes</h1>
        <p className="text-muted-foreground text-sm">
          Generación de reportes en formato PDF.
        </p>
      </div>

      <Tabs
        value={activeTab}
        onValueChange={(v) => onTabChange(v as ReportTab)}
      >
        <TabsList>
          {visibleTabs.map((t) => (
            <TabsTrigger key={t.value} value={t.value}>
              {t.label}
            </TabsTrigger>
          ))}
        </TabsList>

        {/* Only the active tab panel is mounted (RF-2.7, lazy mounting). */}
        {visibleTabs.map((t) =>
          t.value === activeTab ? (
            <TabsContent key={t.value} value={t.value}>
              {renderTabContent ? (
                renderTabContent(t.value)
              ) : (
                <div className="py-8 text-center text-sm text-muted-foreground">
                  {t.label} — próximamente.
                </div>
              )}
            </TabsContent>
          ) : null,
        )}
      </Tabs>
    </div>
  );
}

/**
 * Route-connected entry point for /admin/reports.
 *
 * Reads search params from the URL and wires navigate.
 * Tab visibility and permission-empty state are handled by ReportsTabShell.
 *
 * RF-2.2: when the permission-fallback resolves to a different tab than the URL
 * param, the URL is corrected (replace: true so no extra history entry).
 * resolveActiveTab is the single source of truth — used here for URL sync AND
 * passed down as activeTab to ReportsTabShell so both agree on first render.
 */
export function ReportsPage() {
  const { tab, sectionId, periodId, programId, studentId, year } =
    Route.useSearch();
  const navigate = Route.useNavigate();
  const session = useSession();

  // Single resolution point — no duplicated filter/resolve logic in the shell.
  const resolvedTab = resolveActiveTab(session, tab);

  // Sync URL when the permission-fallback corrects the tab (RF-2.2).
  // Replace so it doesn't pollute the history stack.
  useEffect(() => {
    if (resolvedTab !== tab) {
      navigate({
        search: (prev) => ({ ...prev, tab: resolvedTab }),
        replace: true,
      });
    }
  }, [resolvedTab, tab, navigate]);

  const handleTabChange = (value: ReportTab) => {
    navigate({ search: (prev) => ({ ...prev, tab: value }) });
  };

  const handleSectionChange = (newSectionId: string) => {
    navigate({ search: (prev) => ({ ...prev, sectionId: newSectionId }) });
  };

  const handlePeriodChange = (newPeriodId: string) => {
    navigate({ search: (prev) => ({ ...prev, periodId: newPeriodId }) });
  };

  const handleProgramChange = (newProgramId: string) => {
    navigate({ search: (prev) => ({ ...prev, programId: newProgramId }) });
  };

  const handleYearChange = (newYear: number | undefined) => {
    navigate({ search: (prev) => ({ ...prev, year: newYear }) });
  };

  const handleStudentChange = (newStudentId: string) => {
    navigate({ search: (prev) => ({ ...prev, studentId: newStudentId }) });
  };

  // renderTabContent injects the real view components per tab.
  // Only the active tab panel is rendered (lazy mounting, RF-2.7).
  const renderTabContent = (activeTabValue: ReportTab): ReactNode => {
    switch (activeTabValue) {
      case "section-grade":
        return (
          <SectionGradeReportView
            sectionId={sectionId}
            isActive={resolvedTab === "section-grade"}
            onSectionChange={handleSectionChange}
          />
        );
      case "occupancy":
        return (
          <SectionOccupancyReportView
            periodId={periodId}
            isActive={resolvedTab === "occupancy"}
            onPeriodChange={handlePeriodChange}
          />
        );
      case "program-summary":
        return (
          <ProgramSummaryReportView
            programId={programId}
            year={year}
            isActive={resolvedTab === "program-summary"}
            onProgramChange={handleProgramChange}
            onYearChange={handleYearChange}
          />
        );
      case "student-record":
        return (
          <StudentRecordReportView
            studentId={studentId}
            isActive={resolvedTab === "student-record"}
            onStudentChange={handleStudentChange}
          />
        );
      default:
        return (
          <div className="py-8 text-center text-sm text-muted-foreground">
            Próximamente.
          </div>
        );
    }
  };

  // Pass resolvedTab (not raw tab) so the shell shows the correct tab on
  // first render without waiting for the useEffect URL correction above.
  return (
    <ReportsTabShell
      activeTab={resolvedTab}
      onTabChange={handleTabChange}
      renderTabContent={renderTabContent}
    />
  );
}
