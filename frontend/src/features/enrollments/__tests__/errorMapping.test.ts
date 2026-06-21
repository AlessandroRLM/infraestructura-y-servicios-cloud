import { Code, ConnectError } from "@connectrpc/connect";
import { describe, expect, it } from "vitest";
import {
  mapCreateEnrollmentError,
  mapLifecycleError,
} from "../hooks/errorMapping";

describe("mapCreateEnrollmentError", () => {
  it("AlreadyExists returns the duplicate-enrollment inline message", () => {
    const err = new ConnectError("already exists", Code.AlreadyExists);
    expect(mapCreateEnrollmentError(err)).toBe(
      "Ya existe una matrícula para este estudiante, programa y año.",
    );
  });

  it("InvalidArgument returns the duplicate-enrollment inline message", () => {
    const err = new ConnectError("invalid argument", Code.InvalidArgument);
    expect(mapCreateEnrollmentError(err)).toBe(
      "Ya existe una matrícula para este estudiante, programa y año.",
    );
  });

  it("FailedPrecondition 'quota not found' returns the missing-quota message", () => {
    const err = new ConnectError(
      "enrollment: quota not found",
      Code.FailedPrecondition,
    );
    expect(mapCreateEnrollmentError(err)).toBe(
      "No hay cupo de matrícula definido para este programa y año.",
    );
  });

  it("FailedPrecondition 'quota full' returns the quota-full message", () => {
    const err = new ConnectError(
      "enrollment: quota full",
      Code.FailedPrecondition,
    );
    expect(mapCreateEnrollmentError(err)).toBe(
      "El cupo de matrícula para este programa y año está completo.",
    );
  });

  it("transport error returns null (caller shows a toast)", () => {
    const err = new ConnectError("unavailable", Code.Unavailable);
    expect(mapCreateEnrollmentError(err)).toBeNull();
  });

  it("non-ConnectError returns null", () => {
    expect(mapCreateEnrollmentError(new Error("network"))).toBeNull();
  });

  it("no raw codes leak — returned message is human Spanish, never a code", () => {
    const result = mapCreateEnrollmentError(
      new ConnectError("already exists", Code.AlreadyExists),
    );
    expect(result).not.toMatch(/^\d+$/);
    expect(result).not.toMatch(/failed_precondition|already_exists|\[\d/i);
  });
});

describe("mapLifecycleError", () => {
  it("FailedPrecondition returns precondition", () => {
    const err = new ConnectError(
      "failed precondition",
      Code.FailedPrecondition,
    );
    expect(mapLifecycleError(err)).toBe("precondition");
  });

  it("transport error returns transport", () => {
    const err = new ConnectError("unavailable", Code.Unavailable);
    expect(mapLifecycleError(err)).toBe("transport");
  });

  it("non-ConnectError returns transport", () => {
    expect(mapLifecycleError(new Error("network"))).toBe("transport");
  });

  it("NotFound returns transport (not precondition)", () => {
    const err = new ConnectError("not found", Code.NotFound);
    expect(mapLifecycleError(err)).toBe("transport");
  });
});
