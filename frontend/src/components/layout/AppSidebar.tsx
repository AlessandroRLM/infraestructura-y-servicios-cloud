import { Link, linkOptions } from "@tanstack/react-router";
import {
  BookOpen,
  ChartColumn,
  ClipboardList,
  GraduationCap,
  ListChecks,
  PenLine,
  Users,
  Wallet,
} from "lucide-react";
import {
  hasPermission,
  LogoutButton,
  routePermissions,
  type SessionState,
  useSession,
} from "@/features/auth";
import { SwitchAreaControl } from "./SwitchAreaControl";

// No array annotation on purpose: `satisfies` validates the shape while
// preserving each linkOptions() return type so Link receives a typed `to`.
// The type alias below is inferred from the arrays — never declared by hand.

/** Admin area navigation — links to every /admin/* feature route. */
export const ADMIN_NAV = [
  {
    label: "Académico",
    icon: BookOpen,
    options: linkOptions({
      to: "/admin/academics",
      search: { tab: "programs", q: "", pageSize: 20 },
    }),
  },
  {
    label: "Inscripciones",
    icon: ClipboardList,
    options: linkOptions({
      to: "/admin/enrollments",
      search: { q: "", pageSize: 20 },
    }),
  },
  {
    label: "Secciones",
    icon: ListChecks,
    options: linkOptions({ to: "/admin/section-enrollments" }),
  },
  {
    label: "Notas",
    icon: PenLine,
    options: linkOptions({ to: "/admin/grades" }),
  },
  {
    label: "Reportes",
    icon: ChartColumn,
    options: linkOptions({
      to: "/admin/reports",
      search: {
        tab: "section-grade",
        sectionId: "",
        periodId: "",
        programId: "",
        studentId: "",
        year: undefined,
      },
    }),
  },
  {
    label: "Usuarios",
    icon: Users,
    options: linkOptions({
      to: "/admin/users",
      search: { q: "", pageSize: 20 },
    }),
  },
] as const;

/** Participant area navigation — links to every /app/* feature route. */
export const PARTICIPANT_NAV = [
  {
    label: "Mis notas",
    icon: PenLine,
    options: linkOptions({
      to: "/app/grades",
      search: { period: "", program: "", pageSize: 20 },
    }),
  },
  {
    label: "Mis matrículas",
    icon: Wallet,
    options: linkOptions({
      to: "/app/enrollments",
      search: { pageSize: 20 },
    }),
  },
  {
    label: "Mis secciones",
    icon: ListChecks,
    options: linkOptions({ to: "/app/section-enrollments" }),
  },
] as const;

/** A navigation entry from either area nav array. */
export type NavItem =
  | (typeof ADMIN_NAV)[number]
  | (typeof PARTICIPANT_NAV)[number];

// Visibility derives from the same ROUTE_PERMISSIONS map the route guards use:
// a link shows when its route is unguarded or the session holds one of the
// route's permissions (ANY). Single source of truth, no duplication.
function isVisible(session: SessionState, item: NavItem): boolean {
  const permissions = routePermissions(item.options.to);
  if (!permissions) return true;
  return permissions.some((permission) => hasPermission(session, permission));
}

// No display name on the wire (only email + roles); derive one from the local part.
function displayName(email: string): string {
  return email.split("@")[0] || "Usuario";
}

function initials(name: string): string {
  const parts = name.split(/[.\-_]+/).filter(Boolean);
  const raw = parts.length > 1 ? parts[0][0] + parts[1][0] : name.slice(0, 2);
  return raw.toUpperCase();
}

interface AppSidebarProps {
  /** Navigation entries for the current area. */
  nav: readonly NavItem[];
  /**
   * When `true`, renders a "Cambiar área" control in the sidebar footer.
   * Should be `true` only for dual-eligible users.
   */
  showSwitchArea?: boolean;
}

// Same background as the canvas + a border, so the shell reads as one space
// rather than fragmenting into "sidebar world" vs "content world".
export function AppSidebar({ nav, showSwitchArea = false }: AppSidebarProps) {
  const session = useSession();
  const isAuth = session.status === "authenticated";
  const name = isAuth ? displayName(session.email) : "";
  const role_label = isAuth
    ? ((session.roles[0] as string | undefined) ?? "")
    : "";
  const items = nav.filter((item) => isVisible(session, item));

  return (
    <aside className="flex w-64 shrink-0 flex-col border-r bg-background">
      <div className="flex h-16 items-center gap-2 border-b px-5">
        <span className="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
          <GraduationCap className="size-5" aria-hidden />
        </span>
        <span className="font-semibold tracking-tight">Académico</span>
      </div>

      <nav className="flex-1 space-y-1 overflow-y-auto p-3">
        {items.map((item) => (
          <Link
            key={item.options.to}
            {...item.options}
            activeProps={{ "data-active": "true" }}
            className="flex items-center gap-3 rounded-md px-3 py-2 text-muted-foreground text-sm transition-colors hover:bg-accent hover:text-foreground data-[active=true]:bg-accent data-[active=true]:font-medium data-[active=true]:text-foreground"
          >
            <item.icon className="size-4" aria-hidden />
            {item.label}
          </Link>
        ))}
      </nav>

      <div className="flex flex-col gap-3 border-t p-3">
        {showSwitchArea && <SwitchAreaControl />}
        <Link
          {...linkOptions({ to: "/profile" })}
          activeProps={{ "data-active": "true" }}
          className="flex items-center gap-3 rounded-md px-1 py-1 transition-colors hover:bg-accent data-[active=true]:bg-accent"
        >
          <span className="flex size-9 shrink-0 items-center justify-center rounded-full bg-muted font-medium text-foreground text-sm">
            {initials(name)}
          </span>
          <div className="min-w-0 flex-1">
            <div className="truncate font-medium text-sm">{name}</div>
            <div className="truncate text-muted-foreground text-xs">
              {role_label}
            </div>
          </div>
        </Link>
        <LogoutButton className="w-full justify-start" />
      </div>
    </aside>
  );
}
