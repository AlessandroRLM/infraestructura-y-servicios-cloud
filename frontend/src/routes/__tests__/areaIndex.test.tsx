import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { SessionState } from "@/features/auth";
import { renderWithProviders } from "@/test";

// Full admin session: holds catalog.manage → first accessible admin nav = /admin/academics
const adminSession: SessionState = {
  status: "authenticated",
  userId: "1",
  email: "admin@test.com",
  roles: ["admin"],
  permissions: [
    "catalog.manage",
    "users.manage",
    "enrollment.manage",
    "grades.read",
    "grades.write",
    "reports.read",
  ],
};

// Teacher session: grades.write → admin-eligible only → first accessible admin nav = /admin/grades
// (no catalog.manage/enrollment.manage so academics/enrollments/section-enrollments are hidden)
const teacherSession: SessionState = {
  status: "authenticated",
  userId: "2",
  email: "teacher@test.com",
  roles: ["teacher"],
  permissions: [
    "grades.read",
    "grades.write",
    "reports.read",
    "section.view_teaching",
    "section_enrollment.view_teaching",
    "profile.view_own",
    "profile.edit_own",
    "profile.view_names",
  ],
};

// Session with zero admin-nav permissions → forbidden
const noAdminNavSession: SessionState = {
  status: "authenticated",
  userId: "3",
  email: "nobody@test.com",
  roles: [],
  permissions: ["profile.view_own"],
};

// Student session: grades.view_own → first accessible participant nav = /app/grades
const studentSession: SessionState = {
  status: "authenticated",
  userId: "4",
  email: "student@test.com",
  roles: ["student"],
  permissions: [
    "grades.view_own",
    "section_enrollment.view_own",
    "sections.enroll",
  ],
};

describe("/admin area index redirect", () => {
  it("admin session lands on /admin/academics (first accessible nav entry)", async () => {
    const { router } = renderWithProviders({
      route: "/admin",
      session: adminSession,
    });

    await waitFor(() =>
      expect(router.state.location.pathname).toBe("/admin/academics"),
    );
  });

  it("teacher session lands on /admin/grades (first accessible nav entry for teacher)", async () => {
    const { router } = renderWithProviders({
      route: "/admin",
      session: teacherSession,
    });

    await waitFor(() =>
      expect(router.state.location.pathname).toBe("/admin/grades"),
    );
  });

  it("session with no accessible admin nav entries lands on /forbidden", async () => {
    const { router } = renderWithProviders({
      route: "/admin",
      session: noAdminNavSession,
    });

    await waitFor(() =>
      expect(router.state.location.pathname).toBe("/forbidden"),
    );
  });
});

describe("/app area index redirect", () => {
  it("student session lands on /app/grades (first accessible participant nav entry)", async () => {
    const { router } = renderWithProviders({
      route: "/app",
      session: studentSession,
    });

    await waitFor(() =>
      expect(router.state.location.pathname).toBe("/app/grades"),
    );
  });
});
