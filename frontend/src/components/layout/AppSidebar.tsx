import { Link, linkOptions } from "@tanstack/react-router";
import {
  BookOpen,
  ChartColumn,
  ClipboardList,
  GraduationCap,
  LayoutDashboard,
  ListChecks,
  type LucideIcon,
  PenLine,
  ShieldCheck,
  Users,
} from "lucide-react";
import {
  hasPermission,
  LogoutButton,
  type Permission,
  type SessionState,
  useSession,
} from "@/features/auth";

interface NavMeta {
  label: string;
  icon: LucideIcon;
  // Visible if the session has ANY of these; undefined = always visible.
  permissions?: Permission[];
}

// No array annotation on purpose: `satisfies` validates the meta fields while
// preserving each linkOptions() return type. linkOptions checks `to` against
// the generated route tree at build time, so a non-existent path fails tsc.
const NAV = [
  { label: "Inicio", icon: LayoutDashboard, options: linkOptions({ to: "/" }) },
  {
    label: "Académico",
    icon: BookOpen,
    permissions: ["catalog.manage"],
    options: linkOptions({ to: "/academics" }),
  },
  {
    label: "Inscripciones",
    icon: ClipboardList,
    permissions: ["enrollment.manage"],
    options: linkOptions({ to: "/enrollments" }),
  },
  {
    label: "Secciones",
    icon: ListChecks,
    permissions: ["sections.enroll", "section_enrollment.view_own"],
    options: linkOptions({ to: "/section-enrollments" }),
  },
  {
    label: "Notas",
    icon: PenLine,
    permissions: ["grades.read", "grades.write", "grades.view_own"],
    options: linkOptions({ to: "/grades" }),
  },
  {
    label: "Reportes",
    icon: ChartColumn,
    permissions: ["reports.read"],
    options: linkOptions({ to: "/reports" }),
  },
  {
    label: "Usuarios",
    icon: Users,
    permissions: ["users.manage"],
    options: linkOptions({ to: "/users" }),
  },
  {
    label: "Control de acceso",
    icon: ShieldCheck,
    permissions: ["users.manage"],
    options: linkOptions({ to: "/access-control" }),
  },
] satisfies readonly (NavMeta & { options: object })[];

type NavItem = (typeof NAV)[number];

const ROLE_LABELS: Record<string, string> = {
  admin: "Administrador",
  teacher: "Profesor",
  student: "Estudiante",
};
// Most privileged role first, so a multi-role user shows their highest cargo.
const ROLE_PRIORITY = ["admin", "teacher", "student"];

function isVisible(session: SessionState, item: NavItem): boolean {
  const permissions = "permissions" in item ? item.permissions : undefined;
  if (!permissions) {
    return true;
  }
  return permissions.some((p) => hasPermission(session, p));
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

function roleLabel(roles: string[]): string {
  const primary = ROLE_PRIORITY.find((r) => roles.includes(r)) ?? roles[0];
  return primary ? (ROLE_LABELS[primary] ?? primary) : "";
}

// Same background as the canvas + a border, so the shell reads as one space
// rather than fragmenting into "sidebar world" vs "content world".
export function AppSidebar() {
  const session = useSession();
  const isAuth = session.status === "authenticated";
  const name = isAuth ? displayName(session.email) : "";
  const role = isAuth ? roleLabel(session.roles) : "";
  const items = NAV.filter((item) => isVisible(session, item));

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
            key={String(item.options.to)}
            {...item.options}
            activeOptions={{ exact: item.options.to === "/" }}
            activeProps={{ "data-active": "true" }}
            className="flex items-center gap-3 rounded-md px-3 py-2 text-muted-foreground text-sm transition-colors hover:bg-accent hover:text-foreground data-[active=true]:bg-accent data-[active=true]:font-medium data-[active=true]:text-foreground"
          >
            <item.icon className="size-4" aria-hidden />
            {item.label}
          </Link>
        ))}
      </nav>

      <div className="flex flex-col gap-3 border-t p-3">
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
            <div className="truncate text-muted-foreground text-xs">{role}</div>
          </div>
        </Link>
        <LogoutButton className="w-full justify-start" />
      </div>
    </aside>
  );
}
