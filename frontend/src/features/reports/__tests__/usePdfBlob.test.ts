/**
 * usePdfBlob lifecycle tests — AC-7.a, AC-7.b, AC-3.a
 *
 * Mocks the renderReportPdf seam so no real PDF/WASM execution occurs.
 * Spies on URL.createObjectURL and URL.revokeObjectURL to assert the blob
 * lifecycle contract.
 */
import { renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ReportPdfModel } from "../pdf/model";

// Mock the seam before importing the hook.
vi.mock("../pdf/renderPdf", () => ({
  renderReportPdf: vi.fn(),
}));

import { usePdfBlob } from "../hooks/usePdfBlob";
import { renderReportPdf } from "../pdf/renderPdf";

const mockRenderReportPdf = vi.mocked(renderReportPdf);

function makeModel(title: string): ReportPdfModel {
  return {
    title,
    appliedFilter: "test filter",
    generatedAt: "2026-06-17T20:00:00Z",
    columns: [{ key: "col", label: "Col", width: 100 }],
    rows: [["value"]],
    footer: "Test footer",
  };
}

function makeBlob(label: string) {
  return new Blob([label], { type: "application/pdf" });
}

describe("usePdfBlob", () => {
  // biome-ignore lint/suspicious/noExplicitAny: spy type is intentionally broad for cleanup
  let revokeObjectUrlSpy: any;
  // biome-ignore lint/suspicious/noExplicitAny: spy type is intentionally broad for cleanup
  let createObjectUrlSpy: any;
  let urlCounter = 0;

  beforeEach(() => {
    urlCounter = 0;
    // Reset the seam mock to a default resolved state so any test that
    // doesn't configure it explicitly still gets a functional mock.
    mockRenderReportPdf.mockResolvedValue(makeBlob("default"));
    createObjectUrlSpy = vi
      .spyOn(URL, "createObjectURL")
      .mockImplementation(() => {
        urlCounter++;
        return `blob:test-url-${urlCounter}`;
      });
    revokeObjectUrlSpy = vi
      .spyOn(URL, "revokeObjectURL")
      .mockImplementation(() => undefined);
  });

  afterEach(() => {
    // Selectively restore only the URL spies created in beforeEach.
    // Calling vi.restoreAllMocks() would also restore the vi.mock module
    // mocks, leaving renderReportPdf as () => undefined and breaking
    // subsequent tests across files.
    createObjectUrlSpy.mockRestore();
    revokeObjectUrlSpy.mockRestore();
  });

  it("AC-3.a: model null → renderReportPdf is never called, url is null", () => {
    mockRenderReportPdf.mockResolvedValue(makeBlob("a"));

    const { result } = renderHook(() => usePdfBlob(null));

    expect(mockRenderReportPdf).not.toHaveBeenCalled();
    expect(result.current.url).toBeNull();
    expect(result.current.blob).toBeNull();
    expect(result.current.isRendering).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it("AC-7.b: unmounting with an active blob calls revokeObjectURL", async () => {
    const blobA = makeBlob("a");
    mockRenderReportPdf.mockResolvedValue(blobA);

    const { result, unmount } = renderHook(() =>
      usePdfBlob(makeModel("Model A")),
    );

    // Wait for the render to complete.
    await waitFor(() => expect(result.current.url).not.toBeNull());

    const activeUrl = result.current.url as string;

    // Unmount — cleanup should revoke the URL.
    unmount();

    expect(revokeObjectUrlSpy).toHaveBeenCalledWith(activeUrl);
  });

  it("AC-7.a: model A → model B: createObjectURL called twice, revokeObjectURL(firstUrl) before second url is exposed", async () => {
    const blobA = makeBlob("a");
    const blobB = makeBlob("b");

    mockRenderReportPdf
      .mockResolvedValueOnce(blobA)
      .mockResolvedValueOnce(blobB);

    const modelA = makeModel("Model A");
    const modelB = makeModel("Model B");

    // Record call order across both spies.
    const callOrder: string[] = [];
    createObjectUrlSpy.mockImplementation(() => {
      urlCounter++;
      const url = `blob:test-url-${urlCounter}`;
      callOrder.push(`create:${url}`);
      return url;
    });
    revokeObjectUrlSpy.mockImplementation((u: string) => {
      callOrder.push(`revoke:${u}`);
    });

    const { result, rerender, unmount } = renderHook(
      ({ model }: { model: ReportPdfModel | null }) => usePdfBlob(model),
      { initialProps: { model: modelA } },
    );

    // Wait for first URL.
    await waitFor(() => expect(result.current.url).not.toBeNull());
    const firstUrl = result.current.url as string;
    expect(firstUrl).toBe("blob:test-url-1");

    // Switch to model B — cleanup from first effect fires, revoking first URL.
    rerender({ model: modelB });

    // Wait for second URL to be set.
    await waitFor(() => expect(result.current.url).toBe("blob:test-url-2"));

    // createObjectURL must have been called exactly twice (A and B).
    expect(createObjectUrlSpy).toHaveBeenCalledTimes(2);

    // revokeObjectURL must have been called with the first URL.
    expect(revokeObjectUrlSpy).toHaveBeenCalledWith(firstUrl);

    // revoke(firstUrl) must appear in callOrder BEFORE create(blob:test-url-2).
    const revokeIdx = callOrder.indexOf(`revoke:${firstUrl}`);
    const createSecondIdx = callOrder.indexOf("create:blob:test-url-2");
    expect(revokeIdx).toBeGreaterThanOrEqual(0);
    expect(createSecondIdx).toBeGreaterThanOrEqual(0);
    expect(revokeIdx).toBeLessThan(createSecondIdx);

    // Cleanup.
    unmount();
  });

  it("isRendering transitions to false after the render completes", async () => {
    // The default beforeEach mock resolves immediately.
    // After render completes, isRendering must be false and url must be set.
    const { result } = renderHook(() => usePdfBlob(makeModel("Model C")));

    await waitFor(() => expect(result.current.isRendering).toBe(false));
    expect(result.current.url).not.toBeNull();
  });

  it("surfaces error when renderReportPdf rejects", async () => {
    const renderError = new Error("PDF render failed");
    mockRenderReportPdf.mockRejectedValue(renderError);

    const { result } = renderHook(() => usePdfBlob(makeModel("Model Err")));

    await waitFor(() => expect(result.current.error).not.toBeNull());
    expect(result.current.error?.message).toBe("PDF render failed");
    expect(result.current.isRendering).toBe(false);
    expect(result.current.url).toBeNull();
  });
});
