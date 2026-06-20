import { Code, ConnectError } from "@connectrpc/connect";
import { describe, expect, it, vi } from "vitest";
import {
  mapCreateEnrollmentError,
  mapLifecycleError,
} from "../hooks/errorMapping";

describe("mapCreateEnrollmentError", () => {
  it("AlreadyExists with setError returns handled-inline and sets field message", () => {
    const setError = vi.fn();
    const err = new ConnectError("already exists", Code.AlreadyExists);
    const result = mapCreateEnrollmentError(err, setError);
    expect(result).toBe("handled-inline");
    expect(setError).toHaveBeenCalledWith(
      "root",
      expect.objectContaining({
        message: "Ya existe una matrícula para este estudiante, programa y año.",
      }),
    );
  });

  it("InvalidArgument with setError returns handled-inline and sets field message", () => {
    const setError = vi.fn();
    const err = new ConnectError("invalid argument", Code.InvalidArgument);
    const result = mapCreateEnrollmentError(err, setError);
    expect(result).toBe("handled-inline");
    expect(setError).toHaveBeenCalledWith(
      "root",
      expect.objectContaining({
        message: "Ya existe una matrícula para este estudiante, programa y año.",
      }),
    );
  });

  it("transport error returns toast", () => {
    const setError = vi.fn();
    const err = new ConnectError("unavailable", Code.Unavailable);
    const result = mapCreateEnrollmentError(err, setError);
    expect(result).toBe("toast");
    expect(setError).not.toHaveBeenCalled();
  });

  it("non-ConnectError returns toast without calling setError", () => {
    const setError = vi.fn();
    const result = mapCreateEnrollmentError(new Error("network"), setError);
    expect(result).toBe("toast");
    expect(setError).not.toHaveBeenCalled();
  });

  it("AlreadyExists without setError returns handled-inline", () => {
    const err = new ConnectError("already exists", Code.AlreadyExists);
    const result = mapCreateEnrollmentError(err);
    expect(result).toBe("handled-inline");
  });

  it("no raw codes leak — returned string is not a numeric code", () => {
    const err = new ConnectError("already exists", Code.AlreadyExists);
    const result = mapCreateEnrollmentError(err);
    expect(typeof result).toBe("string");
    expect(result).not.toMatch(/^\d+$/);
  });
});

describe("mapLifecycleError", () => {
  it("FailedPrecondition returns precondition", () => {
    const err = new ConnectError("failed precondition", Code.FailedPrecondition);
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
