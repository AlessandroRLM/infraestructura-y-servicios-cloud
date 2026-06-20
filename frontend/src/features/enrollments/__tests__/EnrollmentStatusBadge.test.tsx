import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { EnrollmentStatusBadge } from "../components/EnrollmentStatusBadge";

describe("EnrollmentStatusBadge", () => {
  it("pending status shows Pendiente with outline variant", () => {
    render(<EnrollmentStatusBadge status="pending" />);
    const badge = screen.getByText("Pendiente");
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveAttribute("data-variant", "outline");
  });

  it("paid status shows Pagada with secondary variant", () => {
    render(<EnrollmentStatusBadge status="paid" />);
    const badge = screen.getByText("Pagada");
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveAttribute("data-variant", "secondary");
  });

  it("cancelled status shows Cancelada with destructive variant", () => {
    render(<EnrollmentStatusBadge status="cancelled" />);
    const badge = screen.getByText("Cancelada");
    expect(badge).toBeInTheDocument();
    // destructive variant uses data-variant attribute
    expect(badge).toHaveAttribute("data-variant", "destructive");
  });

  it("unknown status shows raw text with outline variant", () => {
    render(<EnrollmentStatusBadge status="unknown-status" />);
    const badge = screen.getByText("unknown-status");
    expect(badge).toBeInTheDocument();
    // Falls back to outline
    expect(badge).toHaveAttribute("data-variant", "outline");
  });
});
