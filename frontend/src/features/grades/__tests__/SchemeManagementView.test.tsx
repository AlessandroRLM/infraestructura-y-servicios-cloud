import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { toast } from "sonner";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import { CatalogService } from "@/gen/catalog/v1/catalog_pb";
import {
  EvaluationSchema,
  GradesService,
  ListEvaluationsResponseSchema,
} from "@/gen/grades/v1/grades_pb";
import { renderComponent } from "@/test";
import { SchemeManagementView } from "../components/SchemeManagementView";

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

type GradesImpl = Partial<ServiceImpl<typeof GradesService>>;
type CatalogImpl = Partial<ServiceImpl<typeof CatalogService>>;

/** Course stub returned by the catalog ListCourses stub. */
const stubCourse = {
  id: "course-1",
  code: "MAT101",
  name: "Matemáticas",
  credits: 5,
  createdAt: "",
  updatedAt: "",
};

/** Default catalog stub: returns a single course for any search. */
const defaultCatalogImpl: CatalogImpl = {
  listCourses: async () => ({
    courses: [stubCourse],
    nextPageToken: "",
  }),
};

/** Creates evaluations for a 3-row scheme: [30%, 30%, 40%]. */
function makeEvaluations() {
  return [
    create(EvaluationSchema, {
      id: "ev-1",
      courseId: "course-1",
      weight: "0.300",
      position: 1,
      createdAt: "",
      updatedAt: "",
    }),
    create(EvaluationSchema, {
      id: "ev-2",
      courseId: "course-1",
      weight: "0.300",
      position: 2,
      createdAt: "",
      updatedAt: "",
    }),
    create(EvaluationSchema, {
      id: "ev-3",
      courseId: "course-1",
      weight: "0.400",
      position: 3,
      createdAt: "",
      updatedAt: "",
    }),
  ];
}

function renderView(
  gradesImpl: GradesImpl,
  catalogImpl: CatalogImpl = defaultCatalogImpl,
  initialCourseId?: string,
  initialCourseLabel?: string,
) {
  return renderComponent(
    <SchemeManagementView
      initialCourseId={initialCourseId}
      initialCourseLabel={initialCourseLabel}
    />,
    {
      transport: makeStubTransport(
        [GradesService, gradesImpl],
        [CatalogService, catalogImpl],
      ),
    },
  );
}

/** Opens the course picker popover and selects the stub course (MAT101). */
async function selectCourse(user: ReturnType<typeof userEvent.setup>) {
  // The picker button has role="combobox".
  await user.click(
    screen.getByRole("combobox", { name: /seleccionar asignatura/i }),
  );
  await screen.findByRole("option", { name: /mat101/i });
  await user.click(screen.getByRole("option", { name: /mat101/i }));
}

// ──────────────────────────────────────────────
// S-02: No course selected — scheme section absent
// ──────────────────────────────────────────────

describe("SchemeManagementView — S-02: no course selected", () => {
  it("S-02c: no scheme section is rendered before a course is selected", () => {
    renderView({
      listEvaluations: async () =>
        create(ListEvaluationsResponseSchema, { evaluations: [] }),
    });

    expect(screen.queryByRole("status")).not.toBeInTheDocument();
    expect(
      screen.queryByText(/este curso no tiene un esquema/i),
    ).not.toBeInTheDocument();
  });
});

// ──────────────────────────────────────────────
// S-03 / S-04: Scheme display states
// ──────────────────────────────────────────────

describe("SchemeManagementView — scheme display", () => {
  it("S-04: shows empty state with Crear esquema when course has no evaluations", async () => {
    const user = userEvent.setup();
    renderView({
      listEvaluations: async () =>
        create(ListEvaluationsResponseSchema, { evaluations: [] }),
    });

    await selectCourse(user);

    await screen.findByText(/este curso no tiene un esquema/i);
    expect(
      screen.getByRole("button", { name: /crear esquema/i }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /recrear esquema/i }),
    ).not.toBeInTheDocument();
  });

  it("S-03: shows evaluations ordered by position with percentage weights", async () => {
    const user = userEvent.setup();
    renderView({
      listEvaluations: async () =>
        create(ListEvaluationsResponseSchema, {
          evaluations: makeEvaluations(),
        }),
    });

    await selectCourse(user);

    await screen.findByText("Evaluación 1");
    expect(screen.getByText("Evaluación 2")).toBeInTheDocument();
    expect(screen.getByText("Evaluación 3")).toBeInTheDocument();

    // Weights displayed as percentages.
    const thirties = screen.getAllByText("30%");
    expect(thirties).toHaveLength(2);
    expect(screen.getByText("40%")).toBeInTheDocument();

    // "Recrear esquema" button is shown (not crear) because evaluations exist.
    expect(
      screen.getByRole("button", { name: /recrear esquema/i }),
    ).toBeInTheDocument();
  });
});

// ──────────────────────────────────────────────
// S-05: Loading and error states
// ──────────────────────────────────────────────

describe("SchemeManagementView — loading/error states", () => {
  it("S-05a: shows loading skeleton while ListEvaluations is in flight", async () => {
    const user = userEvent.setup();
    renderView({
      // biome-ignore lint/suspicious/noEmptyBlockStatements: intentional never-resolving promise for loading state test
      listEvaluations: () => new Promise(() => {}),
    });

    await selectCourse(user);

    await screen.findByRole("status", { name: /cargando esquema/i });
    expect(screen.queryByText(/este curso no tiene/i)).not.toBeInTheDocument();
  });

  it("S-05b: shows error message and Reintentar button when ListEvaluations fails", async () => {
    const user = userEvent.setup();
    renderView({
      listEvaluations: async () => {
        throw new ConnectError("unavailable", Code.Unavailable);
      },
    });

    await selectCourse(user);

    await screen.findByText(/no se pudo cargar el esquema/i);
    expect(
      screen.getByRole("button", { name: /reintentar/i }),
    ).toBeInTheDocument();
  });
});

// ──────────────────────────────────────────────
// S-07: Create path — submit blocked while loading
// ──────────────────────────────────────────────

describe("SchemeManagementView — S-07: create scheme", () => {
  it("S-07b: submit is blocked while ListEvaluations is loading (schemeStateKnown=false)", async () => {
    const user = userEvent.setup();
    let resolveEvaluations!: (v: unknown) => void;
    const evaluationsPending = new Promise((resolve) => {
      resolveEvaluations = resolve;
    });

    renderView({
      listEvaluations: () =>
        evaluationsPending as ReturnType<
          GradesImpl["listEvaluations"] & object
        >,
    });

    await selectCourse(user);
    // Skeleton is shown — form is not rendered, no submit possible.
    await screen.findByRole("status", { name: /cargando esquema/i });

    // While loading, form inputs and submit button must not be reachable.
    expect(screen.queryByRole("spinbutton")).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /crear esquema/i }),
    ).not.toBeInTheDocument();

    // Resolve so the test can clean up.
    resolveEvaluations(
      create(ListEvaluationsResponseSchema, { evaluations: [] }),
    );
  });

  it("S-07a: calls CreateEvaluationScheme on empty-scheme course", async () => {
    const user = userEvent.setup();
    const createEvaluationScheme = vi.fn(async () => ({
      evaluations: [],
    }));

    renderView({
      listEvaluations: async () =>
        create(ListEvaluationsResponseSchema, { evaluations: [] }),
      createEvaluationScheme,
    });

    await selectCourse(user);
    await screen.findByText(/este curso no tiene un esquema/i);

    // Open the form via "Crear esquema" in the empty state.
    await user.click(screen.getByRole("button", { name: /crear esquema/i }));

    // Fill row 0 to 100% (single row, valid).
    await waitFor(() => screen.getAllByRole("spinbutton"));
    const inputs = screen.getAllByRole("spinbutton");
    await user.clear(inputs[0]);
    await user.type(inputs[0], "100");
    await user.tab();

    // The form submit button is also "Crear esquema".
    await user.click(screen.getByRole("button", { name: /crear esquema/i }));

    await waitFor(() => {
      expect(createEvaluationScheme).toHaveBeenCalledWith(
        expect.objectContaining({
          courseId: "course-1",
          evaluations: [expect.objectContaining({ weight: "1.000" })],
        }),
        expect.anything(),
      );
    });

    // S-07a: form unmounts after successful create.
    await waitFor(() => {
      expect(screen.queryByRole("spinbutton")).not.toBeInTheDocument();
    });

    await waitFor(() => {
      expect(toast.success).toHaveBeenCalledWith(
        "Esquema creado correctamente.",
      );
    });
  });
});

// ──────────────────────────────────────────────
// S-08: Recreate path — calls RecreateEvaluationScheme (not Create)
// ──────────────────────────────────────────────

describe("SchemeManagementView — S-08: recreate scheme", () => {
  it("S-08b: calls RecreateEvaluationScheme when evaluations already exist", async () => {
    const user = userEvent.setup();
    const recreateEvaluationScheme = vi.fn(async () => ({
      evaluations: makeEvaluations(),
    }));

    renderView({
      listEvaluations: async () =>
        create(ListEvaluationsResponseSchema, {
          evaluations: makeEvaluations(),
        }),
      recreateEvaluationScheme,
    });

    await selectCourse(user);

    // "Recrear esquema" button is in CurrentSchemeDisplay (display mode).
    await screen.findByRole("button", { name: /recrear esquema/i });
    await user.click(screen.getByRole("button", { name: /recrear esquema/i }));

    // Now the form is open with pre-filled rows (30+30+40=100). Submit.
    const submitBtn = await screen.findByRole("button", {
      name: /recrear esquema/i,
    });
    await user.click(submitBtn);

    // AlertDialog confirm appears.
    await screen.findByRole("alertdialog");
    await user.click(screen.getByRole("button", { name: /^recrear$/i }));

    await waitFor(() => {
      expect(recreateEvaluationScheme).toHaveBeenCalledWith(
        expect.objectContaining({
          courseId: "course-1",
          evaluations: [
            expect.objectContaining({ weight: "0.300" }),
            expect.objectContaining({ weight: "0.300" }),
            expect.objectContaining({ weight: "0.400" }),
          ],
        }),
        expect.anything(),
      );
    });

    // S-08b: form unmounts after successful recreate (mirrors S-07a).
    await waitFor(() => {
      expect(screen.queryByRole("spinbutton")).not.toBeInTheDocument();
    });

    await waitFor(() => {
      expect(toast.success).toHaveBeenCalledWith(
        "Esquema recreado correctamente.",
      );
    });
  });
});

// ──────────────────────────────────────────────
// S-09: FailedPrecondition — locked scheme
// ──────────────────────────────────────────────

describe("SchemeManagementView — S-09: FailedPrecondition", () => {
  it("S-09a: shows precondition message inline when RecreateEvaluationScheme returns FailedPrecondition", async () => {
    const user = userEvent.setup();

    renderView({
      listEvaluations: async () =>
        create(ListEvaluationsResponseSchema, {
          evaluations: makeEvaluations(),
        }),
      recreateEvaluationScheme: async () => {
        throw new ConnectError("failed precondition", Code.FailedPrecondition);
      },
    });

    await selectCourse(user);

    // Open the form.
    await screen.findByRole("button", { name: /recrear esquema/i });
    await user.click(screen.getByRole("button", { name: /recrear esquema/i }));

    // Submit the pre-filled form (30+30+40=100).
    const submitBtn = await screen.findByRole("button", {
      name: /recrear esquema/i,
    });
    await user.click(submitBtn);

    // Confirm in AlertDialog.
    await screen.findByRole("alertdialog");
    await user.click(screen.getByRole("button", { name: /^recrear$/i }));

    // Inline precondition error appears.
    await screen.findByText(/este curso ya tiene notas registradas/i);

    // Form stays open (spinbuttons are visible).
    expect(screen.getAllByRole("spinbutton").length).toBeGreaterThan(0);
  });

  describe("S-09b: precondition error clears on course change (fake timers)", () => {
    beforeEach(() => {
      // shouldAdvanceTime keeps real-time polling (findBy*, waitFor) working
      // while still allowing deterministic manual advances for the debounce.
      vi.useFakeTimers({ shouldAdvanceTime: true });
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it("S-09b: precondition error clears when user selects a different course", async () => {
      // userEvent v14 + fake timers: pass advanceTimers so pointer/keyboard
      // delays inside userEvent are also driven by the fake clock.
      const user = userEvent.setup({
        advanceTimers: vi.advanceTimersByTime.bind(vi),
      });
      const anotherCourse = {
        id: "course-2",
        code: "FIS201",
        name: "Física",
        credits: 4,
        createdAt: "",
        updatedAt: "",
      };

      renderView(
        {
          listEvaluations: async ({ courseId }) => {
            if (courseId === "course-1") {
              return create(ListEvaluationsResponseSchema, {
                evaluations: makeEvaluations(),
              });
            }
            return create(ListEvaluationsResponseSchema, { evaluations: [] });
          },
          recreateEvaluationScheme: async () => {
            throw new ConnectError(
              "failed precondition",
              Code.FailedPrecondition,
            );
          },
        },
        {
          listCourses: async ({ query }) => ({
            courses: query?.includes("Fis") ? [anotherCourse] : [stubCourse],
            nextPageToken: "",
          }),
        },
      );

      // Select first course and trigger FailedPrecondition.
      await selectCourse(user);
      await screen.findByRole("button", { name: /recrear esquema/i });
      await user.click(
        screen.getByRole("button", { name: /recrear esquema/i }),
      );
      const submitBtn = await screen.findByRole("button", {
        name: /recrear esquema/i,
      });
      await user.click(submitBtn);
      await screen.findByRole("alertdialog");
      await user.click(screen.getByRole("button", { name: /^recrear$/i }));
      await screen.findByText(/este curso ya tiene notas registradas/i);

      // Change course: open the picker (now labelled "MAT101 — Matemáticas" after selection).
      const picker = screen.getByRole("combobox", {
        name: /mat101/i,
      });
      await user.click(picker);
      // Type to filter for the second course; advance past the 300ms debounce
      // deterministically so the query fires before findByRole times out.
      const input = await screen.findByPlaceholderText(/buscar asignatura/i);
      await user.type(input, "Fis");
      await vi.advanceTimersByTimeAsync(300);
      await screen.findByRole("option", { name: /fis201/i });
      await user.click(screen.getByRole("option", { name: /fis201/i }));

      // The error message should no longer be visible.
      expect(
        screen.queryByText(/este curso ya tiene notas registradas/i),
      ).not.toBeInTheDocument();
    });
  });
});

// ──────────────────────────────────────────────
// Generic error → toast
// ──────────────────────────────────────────────

describe("SchemeManagementView — generic error handling", () => {
  it("shows toast on generic transport error from CreateEvaluationScheme", async () => {
    const user = userEvent.setup();

    renderView({
      listEvaluations: async () =>
        create(ListEvaluationsResponseSchema, { evaluations: [] }),
      createEvaluationScheme: async () => {
        throw new ConnectError("unavailable", Code.Unavailable);
      },
    });

    await selectCourse(user);
    await screen.findByText(/este curso no tiene un esquema/i);
    await user.click(screen.getByRole("button", { name: /crear esquema/i }));

    const inputs = screen.getAllByRole("spinbutton");
    await user.clear(inputs[0]);
    await user.type(inputs[0], "100");
    await user.tab();

    await user.click(screen.getByRole("button", { name: /crear esquema/i }));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith(
        "No se pudo guardar el esquema. Inténtalo de nuevo.",
      );
    });
  });
});

// ──────────────────────────────────────────────
// Change 2: initialCourseId pre-scope
// ──────────────────────────────────────────────

describe("SchemeManagementView — initialCourseId pre-scope (Change 2)", () => {
  it("fires useEvaluations immediately when initialCourseId is provided", async () => {
    // When initialCourseId is set, the scheme section must render without any
    // user interaction — useEvaluations fires on mount for the preset id.
    renderView(
      {
        listEvaluations: async () =>
          create(ListEvaluationsResponseSchema, { evaluations: [] }),
      },
      defaultCatalogImpl,
      "course-1",
    );

    // The empty-scheme message must appear without selecting a course manually.
    await screen.findByText(/este curso no tiene un esquema/i);
  });

  it("renders the existing scheme immediately when initialCourseId has evaluations", async () => {
    renderView(
      {
        listEvaluations: async () =>
          create(ListEvaluationsResponseSchema, {
            evaluations: makeEvaluations(),
          }),
      },
      defaultCatalogImpl,
      "course-1",
    );

    // Evaluation rows must load without the user picking a course.
    await screen.findByText("Evaluación 1");
    expect(screen.getByText("Evaluación 2")).toBeInTheDocument();
    expect(screen.getByText("Evaluación 3")).toBeInTheDocument();
  });

  it("renders blank picker (no pre-scope) when initialCourseId is omitted", () => {
    // Default behaviour unchanged: no scheme section before a course is selected.
    renderView({
      listEvaluations: async () =>
        create(ListEvaluationsResponseSchema, { evaluations: [] }),
    });

    expect(screen.queryByRole("status")).not.toBeInTheDocument();
    expect(
      screen.queryByText(/este curso no tiene un esquema/i),
    ).not.toBeInTheDocument();
  });
});

// ──────────────────────────────────────────────
// Lock course label: Change 1 UX fix
// ──────────────────────────────────────────────

describe("SchemeManagementView — locked course label (Change 1 UX fix)", () => {
  it("shows the locked course label instead of the picker when initialCourseId is set", () => {
    renderView(
      {
        listEvaluations: async () =>
          create(ListEvaluationsResponseSchema, { evaluations: [] }),
      },
      defaultCatalogImpl,
      "course-1",
      "MAT101 — Cálculo I",
    );

    // The label is shown.
    expect(screen.getByText("MAT101 — Cálculo I")).toBeInTheDocument();

    // The searchable picker combobox must NOT be present.
    expect(
      screen.queryByRole("combobox", { name: /seleccionar asignatura/i }),
    ).not.toBeInTheDocument();
  });

  it("falls back to showing initialCourseId when initialCourseLabel is omitted", () => {
    renderView(
      {
        listEvaluations: async () =>
          create(ListEvaluationsResponseSchema, { evaluations: [] }),
      },
      defaultCatalogImpl,
      "course-1",
      // no label — should fall back to the raw ID
    );

    expect(screen.getByText("course-1")).toBeInTheDocument();
    expect(
      screen.queryByRole("combobox", { name: /seleccionar asignatura/i }),
    ).not.toBeInTheDocument();
  });

  it("still renders the picker when initialCourseId is empty", () => {
    renderView({
      listEvaluations: async () =>
        create(ListEvaluationsResponseSchema, { evaluations: [] }),
    });

    // Picker combobox must be present.
    expect(
      screen.getByRole("combobox", { name: /seleccionar asignatura/i }),
    ).toBeInTheDocument();
  });
});

// ──────────────────────────────────────────────
// AlreadyExists → inline already-exists message
// ──────────────────────────────────────────────

describe("SchemeManagementView — AlreadyExists error handling", () => {
  it("shows inline already-exists message when CreateEvaluationScheme returns AlreadyExists", async () => {
    const user = userEvent.setup();

    renderView({
      listEvaluations: async () =>
        create(ListEvaluationsResponseSchema, { evaluations: [] }),
      createEvaluationScheme: async () => {
        throw new ConnectError("already exists", Code.AlreadyExists);
      },
    });

    await selectCourse(user);
    await screen.findByText(/este curso no tiene un esquema/i);
    await user.click(screen.getByRole("button", { name: /crear esquema/i }));

    const inputs = screen.getAllByRole("spinbutton");
    await user.clear(inputs[0]);
    await user.type(inputs[0], "100");
    await user.tab();

    await user.click(screen.getByRole("button", { name: /crear esquema/i }));

    // Inline already-exists message appears in the form.
    await screen.findByText(/el esquema ya existe/i);

    // Form stays open (spinbuttons are visible).
    expect(screen.getAllByRole("spinbutton").length).toBeGreaterThan(0);
  });
});
