import { createFileRoute } from "@tanstack/react-router";
import { StatusScreen } from "@/components/StatusScreen";

// Renders inside the authenticated shell on purpose: the user is logged in and
// stays in the app (sidebar visible) to navigate elsewhere. No permission guard
// here — that would loop, since denied routes redirect into this one.
function ForbiddenScreen() {
  return (
    <StatusScreen
      code="403"
      title="No tienes acceso a esta sección"
      description="Tu cuenta no cuenta con los permisos necesarios para ver esta página. Si crees que es un error, contacta a un administrador."
    />
  );
}

export const Route = createFileRoute("/_authenticated/forbidden")({
  component: ForbiddenScreen,
});
