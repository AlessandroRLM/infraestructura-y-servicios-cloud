/**
 * useStudentRecordReport hook tests — RF-9.2, AC-2.f, RF-3.6
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
  AcademicRecordRowSchema,
  GetStudentRecordReportResponseSchema,
} from "@/gen/reports/v1/reports_pb";
import {
  ReportsService,
  useStudentRecordReport,
} from "../hooks/useStudentRecordReport";

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

function makeSuccessResponse(truncated = false) {
  return create(GetStudentRecordReportResponseSchema, {
    studentId: "student-uuid-1",
    studentName: "Ana García",
    rows: [
      create(AcademicRecordRowSchema, {
        academicPeriodId: "period-uuid-1",
        academicPeriodName: "2026 · Semestre 1",
        sectionId: "section-uuid-1",
        courseName: "Cálculo I",
        enrollmentStatus: "passed",
        finalGrade: "6.5",
        outcome: "passed",
      }),
    ],
    generatedAt: "2026-06-18T10:00:00Z",
    truncated,
  });
}

describe("useStudentRecordReport", () => {
  it("AC-2.f: enabled=false when studentId is empty → no transport call", () => {
    const getStudentRecordReport = vi.fn();
    const transport = makeStubTransport([
      ReportsService,
      { getStudentRecordReport },
    ]);

    const { result } = renderHook(() => useStudentRecordReport("", true), {
      wrapper: makeWrapper(transport),
    });

    expect(result.current.data).toBeNull();
    expect(result.current.isLoading).toBe(false);
    expect(getStudentRecordReport).not.toHaveBeenCalled();
  });

  it("AC-2.f: enabled=false when tab is not active → no transport call", () => {
    const getStudentRecordReport = vi.fn();
    const transport = makeStubTransport([
      ReportsService,
      { getStudentRecordReport },
    ]);

    const { result } = renderHook(
      () => useStudentRecordReport("student-uuid-1", false),
      { wrapper: makeWrapper(transport) },
    );

    expect(result.current.data).toBeNull();
    expect(getStudentRecordReport).not.toHaveBeenCalled();
  });

  it("returns rows, generatedAt, truncated=false, studentName when query succeeds", async () => {
    const transport = makeStubTransport([
      ReportsService,
      {
        getStudentRecordReport: async () => makeSuccessResponse(false),
      },
    ]);

    const { result } = renderHook(
      () => useStudentRecordReport("student-uuid-1", true),
      { wrapper: makeWrapper(transport) },
    );

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    await waitFor(() => expect(result.current.data).not.toBeNull());

    expect(result.current.rows).toHaveLength(1);
    expect(result.current.generatedAt).toBe("2026-06-18T10:00:00Z");
    expect(result.current.truncated).toBe(false);
    expect(result.current.studentName).toBe("Ana García");
    expect(result.current.isError).toBe(false);
  });

  it("truncated=true is propagated from the response", async () => {
    const transport = makeStubTransport([
      ReportsService,
      {
        getStudentRecordReport: async () => makeSuccessResponse(true),
      },
    ]);

    const { result } = renderHook(
      () => useStudentRecordReport("student-uuid-1", true),
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
        getStudentRecordReport: async () => {
          throw new ConnectError("unavailable", Code.Unavailable);
        },
      },
    ]);

    const { result } = renderHook(
      () => useStudentRecordReport("student-uuid-1", true),
      { wrapper: makeWrapper(transport) },
    );

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.rows).toHaveLength(0);
    expect(result.current.error).not.toBeNull();
  });
});
