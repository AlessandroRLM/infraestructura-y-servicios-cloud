import {
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import type { GradeSectionGroup as GradeSectionGroupData } from "../groupBySection";
import { formatWeight } from "../groupBySection";

interface GradeSectionGroupProps {
  group: GradeSectionGroupData;
}

/**
 * Renders one accordion item for a section enrollment.
 * Header: course name · code + period + final grade (or "—") + status label.
 * Body: evaluation rows, each labeled "Evaluación {position}" with weight and grade value.
 */
export function GradeSectionGroup({ group }: GradeSectionGroupProps) {
  const displayGrade = group.finalGrade || "—";

  return (
    <AccordionItem value={group.sectionEnrollmentId}>
      <AccordionTrigger className="hover:no-underline px-4">
        <div className="flex flex-1 flex-wrap items-center gap-x-4 gap-y-1 text-sm">
          <span className="font-medium">{group.courseName}</span>
          <span className="text-muted-foreground">{group.courseCode}</span>
          <span className="text-muted-foreground">{group.period}</span>
          <span className="ml-auto flex items-center gap-3">
            <span className="font-mono font-medium tabular-nums">
              {displayGrade}
            </span>
            <span className="text-muted-foreground">{group.status}</span>
          </span>
        </div>
      </AccordionTrigger>
      <AccordionContent className="px-4">
        <div className="flex flex-col gap-1">
          {group.evaluations.map((ev) => (
            <div
              key={ev.evaluationId}
              className="flex items-center justify-between rounded-md bg-muted/40 px-3 py-2 text-sm"
            >
              <span className="text-muted-foreground">
                Evaluación {ev.position}
              </span>
              <div className="flex items-center gap-6">
                <span className="text-muted-foreground text-xs">
                  Peso: {formatWeight(ev.weight)}
                </span>
                <span className="font-mono font-medium tabular-nums">
                  {ev.value || "—"}
                </span>
              </div>
            </div>
          ))}
        </div>
      </AccordionContent>
    </AccordionItem>
  );
}
