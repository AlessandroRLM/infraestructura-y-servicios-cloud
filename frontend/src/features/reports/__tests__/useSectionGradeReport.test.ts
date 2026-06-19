/**
 * useSectionGradeReport hook tests — RF-9.2, AC-2.f, RF-3.6
 *
 * Uses makeStubTransport so the hook integrates against the real generated proto types.
 * No mocking of the seam needed — this test operates at the data layer only.
 */
import { create } from "@bufbuild/protobuf";
import { TransportProvider } from "@connectrpc/connect-query";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { createElement } from "react";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import {
  GetSectionGradeReportResponseSchema,
  PartialGradeSchema,
  StudentGradeRowSchema,
} from "@/gen/reports/v1/reports_pb";
import {
  ReportsService,
  useSectionGradeReport,
} from "../hooks/useSectionGradeReport";

function makeWrapper(transport: ReturnType<typeof makeStubTransport>) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0, staleTime: 0 } },
  });
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(
      TransportProvider,
      { transport },
      createElement(QueryClientProvider, { client: queryClient }, children),
    );
  };
}

function makeRow(position: number, value: string) {
  return create(StudentGradeRowSchema, {
    studentId: `s${position}`,
    givenNames: "Ana",
    lastNamePaternal: "García",
    lastNameMaternal: "López",
    partialGrades: [
      create(PartialGradeSchema, {
        evaluationId: `eval-${position}`,
        position,
        value,
      }),
    ],
    finalGrade: value,
    outcome: "passed",
  });
}

function makeSuccessResponse(truncated = false) {
  return create(GetSectionGradeReportResponseSchema, {
    sectionId: "section-uuid-abc",
    rows: [makeRow(1, "5.5"), makeRow(2, "6.0")],
    generatedAt: "2026-06-18T10:00:00Z",
    truncated,
  });
}

describe("useSectionGradeReport", () => {
  it("AC-2.f: enabled=false when sectionId is empty → no transport call", () => {
    const getSectionGradeReport = vi.fn();
    const transport = makeStubTransport([
      ReportsService,
      { getSectionGradeReport },
    ]);

    const { result } = renderHook(() => useSectionGradeReport("", true), {
      wrapper: makeWrapper(transport),
    });

    // Query is disabled — no data, no loading, no error.
    expect(result.current.data).toBeNull();
    expect(result.current.isLoading).toBe(false);
    expect(getSectionGradeReport).not.toHaveBeenCalled();
  });

  it("AC-2.f: enabled=false when tab is not active → no transport call", () => {
    const getSectionGradeReport = vi.fn();
    const transport = makeStubTransport([
      ReportsService,
      { getSectionGradeReport },
    ]);

    const { result } = renderHook(
      () => useSectionGradeReport("section-uuid-abc", false),
      { wrapper: makeWrapper(transport) },
    );

    expect(result.current.data).toBeNull();
    expect(getSectionGradeReport).not.toHaveBeenCalled();
  });

  it("returns rows, generatedAt, truncated=false when query succeeds", async () => {
    const transport = makeStubTransport([
      ReportsService,
      {
        getSectionGradeReport: async () => makeSuccessResponse(false),
      },
    ]);

    const { result } = renderHook(
      () => useSectionGradeReport("section-uuid-abc", true),
      { wrapper: makeWrapper(transport) },
    );

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    await waitFor(() => expect(result.current.data).not.toBeNull());

    expect(result.current.rows).toHaveLength(2);
    expect(result.current.generatedAt).toBe("2026-06-18T10:00:00Z");
    expect(result.current.truncated).toBe(false);
    expect(result.current.isError).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it("truncated=true is propagated from the response", async () => {
    const transport = makeStubTransport([
      ReportsService,
      {
        getSectionGradeReport: async () => makeSuccessResponse(true),
      },
    ]);

    const { result } = renderHook(
      () => useSectionGradeReport("section-uuid-abc", true),
      { wrapper: makeWrapper(transport) },
    );

    await waitFor(() => expect(result.current.data).not.toBeNull());
    expect(result.current.truncated).toBe(true);
  });

  it("error state: isError=true, error is set, rows is empty", async () => {
    const { ConnectError, Code } = await import("@connectrpc/connect");
    const transport = makeStubTransport([
      ReportsService,
      {
        getSectionGradeReport: async () => {
          throw new ConnectError("unavailable", Code.Unavailable);
        },
      },
    ]);

    const { result } = renderHook(
      () => useSectionGradeReport("section-uuid-abc", true),
      { wrapper: makeWrapper(transport) },
    );

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.rows).toHaveLength(0);
    expect(result.current.error).not.toBeNull();
  });
});
