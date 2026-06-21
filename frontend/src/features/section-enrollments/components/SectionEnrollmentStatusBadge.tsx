import { Badge } from "@/components/ui/badge";

const STATUS_MAP: Record<
  string,
  { label: string; variant: "outline" | "secondary" | "destructive" }
> = {
  in_progress: { label: "En curso", variant: "secondary" },
  passed: { label: "Aprobada", variant: "secondary" },
  failed: { label: "Reprobada", variant: "destructive" },
  withdrawn: { label: "Retirada", variant: "outline" },
};

interface SectionEnrollmentStatusBadgeProps {
  status: string;
}

export function SectionEnrollmentStatusBadge({
  status,
}: SectionEnrollmentStatusBadgeProps) {
  const mapped = STATUS_MAP[status];
  if (mapped) {
    return <Badge variant={mapped.variant}>{mapped.label}</Badge>;
  }
  // Unknown status: show raw text with outline variant (defensive fallback)
  return <Badge variant="outline">{status}</Badge>;
}
