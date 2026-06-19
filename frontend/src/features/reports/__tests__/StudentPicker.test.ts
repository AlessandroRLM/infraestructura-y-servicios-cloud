/**
 * Unit tests for StudentPicker helpers.
 * Tests buildUserLabel pure function.
 */
import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import { UserSummarySchema } from "@/gen/iam/v1/iam_pb";
import { buildUserLabel } from "../components/StudentPicker";

function makeUser(overrides: {
  id?: string;
  email?: string;
  displayName?: string;
}) {
  return create(UserSummarySchema, {
    id: overrides.id ?? "user-uuid-1",
    email: overrides.email ?? "ana.garcia@test.com",
    displayName: overrides.displayName ?? "Ana García",
    roles: [],
    status: 0,
  });
}

describe("buildUserLabel", () => {
  it("returns 'Name — email' when displayName differs from email", () => {
    const user = makeUser({ displayName: "Ana García", email: "ana@test.com" });
    expect(buildUserLabel(user)).toBe("Ana García — ana@test.com");
  });

  it("returns email alone when displayName equals email", () => {
    const user = makeUser({
      displayName: "ana@test.com",
      email: "ana@test.com",
    });
    expect(buildUserLabel(user)).toBe("ana@test.com");
  });

  it("returns email alone when displayName is empty", () => {
    const user = makeUser({ displayName: "", email: "ana@test.com" });
    expect(buildUserLabel(user)).toBe("ana@test.com");
  });

  it("trims whitespace from displayName before comparing", () => {
    const user = makeUser({ displayName: "   ", email: "ana@test.com" });
    expect(buildUserLabel(user)).toBe("ana@test.com");
  });
});
