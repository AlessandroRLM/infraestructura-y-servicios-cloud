/**
 * ReportStateBoundary state machine tests — AC-4.a, AC-4.b, AC-4.c, AC-4.d, AC-4.e
 * RF-6.4: structured ConnectError matching (no substring matching on raw codes).
 */
import { Code, ConnectError } from "@connectrpc/connect";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { ReportStateBoundary } from "../components/ReportStateBoundary";

// biome-ignore lint/suspicious/noEmptyBlockStatements: intentional no-op for tests
const noOp = () => {};

const defaultProps = {
  filterSet: true,
  isFetching: false,
  isRendering: false,
  isError: false,
  error: null,
  isEmpty: false,
  truncated: false as const,
  onRetry: noOp,
};

describe("ReportStateBoundary", () => {
  it("AC-4.a: no filter set → shows prompt, no children", () => {
    render(
      <ReportStateBoundary {...defaultProps} filterSet={false}>
        <div>content</div>
      </ReportStateBoundary>,
    );

    expect(
      screen.getByText("Selecciona un filtro para generar el reporte."),
    ).toBeInTheDocument();
    expect(screen.queryByText("content")).not.toBeInTheDocument();
  });

  it("AC-4.b: isRendering true → spinner + 'Generando PDF' text", () => {
    render(
      <ReportStateBoundary {...defaultProps} isRendering>
        <div>content</div>
      </ReportStateBoundary>,
    );

    const status = screen.getByRole("status");
    expect(status).toBeInTheDocument();
    expect(screen.getByText(/Generando PDF/)).toBeInTheDocument();
    expect(screen.queryByText("content")).not.toBeInTheDocument();
  });

  it("AC-4.b: isFetching true → spinner + 'Generando PDF' text", () => {
    render(
      <ReportStateBoundary {...defaultProps} isFetching>
        <div>content</div>
      </ReportStateBoundary>,
    );

    expect(screen.getByRole("status")).toBeInTheDocument();
    expect(screen.getByText(/Generando PDF/)).toBeInTheDocument();
  });

  it("AC-4.c: isEmpty → no iframe, no download button, shows empty message", () => {
    render(
      <ReportStateBoundary {...defaultProps} isEmpty>
        <iframe title="preview" />
        <a href="/pdf" download>
          Descargar PDF
        </a>
      </ReportStateBoundary>,
    );

    expect(
      screen.queryByRole("link", { name: /Descargar/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByText("No hay datos disponibles para este reporte."),
    ).toBeInTheDocument();
  });

  it("AC-4.d: isError → 'Reintentar' button + human-readable message", async () => {
    const onRetry = vi.fn();
    const user = userEvent.setup();

    render(
      <ReportStateBoundary
        {...defaultProps}
        isError
        error={new Error("network error")}
        onRetry={onRetry}
      >
        <div>content</div>
      </ReportStateBoundary>,
    );

    // Human-readable error message visible
    expect(screen.getByRole("alert")).toBeInTheDocument();

    // Reintentar button visible and interactive
    const retryBtn = screen.getByRole("button", { name: /Reintentar/i });
    expect(retryBtn).toBeInTheDocument();

    await user.click(retryBtn);
    expect(onRetry).toHaveBeenCalledOnce();

    // Children not rendered in error state
    expect(screen.queryByText("content")).not.toBeInTheDocument();
  });

  it("error message does not expose raw gRPC code for permission_denied", () => {
    // Uses a real ConnectError so the structured matcher fires correctly.
    const permError = new ConnectError(
      "permission denied",
      Code.PermissionDenied,
    );
    render(
      <ReportStateBoundary
        {...defaultProps}
        isError
        error={permError}
        onRetry={noOp}
      >
        <div>content</div>
      </ReportStateBoundary>,
    );

    // Should NOT contain raw error code
    expect(screen.queryByText(/PERMISSION_DENIED/)).not.toBeInTheDocument();
    // Should contain safe user-facing message
    expect(
      screen.getByText("No tienes permiso para ver este reporte."),
    ).toBeInTheDocument();
  });

  it("AC-4.e: truncated=true → TruncationBanner shown above children", () => {
    render(
      <ReportStateBoundary {...defaultProps} truncated truncatedTo={500}>
        <div>pdf preview</div>
      </ReportStateBoundary>,
    );

    expect(
      screen.getByText(/Este reporte muestra solo las primeras 500 filas/),
    ).toBeInTheDocument();
    expect(screen.getByText("pdf preview")).toBeInTheDocument();
  });

  it("ready state with no truncation → renders children only", () => {
    render(
      <ReportStateBoundary {...defaultProps}>
        <div>pdf content</div>
      </ReportStateBoundary>,
    );

    expect(screen.getByText("pdf content")).toBeInTheDocument();
    expect(screen.queryByText(/primeras/)).not.toBeInTheDocument();
  });

  it("loading state takes priority over error", () => {
    render(
      <ReportStateBoundary
        {...defaultProps}
        isFetching
        isError
        error={new Error("err")}
      >
        <div>content</div>
      </ReportStateBoundary>,
    );

    // Loading shown, not error
    expect(screen.getByRole("status")).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("RF-6.4: real ConnectError Code.PermissionDenied → permission copy", () => {
    const permError = new ConnectError(
      "permission denied",
      Code.PermissionDenied,
    );

    render(
      <ReportStateBoundary
        {...defaultProps}
        isError
        error={permError}
        onRetry={noOp}
      >
        <div>content</div>
      </ReportStateBoundary>,
    );

    expect(
      screen.getByText("No tienes permiso para ver este reporte."),
    ).toBeInTheDocument();
  });

  it("RF-6.4: real ConnectError Code.Unauthenticated → generic copy (not permission copy)", () => {
    const unauthError = new ConnectError(
      "unauthenticated",
      Code.Unauthenticated,
    );

    render(
      <ReportStateBoundary
        {...defaultProps}
        isError
        error={unauthError}
        onRetry={noOp}
      >
        <div>content</div>
      </ReportStateBoundary>,
    );

    // Must NOT show the permission copy
    expect(
      screen.queryByText("No tienes permiso para ver este reporte."),
    ).not.toBeInTheDocument();
    // Must show the generic copy
    expect(
      screen.getByText(
        "Ocurrió un error al generar el reporte. Inténtalo de nuevo.",
      ),
    ).toBeInTheDocument();
  });

  it("RF-6.4: non-permission ConnectError → generic copy", () => {
    const internalError = new ConnectError("internal", Code.Internal);

    render(
      <ReportStateBoundary
        {...defaultProps}
        isError
        error={internalError}
        onRetry={noOp}
      >
        <div>content</div>
      </ReportStateBoundary>,
    );

    expect(
      screen.queryByText("No tienes permiso para ver este reporte."),
    ).not.toBeInTheDocument();
    expect(
      screen.getByText(
        "Ocurrió un error al generar el reporte. Inténtalo de nuevo.",
      ),
    ).toBeInTheDocument();
  });

  it("RF-4.5: truncated=true always renders TruncationBanner", () => {
    render(
      <ReportStateBoundary {...defaultProps} truncated={true} truncatedTo={100}>
        <div>children</div>
      </ReportStateBoundary>,
    );

    expect(
      screen.getByText(/Este reporte muestra solo las primeras 100 filas/),
    ).toBeInTheDocument();
    expect(screen.getByText("children")).toBeInTheDocument();
  });
});
