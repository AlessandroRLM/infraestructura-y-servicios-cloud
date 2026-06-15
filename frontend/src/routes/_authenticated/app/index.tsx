import { createFileRoute, redirect } from "@tanstack/react-router";

// The participant area root redirects to the participant grades view with its
// required default search params so app/grades validateSearch receives a
// well-formed object.
export const Route = createFileRoute("/_authenticated/app/")({
  beforeLoad: () => {
    throw redirect({
      to: "/app/grades",
      search: { period: "", program: "", pageSize: 20 },
    });
  },
});
