/**
 * usePrograms hook tests — server-side search forwarding.
 * Verifies that the query argument is passed through to the ListPrograms RPC.
 */
import { create } from "@bufbuild/protobuf";
import { TransportProvider } from "@connectrpc/connect-query";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { createElement } from "react";
import { describe, expect, it, vi } from "vitest";
import { usePrograms } from "@/core/catalog";
import { makeStubTransport } from "@/core/test";
import {
  CatalogService,
  ListProgramsResponseSchema,
  ProgramSchema,
} from "@/gen/catalog/v1/catalog_pb";

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

function makeProgram(id: string, code: string, name: string) {
  return create(ProgramSchema, {
    id,
    code,
    name,
    createdAt: "",
    updatedAt: "",
  });
}

describe("usePrograms — server-side search", () => {
  it("forwards the query string to the listPrograms RPC request", async () => {
    const listPrograms = vi.fn(async () =>
      create(ListProgramsResponseSchema, {
        programs: [makeProgram("p1", "ICI", "Ingeniería Civil")],
        nextPageToken: "",
      }),
    );

    const transport = makeStubTransport([CatalogService, { listPrograms }]);

    const { result } = renderHook(() => usePrograms("civil"), {
      wrapper: makeWrapper(transport),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(listPrograms).toHaveBeenCalled();
    const calls = listPrograms.mock.calls as unknown as Array<
      [{ query: string }]
    >;
    expect(calls[0]?.[0].query).toBe("civil");
  });

  it("returns programs from the RPC response", async () => {
    const transport = makeStubTransport([
      CatalogService,
      {
        listPrograms: async () =>
          create(ListProgramsResponseSchema, {
            programs: [
              makeProgram("p1", "ICI", "Ingeniería Civil"),
              makeProgram("p2", "ICE", "Ingeniería Eléctrica"),
            ],
            nextPageToken: "",
          }),
      },
    ]);

    const { result } = renderHook(() => usePrograms(""), {
      wrapper: makeWrapper(transport),
    });

    await waitFor(() => expect(result.current.programs).toHaveLength(2));
    expect(result.current.programs[0]?.code).toBe("ICI");
  });

  it("defaults query to empty string when called without args", async () => {
    const listPrograms = vi.fn(async () =>
      create(ListProgramsResponseSchema, { programs: [], nextPageToken: "" }),
    );

    const transport = makeStubTransport([CatalogService, { listPrograms }]);

    renderHook(() => usePrograms(), { wrapper: makeWrapper(transport) });

    await waitFor(() => expect(listPrograms).toHaveBeenCalled());
    const calls = listPrograms.mock.calls as unknown as Array<
      [{ query: string }]
    >;
    expect(calls[0]?.[0].query).toBe("");
  });
});
