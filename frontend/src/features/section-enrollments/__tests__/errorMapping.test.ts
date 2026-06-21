/**
 * Error mapping tests for section-enrollments feature.
 *
 * Covers:
 *  - FailedPrecondition → "precondition"
 *  - ResourceExhausted → "saturated"
 *  - AlreadyExists → "already_enrolled"
 *  - Everything else → "transport"
 */
import { Code, ConnectError } from "@connectrpc/connect";
import { describe, expect, it } from "vitest";
import {
  mapEnrollSectionError,
  mapWithdrawSectionError,
} from "../hooks/errorMapping";

describe("mapEnrollSectionError", () => {
  it("FailedPrecondition → precondition", () => {
    expect(
      mapEnrollSectionError(
        new ConnectError("section full", Code.FailedPrecondition),
      ),
    ).toBe("precondition");
  });

  it("ResourceExhausted → saturated", () => {
    expect(
      mapEnrollSectionError(
        new ConnectError("admission saturated", Code.ResourceExhausted),
      ),
    ).toBe("saturated");
  });

  it("AlreadyExists → already_enrolled", () => {
    expect(
      mapEnrollSectionError(
        new ConnectError("already exists", Code.AlreadyExists),
      ),
    ).toBe("already_enrolled");
  });

  it("Internal → transport", () => {
    expect(
      mapEnrollSectionError(new ConnectError("internal", Code.Internal)),
    ).toBe("transport");
  });

  it("non-ConnectError → transport", () => {
    expect(mapEnrollSectionError(new Error("network error"))).toBe("transport");
  });
});

describe("mapWithdrawSectionError", () => {
  it("FailedPrecondition → precondition", () => {
    expect(
      mapWithdrawSectionError(
        new ConnectError("not in_progress", Code.FailedPrecondition),
      ),
    ).toBe("precondition");
  });

  it("NotFound → not_found", () => {
    expect(
      mapWithdrawSectionError(new ConnectError("not found", Code.NotFound)),
    ).toBe("not_found");
  });

  it("Internal → transport", () => {
    expect(
      mapWithdrawSectionError(new ConnectError("internal", Code.Internal)),
    ).toBe("transport");
  });
});
