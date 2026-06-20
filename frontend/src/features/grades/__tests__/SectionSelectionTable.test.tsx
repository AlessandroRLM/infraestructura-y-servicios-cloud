import type { Transport } from "@connectrpc/connect";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import type {
  AuthenticatedSession,
  Permission,
  SessionSource,
  SessionState,
} from "@/features/auth";
import {
  CatalogService,
  type ListOwnSectionsRequest,
} from "@/gen/catalog/v1/catalog_pb";
import { GradesService } from "@/gen/grades/v1/grades_pb";
import { ProfileService } from "@/gen/profiles/v1/profiles_pb";
import { SectionEnrollmentService } from "@/gen/section_enrollment/v1/section_enrollment_pb";
import { renderComponent, renderWithProviders } from "@/test";
import { SectionSelectionTable } from "../components/SectionSelectionTable";

function session(permissions: Permission[] = ["grades.write"]): SessionState {
  return {
    status: "authenticated",
    userId: "u-1",
    email: "teacher@test.com",
    roles: ["teacher"],
    permissions,
  };
}

/**
 * SessionSource that returns a real authenticated session.
 * Passed to renderWithProviders so that if the QueryClient's gcTime:0 evicts
 * the seeded session during a navigation, ensureQueryData re-fetches and still
 * gets a valid session instead of null (which would redirect to /login).
 */
function authenticatedSessionSource(
  permissions: Permission[] = ["grades.write"],
): SessionSource {
  const data: AuthenticatedSession = {
    userId: "u-1",
    email: "teacher@test.com",
    roles: ["teacher"],
    permissions,
  };
  return { getSession: async () => data };
}

const stubSections = [
  {
    id: "sec-1",
    courseId: "course-1",
    academicPeriodId: "period-1",
    seatCapacity: 30,
    courseCode: "MAT101",
    courseName: "Matemáticas I",
    periodYear: 2024,
    periodTerm: 1,
  },
  {
    id: "sec-2",
    courseId: "course-2",
    academicPeriodId: "period-1",
    seatCapacity: 25,
    courseCode: "FIS101",
    courseName: "Física I",
    periodYear: 2024,
    periodTerm: 2,
  },
];

/** Transport covering all RPCs touched at /admin/grades (list) and on click-through to
 *  /admin/grades/$sectionId (grid), so navigation tests don't cause unhandled fetch errors. */
const fullTransport = makeStubTransport(
  [
    CatalogService,
    {
      listOwnSections: async () => ({
        sections: stubSections,
        nextPageToken: "",
      }),
      listCourses: async () => ({ courses: [], nextPageToken: "" }),
    },
  ],
  [GradesService, { listEvaluations: async () => ({ evaluations: [] }) }],
  [
    SectionEnrollmentService,
    {
      listSectionRosterForTeacher: async () => ({
        sectionEnrollments: [],
        nextPageToken: "",
      }),
    },
  ],
  [ProfileService, { listDisplayNamesByIDs: async () => ({ names: [] }) }],
);

const sectionsTransport = makeStubTransport([
  CatalogService,
  {
    listOwnSections: async () => ({
      sections: stubSections,
      nextPageToken: "",
    }),
  },
]);

const emptyTransport = makeStubTransport([
  CatalogService,
  {
    listOwnSections: async () => ({ sections: [], nextPageToken: "" }),
  },
]);

// ──────────────────────────────────────────────
// Helper: renders SectionSelectionTable directly with controlled props.
// Used for unit-level tests that don't need router context.
// ──────────────────────────────────────────────
function renderTable(
  opts: {
    transport?: Transport;
    q?: string;
    pageSize?: number;
    onQueryChange?: (v: string) => void;
    onPageSizeChange?: (n: number) => void;
    onSelectSection?: (s: (typeof stubSections)[number]) => void;
  } = {},
) {
  const {
    transport = sectionsTransport,
    q = "",
    pageSize = 50,
    onQueryChange = vi.fn(),
    onPageSizeChange = vi.fn(),
    onSelectSection = vi.fn(),
  } = opts;

  return renderComponent(
    <SectionSelectionTable
      q={q}
      pageSize={pageSize}
      onQueryChange={onQueryChange}
      onPageSizeChange={onPageSizeChange}
      onSelectSection={onSelectSection}
    />,
    { session: session(), transport },
  );
}

describe("SectionSelectionTable", () => {
  it("renders section rows with Asignatura, Sección, Período columns (no cupo)", async () => {
    renderTable();

    // Wait for rows to appear
    expect(await screen.findByText("Matemáticas I")).toBeInTheDocument();
    expect(screen.getByText("Física I")).toBeInTheDocument();

    // Section codes appear in the Sección column
    expect(screen.getByText("MAT101")).toBeInTheDocument();
    expect(screen.getByText("FIS101")).toBeInTheDocument();

    // Period appears as "year · Semestre term"
    expect(screen.getByText("2024 · Semestre 1")).toBeInTheDocument();
    expect(screen.getByText("2024 · Semestre 2")).toBeInTheDocument();

    // No cupo column header
    expect(screen.queryByText(/cupo/i)).not.toBeInTheDocument();
  });

  it("calls onSelectSection when a row is clicked", async () => {
    const onSelect = vi.fn();
    const user = userEvent.setup();

    renderTable({ onSelectSection: onSelect });

    await screen.findByText("Matemáticas I");
    await user.click(screen.getByText("Matemáticas I"));

    expect(onSelect).toHaveBeenCalledOnce();
    expect(onSelect).toHaveBeenCalledWith(
      expect.objectContaining({ id: "sec-1", courseCode: "MAT101" }),
    );
  });

  it("shows empty state when no sections are returned", async () => {
    renderTable({ transport: emptyTransport });

    expect(
      await screen.findByText("No hay secciones asignadas."),
    ).toBeInTheDocument();
  });

  it("navigates to /admin/grades/$sectionId when a row is clicked (route integration)", async () => {
    const user = userEvent.setup();

    const { router } = renderWithProviders({
      route: "/admin/grades",
      session: session(),
      sessionSource: authenticatedSessionSource(),
      transport: fullTransport,
    });

    await screen.findByText("Matemáticas I");
    await user.click(screen.getByText("Matemáticas I"));

    // Clicking a row navigates to /admin/grades/$sectionId
    await waitFor(() => {
      expect(router.state.location.pathname).toBe("/admin/grades/sec-1");
    });
  });
});

// ──────────────────────────────────────────────
// Change 2: server-side search in section table
// ──────────────────────────────────────────────

describe("SectionSelectionTable — server-side search", () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("calls onQueryChange with the debounced value after typing", async () => {
    const user = userEvent.setup({
      advanceTimers: vi.advanceTimersByTime.bind(vi),
    });
    const onQueryChange = vi.fn();

    // renderTable (renderComponent) avoids router bootstrap timing issues
    // with fake timers. Tests the component's debounce behavior in isolation.
    renderTable({
      transport: sectionsTransport,
      onQueryChange,
    });

    await screen.findByText("Matemáticas I");

    const input = screen.getByPlaceholderText("Buscar asignatura…");
    await user.type(input, "MAT");

    // Advance past the 300ms debounce.
    await vi.advanceTimersByTimeAsync(300);

    // onQueryChange fires with the debounced value — the route container
    // is responsible for writing it to the URL search param.
    await waitFor(() => {
      expect(onQueryChange).toHaveBeenCalledWith("MAT");
    });
  });
});

describe("SectionSelectionTable — search route integration", () => {
  it("passes the debounced query to useOwnSections after typing and syncs it to the URL", async () => {
    // Uses real timers + a configured sessionSource so that navigation during
    // the test doesn't evict the session from the QueryClient (gcTime:0).
    const user = userEvent.setup();
    const listOwnSections = vi.fn(async (req: ListOwnSectionsRequest) => ({
      sections: req.query === "MAT" ? [stubSections[0]] : stubSections,
      nextPageToken: "",
    }));

    const transport = makeStubTransport([CatalogService, { listOwnSections }]);

    const { router } = renderWithProviders({
      route: "/admin/grades",
      session: session(),
      sessionSource: authenticatedSessionSource(),
      transport,
    });

    // Wait for initial load.
    await screen.findByText("Matemáticas I");

    // Type into the search box.
    const input = screen.getByPlaceholderText("Buscar asignatura…");
    await user.type(input, "MAT");

    // Wait past the 300ms debounce (real timers — waitFor polls until timeout).
    // The URL search param q must be updated to "MAT".
    await waitFor(
      () => {
        expect(router.state.location.searchStr).toContain("q=MAT");
      },
      { timeout: 1000 },
    );

    // useOwnSections should have been called with query="MAT" after the debounce.
    await waitFor(() => {
      const queryCalls = listOwnSections.mock.calls.map(([req]) => req.query);
      expect(queryCalls).toContain("MAT");
    });
  });
});

describe("SectionSelectionTable — search empty state", () => {
  it("shows no-results empty state when q is non-empty and no sections are returned", async () => {
    const transport = makeStubTransport([
      CatalogService,
      {
        listOwnSections: async () => ({ sections: [], nextPageToken: "" }),
      },
    ]);

    // q="XYZ" → component shows the search-specific empty state.
    renderTable({ transport, q: "XYZ" });

    await screen.findByText("No se encontraron secciones para la búsqueda.");
  });
});

// ──────────────────────────────────────────────
// Page-size selector
// ──────────────────────────────────────────────

describe("SectionSelectionTable — page-size selector", () => {
  it("defaults to 50 and forwards a changed page size to useOwnSections and URL", async () => {
    const user = userEvent.setup();
    const listOwnSections = vi.fn(async (_req: ListOwnSectionsRequest) => ({
      sections: stubSections,
      nextPageToken: "",
    }));
    const transport = makeStubTransport([CatalogService, { listOwnSections }]);

    // Provide a real sessionSource so that if gcTime:0 evicts the seeded session
    // during the navigate({search:...}) call, ensureQueryData re-fetches correctly.
    const { router } = renderWithProviders({
      route: "/admin/grades",
      session: session(),
      sessionSource: authenticatedSessionSource(),
      transport,
    });

    await screen.findByText("Matemáticas I");

    // Initial load uses the default page size of 50.
    expect(listOwnSections.mock.calls.map(([req]) => req.pageSize)).toContain(
      50,
    );

    // Open the selector and pick "20 por página".
    await user.click(
      screen.getByRole("combobox", { name: /filas por página/i }),
    );
    await user.click(screen.getByRole("option", { name: /20 por página/i }));

    // The URL search param pageSize must be updated to 20.
    await waitFor(() => {
      expect(router.state.location.searchStr).toContain("pageSize=20");
    });

    // The new page size is forwarded to the RPC.
    await waitFor(() => {
      expect(listOwnSections.mock.calls.map(([req]) => req.pageSize)).toContain(
        20,
      );
    });
  });
});
