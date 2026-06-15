import { createFileRoute, redirect } from "@tanstack/react-router";

// The admin area root redirects to the first available admin feature with
// its required default search params so the academics route's validateSearch
// receives a well-formed object.
export const Route = createFileRoute("/_authenticated/admin/")({
  beforeLoad: () => {
    throw redirect({
      to: "/admin/academics",
      search: { tab: "programs", q: "", pageSize: 20 },
    });
  },
});
