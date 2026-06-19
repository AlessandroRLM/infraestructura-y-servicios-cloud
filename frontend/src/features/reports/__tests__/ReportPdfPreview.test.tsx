/**
 * ReportPdfPreview component tests — AC-3.b, AC-7.c
 *
 * Mocks the renderReportPdf seam — no real PDF/WASM execution.
 * Asserts download control aria-disabled behavior and href = url from usePdfBlob.
 */
import { render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ReportPdfPreview } from "../components/ReportPdfPreview";

describe("ReportPdfPreview", () => {
  // biome-ignore lint/suspicious/noExplicitAny: spy type is intentionally broad for cleanup
  let createObjectUrlSpy: any;

  beforeEach(() => {
    createObjectUrlSpy = vi
      .spyOn(URL, "createObjectURL")
      .mockReturnValue("blob:test-preview-url");
  });

  afterEach(() => {
    createObjectUrlSpy.mockRestore();
  });

  it("AC-3.b: download control is aria-disabled when url is null", () => {
    render(
      <ReportPdfPreview
        url={null}
        isRendering={false}
        fileName="reporte.pdf"
      />,
    );

    // When url is null, the button renders as a <span> (not <a>), aria-disabled=true
    const btn = screen.getByRole("button");
    expect(btn).toHaveAttribute("aria-disabled", "true");
    // No download anchor present
    expect(
      screen.queryByRole("link", { name: /Descargar/i }),
    ).not.toBeInTheDocument();
  });

  it("AC-3.b: download control is interactive (not aria-disabled) when url is set", () => {
    const testUrl = "blob:test-preview-url";

    render(
      <ReportPdfPreview
        url={testUrl}
        isRendering={false}
        fileName="reporte.pdf"
      />,
    );

    // When ready, Button renders asChild with an <a> tag
    const link = screen.getByRole("link", { name: /Descargar/i });
    expect(link).toBeInTheDocument();
    // aria-disabled should be false/absent
    expect(link).not.toHaveAttribute("aria-disabled", "true");
  });

  it("AC-7.c: href equals the url passed as prop (no second createObjectURL call)", () => {
    const testUrl = "blob:test-preview-url";

    render(
      <ReportPdfPreview
        url={testUrl}
        isRendering={false}
        fileName="reporte.pdf"
      />,
    );

    const link = screen.getByRole("link", { name: /Descargar/i });
    expect(link).toHaveAttribute("href", testUrl);
    // This component must NOT call createObjectURL — the url is passed in directly.
    expect(createObjectUrlSpy).not.toHaveBeenCalled();
  });

  it("iframe uses about:blank when url is null (not empty src)", () => {
    render(<ReportPdfPreview url={null} isRendering={false} />);

    const iframe = screen.getByTitle("Vista previa del reporte");
    expect(iframe).toHaveAttribute("src", "about:blank");
  });

  it("iframe src equals the url when provided", () => {
    const testUrl = "blob:test-preview-url";

    render(<ReportPdfPreview url={testUrl} isRendering={false} />);

    const iframe = screen.getByTitle("Vista previa del reporte");
    expect(iframe).toHaveAttribute("src", testUrl);
  });

  it("AC-5.a: generatedAt displayed as human-readable when provided", () => {
    render(
      <ReportPdfPreview
        url="blob:test-preview-url"
        isRendering={false}
        generatedAt="2026-06-17T20:15:00Z"
      />,
    );

    // The text should start with "Generado el" and contain "junio"
    expect(screen.getByText(/Generado el.*junio.*2026/)).toBeInTheDocument();
  });

  it("generatedAt not rendered when not provided", () => {
    render(
      <ReportPdfPreview url="blob:test-preview-url" isRendering={false} />,
    );

    expect(screen.queryByText(/Generado el/)).not.toBeInTheDocument();
  });

  /**
   * AC-5.b: no refresh/regenerate control exists in the preview surface.
   * The component must NOT render any button whose accessible name matches
   * the refresh/regenerate pattern — that control was explicitly excluded from
   * the report spec (it would imply server-side re-generation, which is out of
   * scope; the PDF is client-generated and the user can re-submit the form).
   */
  it("AC-5.b: no refresh/regenerate button is present", () => {
    render(
      <ReportPdfPreview url="blob:test-preview-url" isRendering={false} />,
    );

    expect(
      screen.queryByRole("button", {
        name: /actualizar|regenerar|refresh|recargar/i,
      }),
    ).toBeNull();
  });
});
