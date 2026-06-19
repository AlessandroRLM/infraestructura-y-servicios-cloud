import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import type { Permission, SessionState } from "@/features/auth";
import {
  CatalogService,
  type ListOwnSectionsRequest,
} from "@/gen/catalog/v1/catalog_pb";
import { renderComponent } from "@/test";
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

describe("SectionSelectionTable", () => {
  it("renders section rows with Asignatura, Sección, Período columns (no cupo)", async () => {
    const onSelect = vi.fn();

    renderComponent(<SectionSelectionTable onSelectSection={onSelect} />, {
      session: session(),
      transport: sectionsTransport,
    });

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

    renderComponent(<SectionSelectionTable onSelectSection={onSelect} />, {
      session: session(),
      transport: sectionsTransport,
    });

    await screen.findByText("Matemáticas I");
    await user.click(screen.getByText("Matemáticas I"));

    expect(onSelect).toHaveBeenCalledOnce();
    expect(onSelect).toHaveBeenCalledWith(
      expect.objectContaining({ id: "sec-1", courseCode: "MAT101" }),
    );
  });

  it("shows empty state when no sections are returned", async () => {
    const onSelect = vi.fn();

    renderComponent(<SectionSelectionTable onSelectSection={onSelect} />, {
      session: session(),
      transport: emptyTransport,
    });

    expect(
      await screen.findByText("No hay secciones asignadas."),
    ).toBeInTheDocument();
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

  it("passes the debounced query to useOwnSections after typing", async () => {
    const user = userEvent.setup({
      advanceTimers: vi.advanceTimersByTime.bind(vi),
    });
    const listOwnSections = vi.fn(async (req: ListOwnSectionsRequest) => ({
      sections: req.query === "MAT" ? [stubSections[0]] : stubSections,
      nextPageToken: "",
    }));

    const transport = makeStubTransport([CatalogService, { listOwnSections }]);

    renderComponent(<SectionSelectionTable onSelectSection={vi.fn()} />, {
      session: session(),
      transport,
    });

    // Wait for initial load.
    await screen.findByText("Matemáticas I");

    // Type into the search box.
    const input = screen.getByPlaceholderText("Buscar asignatura…");
    await user.type(input, "MAT");

    // Advance past the 300ms debounce.
    await vi.advanceTimersByTimeAsync(300);

    // useOwnSections should have been called with query="MAT" after the debounce.
    await waitFor(() => {
      const queryCalls = listOwnSections.mock.calls.map(
        ([req]) => (req as ListOwnSectionsRequest).query,
      );
      expect(queryCalls).toContain("MAT");
    });
  });

  it("shows no-results empty state when search yields no sections", async () => {
    const user = userEvent.setup({
      advanceTimers: vi.advanceTimersByTime.bind(vi),
    });

    const transport = makeStubTransport([
      CatalogService,
      {
        listOwnSections: async () => ({ sections: [], nextPageToken: "" }),
      },
    ]);

    renderComponent(<SectionSelectionTable onSelectSection={vi.fn()} />, {
      session: session(),
      transport,
    });

    // Type a query to activate the search-specific empty state.
    const input = screen.getByPlaceholderText("Buscar asignatura…");
    await user.type(input, "XYZ");
    await vi.advanceTimersByTimeAsync(300);

    await screen.findByText("No se encontraron secciones para la búsqueda.");
  });
});
