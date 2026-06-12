import type { QueryClient } from "@tanstack/react-query";
import { createRootRouteWithContext, Outlet } from "@tanstack/react-router";
import type { SessionSource } from "@/features/auth";

interface RouterContext {
  queryClient: QueryClient;
  sessionSource: SessionSource;
}

// Infra failures thrown from beforeLoad land here, distinct from the clean
// logged-out redirect which never reaches the error boundary.
function AppError() {
  return (
    <div data-testid="app-error">
      <p>Something went wrong. The service may be temporarily unavailable.</p>
      <button type="button" onClick={() => window.location.reload()}>
        Retry
      </button>
    </div>
  );
}

export const Route = createRootRouteWithContext<RouterContext>()({
  component: () => <Outlet />,
  errorComponent: AppError,
});
