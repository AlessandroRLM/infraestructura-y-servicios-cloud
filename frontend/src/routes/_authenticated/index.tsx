import { createFileRoute, redirect } from "@tanstack/react-router";
import { readPreferredArea, resolveEntryTarget } from "@/features/auth";

export const Route = createFileRoute("/_authenticated/")({
  // Redirect immediately — never render a component.
  // The target is resolved from the session's eligibility (injected by the
  // parent _authenticated/route.tsx beforeLoad) and the stored area preference.
  beforeLoad: ({ context }) => {
    const target = resolveEntryTarget(context.eligibility, readPreferredArea());
    throw redirect({ to: target });
  },
});
