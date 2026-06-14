import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";
import { requireRoutePermission } from "@/features/auth";
import { UsersPage } from "@/features/users";

const usersSearchSchema = z.object({
  q: z.string().default("").catch(""),
});

export const Route = createFileRoute("/_authenticated/users")({
  validateSearch: usersSearchSchema,
  beforeLoad: ({ context }) => {
    requireRoutePermission(context.session, "/users");
  },
  component: UsersPage,
});
