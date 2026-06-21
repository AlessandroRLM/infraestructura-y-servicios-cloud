import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { SectionEnrollmentStatusBadge } from "../components/SectionEnrollmentStatusBadge";

describe("SectionEnrollmentStatusBadge", () => {
  it("in_progress status shows En curso with secondary variant", () => {
    render(<SectionEnrollmentStatusBadge status="in_progress" />);
    const badge = screen.getByText("En curso");
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveAttribute("data-variant", "secondary");
  });

  it("passed status shows Aprobada with secondary variant", () => {
    render(<SectionEnrollmentStatusBadge status="passed" />);
    const badge = screen.getByText("Aprobada");
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveAttribute("data-variant", "secondary");
  });

  it("failed status shows Reprobada with destructive variant", () => {
    render(<SectionEnrollmentStatusBadge status="failed" />);
    const badge = screen.getByText("Reprobada");
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveAttribute("data-variant", "destructive");
  });

  it("withdrawn status shows Retirada with outline variant", () => {
    render(<SectionEnrollmentStatusBadge status="withdrawn" />);
    const badge = screen.getByText("Retirada");
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveAttribute("data-variant", "outline");
  });

  it("unknown status shows raw text with outline variant", () => {
    render(<SectionEnrollmentStatusBadge status="unknown-status" />);
    const badge = screen.getByText("unknown-status");
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveAttribute("data-variant", "outline");
  });
});
