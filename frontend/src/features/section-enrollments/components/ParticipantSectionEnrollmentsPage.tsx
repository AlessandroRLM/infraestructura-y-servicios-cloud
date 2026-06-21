import { EnrollableSectionsList } from "./EnrollableSectionsList";
import { OwnSectionEnrollmentsList } from "./OwnSectionEnrollmentsList";

interface ParticipantSectionEnrollmentsPageProps {
  pageSize: number;
  onPageSizeChange: (n: number) => void;
}

/**
 * Student-facing page that composes two independent panels:
 * 1. "Inscribirme a secciones" — shows sections available for self-enrollment.
 * 2. "Mis secciones" — shows the student's existing section enrollments.
 *
 * Each panel resolves independently; a failure in one does not block the other.
 */
export function ParticipantSectionEnrollmentsPage({
  pageSize,
  onPageSizeChange,
}: ParticipantSectionEnrollmentsPageProps) {
  return (
    <div className="flex flex-col gap-10">
      <section>
        <h1 className="font-semibold text-2xl tracking-tight mb-4">
          Inscribirme a secciones
        </h1>
        <EnrollableSectionsList />
      </section>

      <section>
        <OwnSectionEnrollmentsList
          pageSize={pageSize}
          onPageSizeChange={onPageSizeChange}
        />
      </section>
    </div>
  );
}
