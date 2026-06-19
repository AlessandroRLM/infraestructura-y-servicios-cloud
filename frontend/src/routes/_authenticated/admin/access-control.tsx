import { createFileRoute } from "@tanstack/react-router";
import { requireRoutePermission } from "@/features/auth";

function AccessControlPage() {
  return (
    <div className="flex flex-col gap-1">
      <h1 className="font-semibold text-2xl tracking-tight">
        Control de acceso
      </h1>
      <p className="text-muted-foreground">Módulo no disponible.</p>
    </div>
  );
}

export const Route = createFileRoute("/_authenticated/admin/access-control")({
  beforeLoad: ({ context }) => {
    requireRoutePermission(context.session, "/admin/access-control");
  },
  component: AccessControlPage,
});
