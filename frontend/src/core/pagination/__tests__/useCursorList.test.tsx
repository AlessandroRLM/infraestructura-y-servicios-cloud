import { create } from "@bufbuild/protobuf";
import { TransportProvider } from "@connectrpc/connect-query";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it } from "vitest";
import { makeStubTransport } from "@/core/test";
import {
  AuditLogsService,
  ListAuditLogsResponseSchema,
} from "@/gen/audit_logs/v1/audit_logs_pb";
import { useCursorList } from "../useCursorList";

function makeWrapper(transport: ReturnType<typeof makeStubTransport>) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0, staleTime: 0 } },
  });
  return function Wrapper({ children }: { children: ReactNode }) {
    return (
      <TransportProvider transport={transport}>
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      </TransportProvider>
    );
  };
}

describe("useCursorList", () => {
  it("stops pagination when nextPageToken is empty", async () => {
    const transport = makeStubTransport([
      AuditLogsService,
      {
        listAuditLogs: async () =>
          create(ListAuditLogsResponseSchema, {
            logs: [],
            nextPageToken: "",
          }),
      },
    ]);

    const { result } = renderHook(
      () =>
        useCursorList(AuditLogsService.method.listAuditLogs, {
          extract: (res) => ({
            items: res.logs,
            nextPageToken: res.nextPageToken,
          }),
          baseInput: { entity: "test", entityId: "1" },
        }),
      { wrapper: makeWrapper(transport) },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.hasNextPage).toBe(false);
  });

  it("accumulates items across pages via extract", async () => {
    let callCount = 0;
    const transport = makeStubTransport([
      AuditLogsService,
      {
        listAuditLogs: async () => {
          callCount++;
          if (callCount === 1) {
            return create(ListAuditLogsResponseSchema, {
              logs: [
                {
                  id: "1",
                  actorId: "",
                  action: "a",
                  entity: "test",
                  entityId: "1",
                  detail: "",
                  createdAt: "",
                },
              ],
              nextPageToken: "cursor-2",
            });
          }
          return create(ListAuditLogsResponseSchema, {
            logs: [
              {
                id: "2",
                actorId: "",
                action: "b",
                entity: "test",
                entityId: "1",
                detail: "",
                createdAt: "",
              },
            ],
            nextPageToken: "",
          });
        },
      },
    ]);

    const { result } = renderHook(
      () =>
        useCursorList(AuditLogsService.method.listAuditLogs, {
          extract: (res) => ({
            items: res.logs,
            nextPageToken: res.nextPageToken,
          }),
          baseInput: { entity: "test", entityId: "1" },
        }),
      { wrapper: makeWrapper(transport) },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.hasNextPage).toBe(true);
    expect(result.current.items).toHaveLength(1);

    await result.current.fetchNextPage();
    await waitFor(() => expect(result.current.items).toHaveLength(2));
    expect(result.current.hasNextPage).toBe(false);
  });

  it("passes pageSize and pageToken correctly per call", async () => {
    const calls: { pageSize: number; pageToken: string }[] = [];
    const transport = makeStubTransport([
      AuditLogsService,
      {
        listAuditLogs: async (req) => {
          calls.push({ pageSize: req.pageSize, pageToken: req.pageToken });
          return create(ListAuditLogsResponseSchema, {
            logs: [],
            nextPageToken: "",
          });
        },
      },
    ]);

    const { result } = renderHook(
      () =>
        useCursorList(AuditLogsService.method.listAuditLogs, {
          extract: (res) => ({
            items: res.logs,
            nextPageToken: res.nextPageToken,
          }),
          pageSize: 10,
          baseInput: { entity: "test", entityId: "1" },
        }),
      { wrapper: makeWrapper(transport) },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(calls[0].pageSize).toBe(10);
    expect(calls[0].pageToken).toBe("");
  });
});
