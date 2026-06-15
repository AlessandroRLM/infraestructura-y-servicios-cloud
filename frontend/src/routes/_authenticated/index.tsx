import { createFileRoute, redirect } from "@tanstack/react-router";
import { readPreferredArea, resolveEntryTarget } from "@/features/auth";

export const Route = createFileRoute("/_authenticated/")({
  // Redirect immediately — never render a component.
  // The target is resolved from the session's eligibility (injected by the
  // parent _authenticated/route.tsx beforeLoad) and the stored area preference.
  beforeLoad: ({ context }) => {
    const target = resolveEntryTarget(context.eligibility, readPreferredArea());
    // `href` bypasses TanStack Router's registered-route type check so this
    // compiles before /admin, /app, and /choose-area route files are added in
    // Slices 2 and 3. All values of EntryTarget are valid URLs that will exist
    // after the full migration; the type assertion is removed when the route
    // files land (WU-4/5/6 in Slice 2).
    throw redirect({ href: target });
  },
});
