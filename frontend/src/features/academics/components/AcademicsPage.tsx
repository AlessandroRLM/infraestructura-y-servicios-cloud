import { Plus } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { hasPermission, useSession } from "@/features/auth";
import { Route } from "@/routes/_authenticated/admin/academics";
import { CourseDialog } from "./CourseDialog";
import { CoursesTable } from "./CoursesTable";
import { ProgramDialog } from "./ProgramDialog";
import { ProgramsTable } from "./ProgramsTable";
import { SectionDialog } from "./SectionDialog";
import { SectionsTable } from "./SectionsTable";

const TAB_LABELS = {
  programs: "Carreras",
  courses: "Asignaturas",
  sections: "Secciones",
} as const;

type TabKey = keyof typeof TAB_LABELS;

export function AcademicsPage() {
  const session = useSession();
  const canManage = hasPermission(session, "catalog.manage");
  const { tab } = Route.useSearch();
  const navigate = Route.useNavigate();

  const [createProgramOpen, setCreateProgramOpen] = useState(false);
  const [createCourseOpen, setCreateCourseOpen] = useState(false);
  const [createSectionOpen, setCreateSectionOpen] = useState(false);

  const handleTabChange = (value: string) => {
    // Reset q and pageSize on tab switch so each tab starts with a clean state.
    navigate({
      search: {
        tab: value as TabKey,
        q: "",
        pageSize: 20,
      },
    });
  };

  const createLabel =
    tab === "programs"
      ? "Crear carrera"
      : tab === "courses"
        ? "Crear asignatura"
        : "Crear sección";

  const handleCreateClick = () => {
    if (tab === "programs") {
      setCreateProgramOpen(true);
    } else if (tab === "courses") {
      setCreateCourseOpen(true);
    } else {
      setCreateSectionOpen(true);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="font-semibold text-2xl tracking-tight">Académico</h1>
        {canManage && (
          <Button onClick={handleCreateClick} className="gap-2">
            <Plus className="size-4" aria-hidden />
            {createLabel}
          </Button>
        )}
      </div>

      <Tabs value={tab} onValueChange={handleTabChange}>
        <TabsList>
          <TabsTrigger value="programs">{TAB_LABELS.programs}</TabsTrigger>
          <TabsTrigger value="courses">{TAB_LABELS.courses}</TabsTrigger>
          <TabsTrigger value="sections">{TAB_LABELS.sections}</TabsTrigger>
        </TabsList>
        <TabsContent value="programs">
          <ProgramsTable onCreateClick={() => setCreateProgramOpen(true)} />
        </TabsContent>
        <TabsContent value="courses">
          <CoursesTable onCreateClick={() => setCreateCourseOpen(true)} />
        </TabsContent>
        <TabsContent value="sections">
          <SectionsTable onCreateClick={() => setCreateSectionOpen(true)} />
        </TabsContent>
      </Tabs>

      <ProgramDialog
        open={createProgramOpen}
        onOpenChange={setCreateProgramOpen}
      />

      <CourseDialog
        open={createCourseOpen}
        onOpenChange={setCreateCourseOpen}
      />

      <SectionDialog
        open={createSectionOpen}
        onOpenChange={setCreateSectionOpen}
      />
    </div>
  );
}
