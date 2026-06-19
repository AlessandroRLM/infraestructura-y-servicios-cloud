/**
 * ProgramSummaryReportView component tests — RF-9.3, AC-3.b, AC-3.e, AC-4.a..4.e, AC-5.a, AC-6.a
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
  CatalogService,
  ListProgramsResponseSchema,
  ProgramSchema,
} from "@/gen/catalog/v1/catalog_pb";
import {
  GetProgramSummaryReportResponseSchema,
  ProgramEnrollmentRowSchema,
  ReportsService,
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

function makeProgram(id: string, code: string, name: string) {
  return create(ProgramSchema, {
    id,
    code,
    name,
    createdAt: "",
    updatedAt: "",
  });
}

function makeSuccessResponse(truncated = false) {
  return create(GetProgramSummaryReportResponseSchema, {
    programId: "program-uuid",
    programName: "Ingeniería Civil",
    year: 2026,
    rows: [
      create(ProgramEnrollmentRowSchema, {
        quotaId: "quota-1",
        quotaCapacity: 100,
        enrolledCount: 80,
        availableSeats: 20,
        fillPercentage: "80.0",
      }),
    ],
    generatedAt: "2026-06-18T10:00:00Z",
    truncated,
  });
}

type ReportsImpl = Partial<ServiceImpl<typeof ReportsService>>;
type CatalogImpl = Partial<ServiceImpl<typeof CatalogService>>;

const defaultCatalogImpl: CatalogImpl = {
  listPrograms: async () =>
    create(ListProgramsResponseSchema, {
      programs: [makeProgram("program-uuid", "ICI", "Ingeniería Civil")],
      nextPageToken: "",
    }),
  listAcademicPeriods: async () =>
    create(
      (await import("@/gen/catalog/v1/catalog_pb"))
        .ListAcademicPeriodsResponseSchema,
      { academicPeriods: [] },
    ),
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

function renderProgramSummary(
  reportsImpl: ReportsImpl,
  route = "/admin/reports?tab=program-summary&programId=program-uuid&year=2026",
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

describe("ProgramSummaryReportView — states", () => {
  it("AC-4.a: no programId in URL → no-filter prompt visible", async () => {
    renderWithProviders({
      route: "/admin/reports?tab=program-summary",
      transport: makeStubTransport(
        [ReportsService, {}],
        [
          CatalogService,
          {
            listPrograms: async () =>
              create(ListProgramsResponseSchema, {
                programs: [],
                nextPageToken: "",
              }),
            listAcademicPeriods: async () =>
              create(
                (await import("@/gen/catalog/v1/catalog_pb"))
                  .ListAcademicPeriodsResponseSchema,
                { academicPeriods: [] },
              ),
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

    renderProgramSummary({
      getProgramSummaryReport: async () => makeSuccessResponse(),
    });

    await screen.findByRole("heading", { name: "Reportes" });

    await waitFor(() => {
      expect(mockRenderReportPdf).toHaveBeenCalled();
    });

    const calledWith = mockRenderReportPdf.mock.calls[0]?.[0];
    expect(calledWith).not.toBeNull();
    expect(calledWith?.title).toBe("Resumen de Programa");
    expect(calledWith?.generatedAt).toBe("2026-06-18T10:00:00Z");
    expect(calledWith?.rows).toHaveLength(1);
  });

  it("AC-4.d: error state shows 'Reintentar' with human-readable message", async () => {
    renderProgramSummary({
      getProgramSummaryReport: async () => {
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
    renderProgramSummary({
      getProgramSummaryReport: async () => {
        throw new ConnectError("permission denied", Code.PermissionDenied);
      },
    });

    await screen.findByRole("heading", { name: "Reportes" });
    await screen.findByRole("alert");

    expect(screen.queryByText(/PERMISSION_DENIED/)).not.toBeInTheDocument();
    expect(
      screen.getByText("No tienes permiso para ver este reporte."),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /Reintentar/i }),
    ).toBeInTheDocument();
  });

  it("AC-4.e: truncated=true → TruncationBanner visible + model.truncatedTo=200", async () => {
    mockRenderReportPdf.mockResolvedValue(
      new Blob(["pdf"], { type: "application/pdf" }),
    );

    renderProgramSummary({
      getProgramSummaryReport: async () => makeSuccessResponse(true),
    });

    await screen.findByRole("heading", { name: "Reportes" });
    await waitFor(() => expect(mockRenderReportPdf).toHaveBeenCalled());

    await screen.findByText(/primeras 200 filas/i);

    const model = mockRenderReportPdf.mock.calls[0]?.[0];
    expect(model?.truncatedTo).toBe(200);
    expect(model?.footer).toMatch(/truncado a 200/i);
  });

  it("AC-4.c: empty response → empty message, no iframe, no download link", async () => {
    renderProgramSummary({
      getProgramSummaryReport: async () =>
        create(GetProgramSummaryReportResponseSchema, {
          programId: "program-uuid",
          programName: "Ingeniería Civil",
          year: 2026,
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

  it("AC-5.a: generatedAt timestamp appears in the view after PDF renders", async () => {
    mockRenderReportPdf.mockResolvedValue(
      new Blob(["pdf"], { type: "application/pdf" }),
    );

    renderProgramSummary({
      getProgramSummaryReport: async () => makeSuccessResponse(),
    });

    await screen.findByRole("heading", { name: "Reportes" });
    await waitFor(() => expect(mockRenderReportPdf).toHaveBeenCalled());

    await waitFor(() => {
      expect(screen.getByText(/junio de 2026/i)).toBeInTheDocument();
    });
  });
});
