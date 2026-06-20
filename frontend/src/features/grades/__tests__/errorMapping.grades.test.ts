import { Code, ConnectError } from "@connectrpc/connect";
import { describe, expect, it } from "vitest";
import { mapGradeWriteError } from "../components/errorMapping";

describe("mapGradeWriteError", () => {
  it("maps CodeAborted to 'conflict'", () => {
    const err = new ConnectError("stale version", Code.Aborted);
    expect(mapGradeWriteError(err)).toBe("conflict");
  });

  it("maps CodeInternal to 'generic'", () => {
    const err = new ConnectError("internal error", Code.Internal);
    expect(mapGradeWriteError(err)).toBe("generic");
  });

  it("maps CodePermissionDenied to 'generic'", () => {
    const err = new ConnectError("no permission", Code.PermissionDenied);
    expect(mapGradeWriteError(err)).toBe("generic");
  });

  it("maps a plain Error to 'generic'", () => {
    expect(mapGradeWriteError(new Error("network error"))).toBe("generic");
  });

  it("maps a non-error value to 'generic'", () => {
    expect(mapGradeWriteError("unexpected")).toBe("generic");
    expect(mapGradeWriteError(null)).toBe("generic");
    expect(mapGradeWriteError(undefined)).toBe("generic");
  });
});
