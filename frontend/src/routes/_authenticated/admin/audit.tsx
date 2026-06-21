import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";
import { AuditLogsPage } from "@/features/audit";
import { requireRoutePermission } from "@/features/auth";

const auditSearchSchema = z.object({
  actorId: z.string().default("").catch(""),
  from: z.string().default("").catch(""),
  to: z.string().default("").catch(""),
});

export const Route = createFileRoute("/_authenticated/admin/audit")({
  validateSearch: auditSearchSchema,
  beforeLoad: ({ context }) => {
    requireRoutePermission(context.session, "/admin/audit");
  },
  component: AuditLogsPage,
});
