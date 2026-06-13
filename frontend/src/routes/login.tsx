import { createFileRoute } from "@tanstack/react-router";
import { LoginForm, loginSearchSchema } from "@/features/auth";

export const Route = createFileRoute("/login")({
  validateSearch: loginSearchSchema,
  component: LoginRoute,
});

function LoginRoute() {
  const { redirect } = Route.useSearch();
  return <LoginForm redirectTo={redirect} />;
}
