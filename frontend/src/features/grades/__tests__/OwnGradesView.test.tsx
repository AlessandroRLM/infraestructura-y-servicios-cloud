import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { toast } from "sonner";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import type { AuthenticatedSession } from "@/features/auth";
import { EnrollmentService } from "@/gen/enrollment/v1/enrollment_pb";
import {
  GradePeriodSchema,
  GradesService,
  ListOwnGradesResponseSchema,
  OwnGradeSchema,
} from "@/gen/grades/v1/grades_pb";
import { renderWithProviders } from "@/test";

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

type GradesImpl = Partial<ServiceImpl<typeof GradesService>>;

const studentSession = {
  status: "authenticated" as const,
  userId: "s1",
  email: "student@test.com",
  roles: ["student"],
  permissions: ["grades.view_own", "enrollment.view_own"],
};

const studentSessionSource = {
  getSession: async (): Promise<AuthenticatedSession> => ({
    userId: studentSession.userId,
    email: studentSession.email,
    roles: studentSession.roles,
    permissions: studentSession.permissions,
  }),
};

const teacherSession = {
  status: "authenticated" as const,
  userId: "t1",
  email: "teacher@test.com",
  roles: ["teacher"],
  permissions: ["grades.read"],
};

const teacherSessionSource = {
  getSession: async (): Promise<AuthenticatedSession> => ({
    userId: teacherSession.userId,
    email: teacherSession.email,
    roles: teacherSession.roles,
    permissions: teacherSession.permissions,
  }),
};

function makeGrade(
  overrides: Partial<Parameters<typeof create<typeof OwnGradeSchema>>[1]> = {},
) {
  return create(OwnGradeSchema, {
    id: "g1",
    evaluationId: "e1",
    sectionEnrollmentId: "se-1",
    value: "5.0",
    version: 1,
    createdAt: "",
    updatedAt: "",
    courseCode: "MAT101",
    courseName: "Matemáticas",
    evaluationPosition: 1,
    evaluationWeight: "0.500",
    sectionFinalGrade: "6.0",
    sectionStatus: "passed",
    academicPeriodYear: 2026,
    academicPeriodTerm: 1,
    programId: "prog-1",
    programName: "Ingeniería Civil",
    ...overrides,
  });
}

function renderGrades(gradeHandlers: GradesImpl, route = "/grades") {
  return renderWithProviders({
    route,
    transport: makeStubTransport(
      [GradesService, gradeHandlers],
      [
        EnrollmentService,
        {
          listOwnEnrollments: async () => ({
            enrollments: [],
            nextPageToken: "",
          }),
        },
      ],
    ),
    session: studentSession,
    sessionSource: studentSessionSource,
  });
}

describe("GradesPage — role branch", () => {
  it("S-F1a: student with grades.view_own sees Mis notas heading", async () => {
    renderGrades({
      listOwnGrades: async () =>
        create(ListOwnGradesResponseSchema, { grades: [], nextPageToken: "" }),
      listOwnGradePeriods: async () => ({ periods: [] }),
    });

    await screen.findByText("Mis notas");
    expect(screen.queryByText("próximamente")).not.toBeInTheDocument();
  });

  it("S-F1b: non-student role sees placeholder", async () => {
    renderWithProviders({
      route: "/grades",
      transport: makeStubTransport(
        [GradesService, {}],
        [EnrollmentService, {}],
      ),
      session: teacherSession,
      sessionSource: teacherSessionSource,
    });

    await screen.findByText(/próximamente/i);
    expect(screen.queryByText("Mis notas")).not.toBeInTheDocument();
  });
});

describe("OwnGradesView — loading state", () => {
  it("S-F7a: shows skeleton with aria-busy while loading", async () => {
    renderGrades({
      // biome-ignore lint/suspicious/noEmptyBlockStatements: intentional never-resolving promise for loading state test
      listOwnGrades: () => new Promise(() => {}),
      listOwnGradePeriods: async () => ({ periods: [] }),
    });

    await screen.findByText("Mis notas");
    const skeleton = await screen.findByRole("status", {
      name: "Cargando notas",
    });
    expect(skeleton).toBeInTheDocument();
    expect(skeleton).toHaveAttribute("aria-busy", "true");
  });
});

describe("OwnGradesView — empty state", () => {
  it("S-F7b: shows empty state when no grades exist and no filter active", async () => {
    renderGrades({
      listOwnGrades: async () =>
        create(ListOwnGradesResponseSchema, { grades: [], nextPageToken: "" }),
      listOwnGradePeriods: async () => ({ periods: [] }),
    });

    await screen.findByText(/no tienes notas registradas/i);
  });

  it("shows filtered empty state message when filter is active", async () => {
    renderGrades(
      {
        listOwnGrades: async () =>
          create(ListOwnGradesResponseSchema, {
            grades: [],
            nextPageToken: "",
          }),
        listOwnGradePeriods: async () => ({ periods: [] }),
      },
      "/grades?period=some-uuid",
    );

    await screen.findByText(/no hay notas que coincidan/i);
  });
});

describe("OwnGradesView — error state", () => {
  it("S-F7c: shows error message and Reintentar button on fetch failure", async () => {
    renderGrades({
      listOwnGrades: async () => {
        throw new ConnectError("unavailable", Code.Unavailable);
      },
      listOwnGradePeriods: async () => ({ periods: [] }),
    });

    await screen.findByText(/No se pudieron cargar las notas/i);
    expect(
      screen.getByRole("button", { name: /reintentar/i }),
    ).toBeInTheDocument();
  });

  it("Reintentar button triggers a new listOwnGrades request", async () => {
    const user = userEvent.setup();
    let callCount = 0;
    renderGrades({
      listOwnGrades: vi.fn(async () => {
        callCount++;
        if (callCount === 1)
          throw new ConnectError("unavailable", Code.Unavailable);
        return create(ListOwnGradesResponseSchema, {
          grades: [makeGrade()],
          nextPageToken: "",
        });
      }),
      listOwnGradePeriods: async () => ({ periods: [] }),
    });

    await screen.findByRole("button", { name: /reintentar/i });
    await user.click(screen.getByRole("button", { name: /reintentar/i }));
    await screen.findByText("Matemáticas");
  });
});

describe("OwnGradesView — accordion sections", () => {
  it("S-F2a: renders section header with course name, code, period, grade, status", async () => {
    renderGrades({
      listOwnGrades: async () =>
        create(ListOwnGradesResponseSchema, {
          grades: [
            makeGrade({ sectionFinalGrade: "6.0", sectionStatus: "passed" }),
          ],
          nextPageToken: "",
        }),
      listOwnGradePeriods: async () => ({ periods: [] }),
    });

    await screen.findByText("Matemáticas");
    expect(screen.getByText("MAT101")).toBeInTheDocument();
    expect(screen.getByText("2026-1")).toBeInTheDocument();
    expect(screen.getByText("6.0")).toBeInTheDocument();
    expect(screen.getByText("Aprobado")).toBeInTheDocument();
  });

  it("S-F2b: empty final grade displays — in accordion header", async () => {
    renderGrades({
      listOwnGrades: async () =>
        create(ListOwnGradesResponseSchema, {
          grades: [
            makeGrade({ sectionFinalGrade: "", sectionStatus: "in_progress" }),
          ],
          nextPageToken: "",
        }),
      listOwnGradePeriods: async () => ({ periods: [] }),
    });

    await screen.findByText("Matemáticas");
    expect(screen.getByText("—")).toBeInTheDocument();
    expect(screen.getByText("En curso")).toBeInTheDocument();
  });

  it("S-F2c: expanding accordion item reveals evaluation rows", async () => {
    const user = userEvent.setup();
    const grade1 = makeGrade({
      id: "g1",
      evaluationId: "e1",
      evaluationPosition: 1,
      evaluationWeight: "0.500",
      value: "5.0",
    });
    const grade2 = makeGrade({
      id: "g2",
      evaluationId: "e2",
      evaluationPosition: 2,
      evaluationWeight: "0.500",
      value: "6.0",
    });

    renderGrades({
      listOwnGrades: async () =>
        create(ListOwnGradesResponseSchema, {
          grades: [grade1, grade2],
          nextPageToken: "",
        }),
      listOwnGradePeriods: async () => ({ periods: [] }),
    });

    await screen.findByText("Matemáticas");

    // Click to expand the accordion item.
    await user.click(screen.getByText("Matemáticas"));

    await screen.findByText("Evaluación 1");
    expect(screen.getByText("Evaluación 2")).toBeInTheDocument();
  });

  it("S-F3a: evaluation rows show Evaluación {position}, weight as % and value — no name field", async () => {
    const user = userEvent.setup();
    renderGrades({
      listOwnGrades: async () =>
        create(ListOwnGradesResponseSchema, {
          grades: [
            makeGrade({
              evaluationPosition: 2,
              evaluationWeight: "0.400",
              value: "5.5",
            }),
          ],
          nextPageToken: "",
        }),
      listOwnGradePeriods: async () => ({ periods: [] }),
    });

    await screen.findByText("Matemáticas");
    await user.click(screen.getByText("Matemáticas"));

    await screen.findByText("Evaluación 2");
    // Weight renders as percentage, not raw decimal.
    expect(screen.getByText(/40%/)).toBeInTheDocument();
    expect(screen.queryByText(/0\.400/)).not.toBeInTheDocument();
    expect(screen.getByText("5.5")).toBeInTheDocument();
  });

  it("S-F3b: pending evaluation (empty value) renders — instead of empty", async () => {
    const user = userEvent.setup();
    renderGrades({
      listOwnGrades: async () =>
        create(ListOwnGradesResponseSchema, {
          grades: [
            makeGrade({
              evaluationId: "e-pending",
              evaluationPosition: 1,
              evaluationWeight: "1.000",
              value: "",
            }),
          ],
          nextPageToken: "",
        }),
      listOwnGradePeriods: async () => ({ periods: [] }),
    });

    await screen.findByText("Matemáticas");
    await user.click(screen.getByText("Matemáticas"));

    await screen.findByText("Evaluación 1");
    // Empty value renders as em-dash.
    expect(screen.getAllByText("—").length).toBeGreaterThanOrEqual(1);
  });
});

describe("OwnGradesView — pagination (Cargar más)", () => {
  it("S-F6b: no Cargar más when nextPageToken is empty", async () => {
    renderGrades({
      listOwnGrades: async () =>
        create(ListOwnGradesResponseSchema, {
          grades: [makeGrade()],
          nextPageToken: "",
        }),
      listOwnGradePeriods: async () => ({ periods: [] }),
    });

    await screen.findByText("Matemáticas");
    expect(
      screen.queryByRole("button", { name: /cargar más/i }),
    ).not.toBeInTheDocument();
  });

  it("S-F6a: Cargar más appends groups and prior rows remain visible", async () => {
    const user = userEvent.setup();
    let callCount = 0;
    const grade1 = makeGrade({
      id: "g1",
      sectionEnrollmentId: "se-1",
      courseName: "Matemáticas",
    });
    const grade2 = makeGrade({
      id: "g2",
      sectionEnrollmentId: "se-2",
      courseName: "Física",
      courseCode: "FIS201",
    });

    renderGrades({
      listOwnGrades: vi.fn(async () => {
        callCount++;
        if (callCount === 1) {
          return create(ListOwnGradesResponseSchema, {
            grades: [grade1],
            nextPageToken: "cursor-2",
          });
        }
        return create(ListOwnGradesResponseSchema, {
          grades: [grade2],
          nextPageToken: "",
        });
      }),
      listOwnGradePeriods: async () => ({ periods: [] }),
    });

    await screen.findByText("Matemáticas");
    expect(
      screen.getByRole("button", { name: /cargar más/i }),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /cargar más/i }));

    await screen.findByText("Física");
    expect(screen.getByText("Matemáticas")).toBeInTheDocument();
  });

  it("Cargar más failure shows toast, existing rows remain", async () => {
    const user = userEvent.setup();
    let callCount = 0;

    renderGrades({
      listOwnGrades: vi.fn(async () => {
        callCount++;
        if (callCount === 1) {
          return create(ListOwnGradesResponseSchema, {
            grades: [makeGrade({ courseName: "Matemáticas" })],
            nextPageToken: "cursor-2",
          });
        }
        throw new ConnectError("unavailable", Code.Unavailable);
      }),
      listOwnGradePeriods: async () => ({ periods: [] }),
    });

    await screen.findByText("Matemáticas");
    await user.click(screen.getByRole("button", { name: /cargar más/i }));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith(
        "No se pudieron cargar más notas.",
      );
    });
    expect(screen.getByText("Matemáticas")).toBeInTheDocument();
  });
});

describe("OwnGradesView — URL filters", () => {
  it("S-F4a: selecting a period filter updates URL and re-queries with that period", async () => {
    const user = userEvent.setup();
    const listOwnGrades = vi.fn(async () =>
      create(ListOwnGradesResponseSchema, { grades: [], nextPageToken: "" }),
    );
    const period = create(GradePeriodSchema, {
      academicPeriodId: "ap-uuid-1",
      year: 2026,
      term: 1,
    });

    const { router } = renderWithProviders({
      route: "/grades",
      transport: makeStubTransport(
        [
          GradesService,
          {
            listOwnGrades,
            listOwnGradePeriods: async () => ({ periods: [period] }),
          },
        ],
        [
          EnrollmentService,
          {
            listOwnEnrollments: async () => ({
              enrollments: [],
              nextPageToken: "",
            }),
          },
        ],
      ),
      session: studentSession,
      sessionSource: studentSessionSource,
    });

    await screen.findByText("Mis notas");

    // Open the period selector and pick "2026-1".
    await user.click(
      screen.getByRole("combobox", { name: /filtrar por período/i }),
    );
    await user.click(await screen.findByText("2026-1"));

    // URL search param must contain the period UUID.
    await waitFor(() => {
      expect(router.state.location.searchStr).toContain("ap-uuid-1");
    });

    // RPC must have been called with the selected period.
    await waitFor(() => {
      const calls = listOwnGrades.mock.calls as unknown as Array<
        [{ academicPeriodId?: string }]
      >;
      const filtered = calls.find(
        (c) => c[0]?.academicPeriodId === "ap-uuid-1",
      );
      expect(filtered).toBeTruthy();
    });
  });

  it("S-F4c: deep-link ?period=X applies filter on first render", async () => {
    const listOwnGrades = vi.fn(async () =>
      create(ListOwnGradesResponseSchema, { grades: [], nextPageToken: "" }),
    );

    renderGrades(
      { listOwnGrades, listOwnGradePeriods: async () => ({ periods: [] }) },
      "/grades?period=ap-uuid-deep",
    );

    await waitFor(() => {
      expect(listOwnGrades).toHaveBeenCalled();
    });

    const calls = listOwnGrades.mock.calls as unknown as Array<
      [{ academicPeriodId?: string }]
    >;
    expect(calls[0][0]?.academicPeriodId).toBe("ap-uuid-deep");
  });

  it("S-F4b: deep-link with both period and program applies both filters", async () => {
    const listOwnGrades = vi.fn(async () =>
      create(ListOwnGradesResponseSchema, { grades: [], nextPageToken: "" }),
    );

    renderGrades(
      { listOwnGrades, listOwnGradePeriods: async () => ({ periods: [] }) },
      "/grades?period=ap-uuid-1&program=prog-uuid-1",
    );

    await waitFor(() => {
      expect(listOwnGrades).toHaveBeenCalled();
    });

    const calls = listOwnGrades.mock.calls as unknown as Array<
      [{ academicPeriodId?: string; programId?: string }]
    >;
    expect(calls[0][0]?.academicPeriodId).toBe("ap-uuid-1");
    expect(calls[0][0]?.programId).toBe("prog-uuid-1");
  });

  it("S-F4d: selecting a carrera updates URL and re-queries with that programId", async () => {
    const user = userEvent.setup();
    const listOwnGrades = vi.fn(async () =>
      create(ListOwnGradesResponseSchema, { grades: [], nextPageToken: "" }),
    );

    const { router } = renderWithProviders({
      route: "/grades",
      transport: makeStubTransport(
        [
          GradesService,
          {
            listOwnGrades,
            listOwnGradePeriods: async () => ({ periods: [] }),
          },
        ],
        [
          EnrollmentService,
          {
            listOwnEnrollments: async () => ({
              enrollments: [
                {
                  id: "enr-1",
                  studentId: "s1",
                  programId: "prog-uuid-2",
                  programName: "Ingeniería Civil",
                  year: 2026,
                  status: "paid",
                  createdAt: "",
                  updatedAt: "",
                },
              ],
              nextPageToken: "",
            }),
          },
        ],
      ),
      session: studentSession,
      sessionSource: studentSessionSource,
    });

    await screen.findByText("Mis notas");

    // Open the carrera selector and pick "Ingeniería Civil".
    await user.click(
      screen.getByRole("combobox", { name: /filtrar por carrera/i }),
    );
    await user.click(await screen.findByText("Ingeniería Civil"));

    // URL search param must contain the program UUID.
    await waitFor(() => {
      expect(router.state.location.searchStr).toContain("prog-uuid-2");
    });

    // RPC must have been called with the selected programId.
    await waitFor(() => {
      const calls = listOwnGrades.mock.calls as unknown as Array<
        [{ programId?: string }]
      >;
      const filtered = calls.find((c) => c[0]?.programId === "prog-uuid-2");
      expect(filtered).toBeTruthy();
    });
  });
});
