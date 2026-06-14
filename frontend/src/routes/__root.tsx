import type { QueryClient } from "@tanstack/react-query";
import { createRootRouteWithContext, Outlet } from "@tanstack/react-router";
import { StatusScreen } from "@/components/StatusScreen";
import type { SessionSource } from "@/features/auth";

interface RouterContext {
  queryClient: QueryClient;
  sessionSource: SessionSource;
}

// Unmatched URLs (and any thrown notFound()) render here, outside the
// authenticated shell, so it works for logged-out visitors too.
function NotFoundScreen() {
  return (
    <StatusScreen
      code="404"
      title="Página no encontrada"
      description="La página que buscas no existe o fue movida."
    />
  );
}

// Infra failures thrown from beforeLoad land here, distinct from the clean
// logged-out redirect which never reaches the error boundary.
function AppError() {
  return (
    <div data-testid="app-error">
      <p>
        Algo salió mal. El servicio podría no estar disponible temporalmente.
      </p>
      <button type="button" onClick={() => window.location.reload()}>
        Reintentar
      </button>
    </div>
  );
}

export const Route = createRootRouteWithContext<RouterContext>()({
  component: () => <Outlet />,
  errorComponent: AppError,
  notFoundComponent: NotFoundScreen,
});
