import { Badge } from "@/components/ui/badge";

const STATUS_MAP: Record<string, { label: string; variant: "outline" | "secondary" | "destructive" }> = {
  pending: { label: "Pendiente", variant: "outline" },
  paid: { label: "Pagada", variant: "secondary" },
  cancelled: { label: "Cancelada", variant: "destructive" },
};

interface EnrollmentStatusBadgeProps {
  status: string;
}

export function EnrollmentStatusBadge({ status }: EnrollmentStatusBadgeProps) {
  const mapped = STATUS_MAP[status];
  if (mapped) {
    return <Badge variant={mapped.variant}>{mapped.label}</Badge>;
  }
  // Unknown status: show raw text with outline variant (defensive fallback)
  return <Badge variant="outline">{status}</Badge>;
}
