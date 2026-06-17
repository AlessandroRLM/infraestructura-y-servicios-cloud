import { Code, ConnectError } from "@connectrpc/connect";
import { describe, expect, it } from "vitest";
import { mapSchemeError } from "../components/errorMapping";

describe("mapSchemeError", () => {
  it("maps FailedPrecondition to 'precondition'", () => {
    const err = new ConnectError("scheme locked", Code.FailedPrecondition);
    expect(mapSchemeError(err)).toBe("precondition");
  });

  it("maps AlreadyExists to 'already-exists'", () => {
    const err = new ConnectError("scheme exists", Code.AlreadyExists);
    expect(mapSchemeError(err)).toBe("already-exists");
  });

  it("maps an unknown ConnectError code to 'generic'", () => {
    const err = new ConnectError("internal", Code.Internal);
    expect(mapSchemeError(err)).toBe("generic");
  });

  it("maps a plain Error to 'generic'", () => {
    expect(mapSchemeError(new Error("network error"))).toBe("generic");
  });

  it("maps a non-error value to 'generic'", () => {
    expect(mapSchemeError("something went wrong")).toBe("generic");
    expect(mapSchemeError(null)).toBe("generic");
    expect(mapSchemeError(undefined)).toBe("generic");
  });
});
