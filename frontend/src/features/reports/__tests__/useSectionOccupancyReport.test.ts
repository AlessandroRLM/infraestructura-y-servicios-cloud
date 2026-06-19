/**
 * useSectionOccupancyReport hook tests — RF-9.2, AC-2.f, RF-3.6
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
  GetSectionOccupancyReportResponseSchema,
  SectionOccupancyRowSchema,
} from "@/gen/reports/v1/reports_pb";
import {
  ReportsService,
  useSectionOccupancyReport,
} from "../hooks/useSectionOccupancyReport";

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
  return create(GetSectionOccupancyReportResponseSchema, {
    academicPeriodId: "period-uuid-1",
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

describe("useSectionOccupancyReport", () => {
  it("AC-2.f: enabled=false when periodId is empty → no transport call", () => {
    const getSectionOccupancyReport = vi.fn();
    const transport = makeStubTransport([
      ReportsService,
      { getSectionOccupancyReport },
    ]);

    const { result } = renderHook(() => useSectionOccupancyReport("", true), {
      wrapper: makeWrapper(transport),
    });

    expect(result.current.data).toBeNull();
    expect(result.current.isLoading).toBe(false);
    expect(getSectionOccupancyReport).not.toHaveBeenCalled();
  });

  it("AC-2.f: enabled=false when tab is not active → no transport call", () => {
    const getSectionOccupancyReport = vi.fn();
    const transport = makeStubTransport([
      ReportsService,
      { getSectionOccupancyReport },
    ]);

    const { result } = renderHook(
      () => useSectionOccupancyReport("period-uuid-1", false),
      { wrapper: makeWrapper(transport) },
    );

    expect(result.current.data).toBeNull();
    expect(getSectionOccupancyReport).not.toHaveBeenCalled();
  });

  it("returns rows, generatedAt, truncated=false when query succeeds", async () => {
    const transport = makeStubTransport([
      ReportsService,
      {
        getSectionOccupancyReport: async () => makeSuccessResponse(false),
      },
    ]);

    const { result } = renderHook(
      () => useSectionOccupancyReport("period-uuid-1", true),
      { wrapper: makeWrapper(transport) },
    );

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    await waitFor(() => expect(result.current.data).not.toBeNull());

    expect(result.current.rows).toHaveLength(1);
    expect(result.current.generatedAt).toBe("2026-06-18T10:00:00Z");
    expect(result.current.truncated).toBe(false);
    expect(result.current.academicPeriodName).toBe("2026 · Semestre 1");
    expect(result.current.isError).toBe(false);
  });

  it("truncated=true is propagated from the response", async () => {
    const transport = makeStubTransport([
      ReportsService,
      {
        getSectionOccupancyReport: async () => makeSuccessResponse(true),
      },
    ]);

    const { result } = renderHook(
      () => useSectionOccupancyReport("period-uuid-1", true),
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
        getSectionOccupancyReport: async () => {
          throw new ConnectError("unavailable", Code.Unavailable);
        },
      },
    ]);

    const { result } = renderHook(
      () => useSectionOccupancyReport("period-uuid-1", true),
      { wrapper: makeWrapper(transport) },
    );

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.rows).toHaveLength(0);
    expect(result.current.error).not.toBeNull();
  });
});
