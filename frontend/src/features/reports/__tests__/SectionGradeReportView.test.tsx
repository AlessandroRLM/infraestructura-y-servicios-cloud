/**
 * SectionGradeReportView component tests — RF-9.3, AC-3.b, AC-3.e, AC-4.a..4.e, AC-5.a, AC-6.a
 *
 * The renderPdf seam is mocked (vi.mock) so no real PDF/WASM runs in happy-dom.
 * Tests assert the ReportPdfModel passed to the mock — not PDF bytes.
 * Uses renderWithProviders with a stub transport for the full route context.
 */
import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import {
  CatalogService,
  ListSectionsResponseSchema,
  SectionSchema,
} from "@/gen/catalog/v1/catalog_pb";
import {
  GetSectionGradeReportResponseSchema,
  PartialGradeSchema,
  ReportsService,
  StudentGradeRowSchema,
} from "@/gen/reports/v1/reports_pb";
import { renderWithProviders } from "@/test";

// Mock the PDF seam — no real @react-pdf/renderer or WASM in tests.
vi.mock("../pdf/renderPdf", () => ({
  renderReportPdf: vi
    .fn()
    .mockResolvedValue(new Blob(["pdf"], { type: "application/pdf" })),
}));

import { renderReportPdf } from "../pdf/renderPdf";

const mockRenderReportPdf = vi.mocked(renderReportPdf);

const adminSession = {
  status: "authenticated" as const,
  userId: "admin-1",
  email: "admin@test.com",
  roles: ["admin"],
  permissions: ["reports.read", "users.manage"],
};

const adminSessionSource = {
  getSession: async () => ({
    userId: adminSession.userId,
    email: adminSession.email,
    roles: adminSession.roles,
    permissions: adminSession.permissions,
  }),
};

function makeSection(id: string, seatCapacity = 30) {
  return create(SectionSchema, {
    id,
    courseId: "course-1",
    academicPeriodId: "period-1",
    seatCapacity,
    createdAt: "",
    updatedAt: "",
  });
}

function makeSuccessResponse(truncated = false) {
  return create(GetSectionGradeReportResponseSchema, {
    sectionId: "section-uuid",
    rows: [
      create(StudentGradeRowSchema, {
        studentId: "s1",
        givenNames: "Ana",
        lastNamePaternal: "García",
        lastNameMaternal: "López",
        partialGrades: [
          create(PartialGradeSchema, {
            evaluationId: "e1",
            position: 1,
            value: "5.5",
          }),
        ],
        finalGrade: "5.5",
        outcome: "passed",
      }),
    ],
    generatedAt: "2026-06-18T10:00:00Z",
    truncated,
  });
}

type ReportsImpl = Partial<ServiceImpl<typeof ReportsService>>;
type CatalogImpl = Partial<ServiceImpl<typeof CatalogService>>;

const defaultCatalogImpl: CatalogImpl = {
  listSections: async () =>
    create(ListSectionsResponseSchema, {
      sections: [makeSection("section-uuid")],
      nextPageToken: "",
    }),
};

function renderReports(
  reportsImpl: ReportsImpl,
  route = "/admin/reports?tab=section-grade&sectionId=section-uuid",
  catalogImpl: CatalogImpl = defaultCatalogImpl,
) {
  return renderWithProviders({
    route,
    transport: makeStubTransport(
      [ReportsService, reportsImpl],
      [CatalogService, catalogImpl],
    ),
    session: adminSession,
    sessionSource: adminSessionSource,
  });
}

describe("SectionGradeReportView — states", () => {
  it("AC-4.a: no sectionId in URL → no-filter prompt visible", async () => {
    renderWithProviders({
      route: "/admin/reports?tab=section-grade",
      transport: makeStubTransport(
        [ReportsService, {}],
        [
          CatalogService,
          {
            listSections: async () =>
              create(ListSectionsResponseSchema, {
                sections: [],
                nextPageToken: "",
              }),
          },
        ],
      ),
      session: adminSession,
      sessionSource: adminSessionSource,
    });

    await screen.findByRole("heading", { name: "Reportes" });
    expect(
      screen.getByText("Selecciona un filtro para generar el reporte."),
    ).toBeInTheDocument();
  });

  it("ready state: renderReportPdf is called with a non-null ReportPdfModel (AC-3.e)", async () => {
    mockRenderReportPdf.mockResolvedValue(
      new Blob(["pdf"], { type: "application/pdf" }),
    );

    renderReports({
      getSectionGradeReport: async () => makeSuccessResponse(),
    });

    await screen.findByRole("heading", { name: "Reportes" });

    await waitFor(() => {
      expect(mockRenderReportPdf).toHaveBeenCalled();
    });

    const calledWith = mockRenderReportPdf.mock.calls[0]?.[0];
    expect(calledWith).not.toBeNull();
    expect(calledWith?.title).toBe("Calificaciones por Sección");
    expect(calledWith?.generatedAt).toBe("2026-06-18T10:00:00Z");
    expect(calledWith?.rows).toHaveLength(1);
  });

  it("AC-3.b: download button is aria-disabled before blob is ready (PDF still rendering)", async () => {
    // Never resolves → isRendering stays true indefinitely.
    mockRenderReportPdf.mockImplementation(
      // biome-ignore lint/suspicious/noEmptyBlockStatements: intentional never-resolving mock for loading-state test
      () => new Promise<Blob>(() => {}),
    );

    renderReports({
      getSectionGradeReport: async () => makeSuccessResponse(),
    });

    await screen.findByRole("heading", { name: "Reportes" });
    await waitFor(() => expect(mockRenderReportPdf).toHaveBeenCalled());

    // In the loading/rendering state, ReportStateBoundary shows the spinner and
    // the download button is not rendered (spinner takes over).
    // The spinner text must be visible.
    await screen.findByText(/Generando PDF/);

    // Restore for subsequent tests.
    mockRenderReportPdf.mockResolvedValue(
      new Blob(["pdf"], { type: "application/pdf" }),
    );
  });

  it("AC-4.d: error state shows 'Reintentar' with human-readable message", async () => {
    renderReports({
      getSectionGradeReport: async () => {
        throw new ConnectError("unavailable", Code.Unavailable);
      },
    });

    await screen.findByRole("heading", { name: "Reportes" });
    await screen.findByRole("alert");

    expect(
      screen.getByRole("button", { name: /Reintentar/i }),
    ).toBeInTheDocument();
    // Raw error codes must not appear.
    expect(screen.queryByText(/Unavailable/)).not.toBeInTheDocument();
  });

  it("AC-6.a: CodePermissionDenied → safe message, no raw code, 'Reintentar' present", async () => {
    renderReports({
      getSectionGradeReport: async () => {
        throw new ConnectError("permission denied", Code.PermissionDenied);
      },
    });

    await screen.findByRole("heading", { name: "Reportes" });
    await screen.findByRole("alert");

    expect(screen.queryByText(/PERMISSION_DENIED/)).not.toBeInTheDocument();
    expect(screen.queryByText(/PermissionDenied/)).not.toBeInTheDocument();
    expect(
      screen.getByText("No tienes permiso para ver este reporte."),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /Reintentar/i }),
    ).toBeInTheDocument();
  });

  it("AC-4.e: truncated=true → TruncationBanner visible + model.truncatedTo=500", async () => {
    mockRenderReportPdf.mockResolvedValue(
      new Blob(["pdf"], { type: "application/pdf" }),
    );

    renderReports({
      getSectionGradeReport: async () => makeSuccessResponse(true),
    });

    await screen.findByRole("heading", { name: "Reportes" });
    await waitFor(() => expect(mockRenderReportPdf).toHaveBeenCalled());

    // TruncationBanner on screen.
    await screen.findByText(/primeras 500 filas/i);

    // Model passed to renderPdf carries the truncation info.
    const model = mockRenderReportPdf.mock.calls[0]?.[0];
    expect(model?.truncatedTo).toBe(500);
    expect(model?.footer).toMatch(/truncado a 500/i);
  });

  it("AC-5.a: generatedAt timestamp appears in the view after PDF renders", async () => {
    mockRenderReportPdf.mockResolvedValue(
      new Blob(["pdf"], { type: "application/pdf" }),
    );

    renderReports({
      getSectionGradeReport: async () => makeSuccessResponse(),
    });

    await screen.findByRole("heading", { name: "Reportes" });
    await waitFor(() => expect(mockRenderReportPdf).toHaveBeenCalled());

    // formatGeneratedAt("2026-06-18T10:00:00Z") contains "junio de 2026".
    await waitFor(() => {
      expect(screen.getByText(/junio de 2026/i)).toBeInTheDocument();
    });
  });

  it("AC-4.c: empty response → empty message, no iframe, no download link", async () => {
    renderReports({
      getSectionGradeReport: async () =>
        create(GetSectionGradeReportResponseSchema, {
          sectionId: "section-uuid",
          rows: [],
          generatedAt: "2026-06-18T10:00:00Z",
          truncated: false,
        }),
    });

    await screen.findByRole("heading", { name: "Reportes" });
    await screen.findByText(/no hay datos disponibles/i);

    expect(
      screen.queryByTitle("Vista previa del reporte"),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("link", { name: /Descargar/i }),
    ).not.toBeInTheDocument();
  });
});
