/**
 * SectionOccupancyReportView component tests — RF-9.3, AC-3.b, AC-3.e, AC-4.a..4.e, AC-5.a, AC-6.a
 *
 * renderPdf seam is mocked so no real WASM runs in happy-dom.
 * Transport is stubbed via makeStubTransport.
 */
import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import {
  AcademicPeriodSchema,
  CatalogService,
  ListAcademicPeriodsResponseSchema,
} from "@/gen/catalog/v1/catalog_pb";
import {
  GetSectionOccupancyReportResponseSchema,
  ReportsService,
  SectionOccupancyRowSchema,
} from "@/gen/reports/v1/reports_pb";
import { renderWithProviders } from "@/test";

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

function makePeriod(id: string, year: number, term: number) {
  return create(AcademicPeriodSchema, {
    id,
    year,
    term,
    startDate: "",
    endDate: "",
    createdAt: "",
    updatedAt: "",
  });
}

function makeSuccessResponse(truncated = false) {
  return create(GetSectionOccupancyReportResponseSchema, {
    academicPeriodId: "period-uuid",
    rows: [
      create(SectionOccupancyRowSchema, {
        sectionId: "section-uuid-1",
        courseName: "Cálculo I",
        capacity: 40,
        activeSeatCount: 30,
        fillPercentage: "75.0",
      }),
    ],
    generatedAt: "2026-06-18T10:00:00Z",
    truncated,
    academicPeriodName: "2026 · Semestre 1",
  });
}

type ReportsImpl = Partial<ServiceImpl<typeof ReportsService>>;
type CatalogImpl = Partial<ServiceImpl<typeof CatalogService>>;

const defaultCatalogImpl: CatalogImpl = {
  listAcademicPeriods: async () =>
    create(ListAcademicPeriodsResponseSchema, {
      academicPeriods: [makePeriod("period-uuid", 2026, 1)],
    }),
  listSections: async () =>
    create(
      (await import("@/gen/catalog/v1/catalog_pb")).ListSectionsResponseSchema,
      { sections: [], nextPageToken: "" },
    ),
  listCourses: async () =>
    create(
      (await import("@/gen/catalog/v1/catalog_pb")).ListCoursesResponseSchema,
      { courses: [], nextPageToken: "" },
    ),
};

function renderOccupancy(
  reportsImpl: ReportsImpl,
  route = "/admin/reports?tab=occupancy&periodId=period-uuid",
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

describe("SectionOccupancyReportView — states", () => {
  it("AC-4.a: no periodId in URL → no-filter prompt visible", async () => {
    renderWithProviders({
      route: "/admin/reports?tab=occupancy",
      transport: makeStubTransport(
        [ReportsService, {}],
        [
          CatalogService,
          {
            listAcademicPeriods: async () =>
              create(ListAcademicPeriodsResponseSchema, {
                academicPeriods: [],
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

    renderOccupancy({
      getSectionOccupancyReport: async () => makeSuccessResponse(),
    });

    await screen.findByRole("heading", { name: "Reportes" });

    await waitFor(() => {
      expect(mockRenderReportPdf).toHaveBeenCalled();
    });

    const calledWith = mockRenderReportPdf.mock.calls[0]?.[0];
    expect(calledWith).not.toBeNull();
    expect(calledWith?.title).toBe("Ocupación por Período");
    expect(calledWith?.generatedAt).toBe("2026-06-18T10:00:00Z");
    expect(calledWith?.rows).toHaveLength(1);
  });

  it("AC-4.d: error state shows 'Reintentar' with human-readable message", async () => {
    renderOccupancy({
      getSectionOccupancyReport: async () => {
        throw new ConnectError("unavailable", Code.Unavailable);
      },
    });

    await screen.findByRole("heading", { name: "Reportes" });
    await screen.findByRole("alert");

    expect(
      screen.getByRole("button", { name: /Reintentar/i }),
    ).toBeInTheDocument();
    expect(screen.queryByText(/Unavailable/)).not.toBeInTheDocument();
  });

  it("AC-6.a: CodePermissionDenied → safe message, no raw code, 'Reintentar' present", async () => {
    renderOccupancy({
      getSectionOccupancyReport: async () => {
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

  it("AC-4.e: truncated=true → TruncationBanner visible + model.truncatedTo=1000", async () => {
    mockRenderReportPdf.mockResolvedValue(
      new Blob(["pdf"], { type: "application/pdf" }),
    );

    renderOccupancy({
      getSectionOccupancyReport: async () => makeSuccessResponse(true),
    });

    await screen.findByRole("heading", { name: "Reportes" });
    await waitFor(() => expect(mockRenderReportPdf).toHaveBeenCalled());

    await screen.findByText(/primeras 1000 filas/i);

    const model = mockRenderReportPdf.mock.calls[0]?.[0];
    expect(model?.truncatedTo).toBe(1000);
    expect(model?.footer).toMatch(/truncado a 1000/i);
  });

  it("AC-4.c: empty response → empty message, no iframe, no download link", async () => {
    renderOccupancy({
      getSectionOccupancyReport: async () =>
        create(GetSectionOccupancyReportResponseSchema, {
          academicPeriodId: "period-uuid",
          rows: [],
          generatedAt: "2026-06-18T10:00:00Z",
          truncated: false,
          academicPeriodName: "2026 · Semestre 1",
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

  it("AC-5.a: generatedAt timestamp appears in the view after PDF renders", async () => {
    mockRenderReportPdf.mockResolvedValue(
      new Blob(["pdf"], { type: "application/pdf" }),
    );

    renderOccupancy({
      getSectionOccupancyReport: async () => makeSuccessResponse(),
    });

    await screen.findByRole("heading", { name: "Reportes" });
    await waitFor(() => expect(mockRenderReportPdf).toHaveBeenCalled());

    await waitFor(() => {
      expect(screen.getByText(/junio de 2026/i)).toBeInTheDocument();
    });
  });
});
