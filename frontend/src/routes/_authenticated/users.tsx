import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";
import { requireAnyPermission } from "@/features/auth";
import { UsersPage } from "@/features/users";

const usersSearchSchema = z.object({
  q: z.string().default("").catch(""),
});

export const Route = createFileRoute("/_authenticated/users")({
  validateSearch: usersSearchSchema,
  beforeLoad: ({ context }) => {
    requireAnyPermission(context.session, ["users.manage"]);
  },
  component: UsersPage,
});
