import { Outlet } from "@tanstack/react-router";

// The _authenticated route is a pathless layout that enforces session auth and
// computes eligibility. Its children (admin/route.tsx, app/route.tsx) each
// provide their own sidebar + shell layout. Shared routes (/profile, /forbidden,
// /choose-area) render without an area-specific sidebar.
export function AuthenticatedLayout() {
  return <Outlet />;
}
