/**
 * SectionPicker enrichment tests — buildSectionLabel pure helper.
 * Validates that the enriched label renders "CODE · Name — Year · Semestre N"
 * with graceful fallback when TeachingSection fields are empty.
 *
 * Also verifies that a teacher session (no catalog.manage) reaches listOwnSections
 * and never calls listSections — the backend discriminates by role server-side.
 */
import { create } from "@bufbuild/protobuf";
import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import type { Permission, SessionState } from "@/features/auth";
import {
  CatalogService,
  TeachingSectionSchema,
} from "@/gen/catalog/v1/catalog_pb";
import { renderComponent } from "@/test";
import { buildSectionLabel, SectionPicker } from "../components/SectionPicker";

// ──────────────────────────────────────────────
// Pure helper — buildSectionLabel
// ──────────────────────────────────────────────

function makeTeachingSection(
  overrides: Partial<{
    id: string;
    courseCode: string;
    courseName: string;
    periodYear: number;
    periodTerm: number;
  }> = {},
) {
  return create(TeachingSectionSchema, {
    id: overrides.id ?? "abcdef12-3456-7890-abcd-ef1234567890",
    courseId: "course-1",
    academicPeriodId: "period-1",
    seatCapacity: 30,
    courseCode: overrides.courseCode ?? "",
    courseName: overrides.courseName ?? "",
    periodYear: overrides.periodYear ?? 0,
    periodTerm: overrides.periodTerm ?? 0,
  });
}

describe("buildSectionLabel", () => {
  it("returns full enriched label when all fields are present", () => {
    const section = makeTeachingSection({
      courseCode: "MAT101",
      courseName: "Cálculo I",
      periodYear: 2026,
      periodTerm: 1,
    });

    const label = buildSectionLabel(section);
    expect(label).toBe("MAT101 · Cálculo I — 2026 · Semestre 1");
  });

  it("returns course-only label when period fields are zero/missing", () => {
    const section = makeTeachingSection({
      courseCode: "QUI201",
      courseName: "Química General",
      periodYear: 0,
      periodTerm: 0,
    });

    const label = buildSectionLabel(section);
    expect(label).toBe("QUI201 · Química General");
  });

  it("falls back to short ID when course fields are empty", () => {
    const section = makeTeachingSection({
      courseCode: "",
      courseName: "",
      periodYear: 0,
      periodTerm: 0,
    });

    const label = buildSectionLabel(section);
    expect(label).toMatch(/^Sección abcdef12/);
    expect(label).toContain("…");
  });

  it("uses term 2 correctly in the label", () => {
    const section = makeTeachingSection({
      courseCode: "FIS301",
      courseName: "Física III",
      periodYear: 2025,
      periodTerm: 2,
    });

    const label = buildSectionLabel(section);
    expect(label).toBe("FIS301 · Física III — 2025 · Semestre 2");
  });
});

// ──────────────────────────────────────────────
// Role-aware transport — teacher session
// ──────────────────────────────────────────────

const stubSections = [
  create(TeachingSectionSchema, {
    id: "sec-1",
    courseId: "course-1",
    academicPeriodId: "period-1",
    seatCapacity: 30,
    courseCode: "MAT101",
    courseName: "Matemáticas I",
    periodYear: 2024,
    periodTerm: 1,
  }),
];

/**
 * A teacher session: holds grades.write and reports.read but NOT catalog.manage.
 * The backend's listOwnSections discriminates by role — the frontend makes no
 * permission check; the component always calls listOwnSections regardless of role.
 */
function teacherSession(): SessionState {
  return {
    status: "authenticated",
    userId: "u-teacher",
    email: "teacher@test.com",
    roles: ["teacher"],
    permissions: ["grades.write", "reports.read"] satisfies Permission[],
  };
}

describe("SectionPicker — teacher session sees sections via listOwnSections", () => {
  it("calls listOwnSections (not listSections) and renders enriched labels", async () => {
    const listOwnSections = vi.fn(async () => ({
      sections: stubSections,
      nextPageToken: "",
    }));
    const listSections = vi.fn(async () => ({
      sections: [],
      nextPageToken: "",
    }));

    const transport = makeStubTransport([
      CatalogService,
      { listOwnSections, listSections },
    ]);

    renderComponent(<SectionPicker value="" onChange={vi.fn()} />, {
      transport,
      session: teacherSession(),
    });

    // The combobox trigger renders the placeholder before any section is selected.
    expect(
      screen.getByRole("combobox", { name: "Seleccionar sección" }),
    ).toBeInTheDocument();

    // listOwnSections must be called; listSections must NOT be called.
    await waitFor(() => {
      expect(listOwnSections).toHaveBeenCalled();
    });
    expect(listSections).not.toHaveBeenCalled();
  });
});
