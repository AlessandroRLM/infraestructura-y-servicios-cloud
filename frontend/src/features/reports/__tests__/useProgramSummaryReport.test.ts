/**
 * useProgramSummaryReport hook tests — RF-9.2, AC-2.f, RF-3.6
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
  GetProgramSummaryReportResponseSchema,
  ProgramEnrollmentRowSchema,
} from "@/gen/reports/v1/reports_pb";
import {
  ReportsService,
  useProgramSummaryReport,
} from "../hooks/useProgramSummaryReport";

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
  return create(GetProgramSummaryReportResponseSchema, {
    programId: "program-uuid-1",
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

describe("useProgramSummaryReport", () => {
  it("AC-2.f: enabled=false when programId is empty → no transport call", () => {
    const getProgramSummaryReport = vi.fn();
    const transport = makeStubTransport([
      ReportsService,
      { getProgramSummaryReport },
    ]);

    const { result } = renderHook(
      () => useProgramSummaryReport("", 2026, true),
      { wrapper: makeWrapper(transport) },
    );

    expect(result.current.data).toBeNull();
    expect(result.current.isLoading).toBe(false);
    expect(getProgramSummaryReport).not.toHaveBeenCalled();
  });

  it("AC-2.f: enabled=false when year is undefined → no transport call", () => {
    const getProgramSummaryReport = vi.fn();
    const transport = makeStubTransport([
      ReportsService,
      { getProgramSummaryReport },
    ]);

    const { result } = renderHook(
      () => useProgramSummaryReport("program-uuid-1", undefined, true),
      { wrapper: makeWrapper(transport) },
    );

    expect(result.current.data).toBeNull();
    expect(getProgramSummaryReport).not.toHaveBeenCalled();
  });

  it("AC-2.f: enabled=false when tab is not active → no transport call", () => {
    const getProgramSummaryReport = vi.fn();
    const transport = makeStubTransport([
      ReportsService,
      { getProgramSummaryReport },
    ]);

    const { result } = renderHook(
      () => useProgramSummaryReport("program-uuid-1", 2026, false),
      { wrapper: makeWrapper(transport) },
    );

    expect(result.current.data).toBeNull();
    expect(getProgramSummaryReport).not.toHaveBeenCalled();
  });

  it("returns rows, generatedAt, truncated=false when query succeeds", async () => {
    const transport = makeStubTransport([
      ReportsService,
      {
        getProgramSummaryReport: async () => makeSuccessResponse(false),
      },
    ]);

    const { result } = renderHook(
      () => useProgramSummaryReport("program-uuid-1", 2026, true),
      { wrapper: makeWrapper(transport) },
    );

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    await waitFor(() => expect(result.current.data).not.toBeNull());

    expect(result.current.rows).toHaveLength(1);
    expect(result.current.generatedAt).toBe("2026-06-18T10:00:00Z");
    expect(result.current.truncated).toBe(false);
    expect(result.current.programName).toBe("Ingeniería Civil");
    expect(result.current.year).toBe(2026);
    expect(result.current.isError).toBe(false);
  });

  it("truncated=true is propagated from the response", async () => {
    const transport = makeStubTransport([
      ReportsService,
      {
        getProgramSummaryReport: async () => makeSuccessResponse(true),
      },
    ]);

    const { result } = renderHook(
      () => useProgramSummaryReport("program-uuid-1", 2026, true),
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
        getProgramSummaryReport: async () => {
          throw new ConnectError("unavailable", Code.Unavailable);
        },
      },
    ]);

    const { result } = renderHook(
      () => useProgramSummaryReport("program-uuid-1", 2026, true),
      { wrapper: makeWrapper(transport) },
    );

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.rows).toHaveLength(0);
    expect(result.current.error).not.toBeNull();
  });
});
