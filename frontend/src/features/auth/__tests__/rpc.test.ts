import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { describe, expect, it } from "vitest";
import { makeStubTransport } from "@/core/test";
import { AuthService, SessionSchema } from "@/gen/auth/v1/auth_pb";
import { createRpcSessionSource } from "../api/rpc";

describe("createRpcSessionSource", () => {
  it("maps the RPC session to the domain shape", async () => {
    const transport = makeStubTransport([
      AuthService,
      {
        getSession: async () =>
          create(SessionSchema, {
            userId: "u-1",
            email: "test@example.com",
            roles: ["teacher"],
            permissions: ["grades.write"],
          }),
      },
    ]);

    await expect(
      createRpcSessionSource(transport).getSession(),
    ).resolves.toEqual({
      userId: "u-1",
      email: "test@example.com",
      roles: ["teacher"],
      permissions: ["grades.write"],
    });
  });

  it("maps CodeUnauthenticated to null (no active session)", async () => {
    const transport = makeStubTransport([
      AuthService,
      {
        getSession: async () => {
          throw new ConnectError("no session", Code.Unauthenticated);
        },
      },
    ]);

    await expect(
      createRpcSessionSource(transport).getSession(),
    ).resolves.toBeNull();
  });

  it("rethrows errors that are not unauthenticated", async () => {
    const transport = makeStubTransport([
      AuthService,
      {
        getSession: async () => {
          throw new ConnectError("boom", Code.Internal);
        },
      },
    ]);

    await expect(
      createRpcSessionSource(transport).getSession(),
    ).rejects.toThrow("boom");
  });
});
