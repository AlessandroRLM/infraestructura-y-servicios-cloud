import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import type { AuthenticatedSession } from "@/features/auth";
import {
  GetUserResponseSchema,
  IamService,
  UserStatus,
  UserSummarySchema,
} from "@/gen/iam/v1/iam_pb";
import {
  ProfileService,
  StudentProfileSchema,
  TeacherProfileSchema,
  UserProfileSchema,
} from "@/gen/profiles/v1/profiles_pb";
import { renderWithProviders } from "@/test";

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

type IamImpl = Partial<ServiceImpl<typeof IamService>>;
type ProfileImpl = Partial<ServiceImpl<typeof ProfileService>>;

const adminSession = {
  status: "authenticated" as const,
  userId: "u-admin",
  email: "admin@test.com",
  roles: ["admin"],
  permissions: ["users.manage"],
};

const adminSessionSource = {
  getSession: async (): Promise<AuthenticatedSession> => ({
    userId: adminSession.userId,
    email: adminSession.email,
    roles: adminSession.roles,
    permissions: adminSession.permissions,
  }),
};

const baseUser = create(UserSummarySchema, {
  id: "u1",
  email: "alice@test.com",
  displayName: "Alice Smith",
  roles: ["student", "teacher"],
  status: UserStatus.ACTIVE,
});

const studentOnlyUser = create(UserSummarySchema, {
  id: "u2",
  email: "student@test.com",
  displayName: "Student User",
  roles: ["student"],
  status: UserStatus.ACTIVE,
});

const teacherOnlyUser = create(UserSummarySchema, {
  id: "u3",
  email: "teacher@test.com",
  displayName: "Teacher User",
  roles: ["teacher"],
  status: UserStatus.ACTIVE,
});

function renderSheet(iamHandlers: IamImpl, profileHandlers: ProfileImpl = {}) {
  return renderWithProviders({
    route: "/users",
    transport: makeStubTransport(
      [IamService, iamHandlers],
      [ProfileService, profileHandlers],
    ),
    session: adminSession,
    sessionSource: adminSessionSource,
  });
}

async function openSheetForUser(email: string) {
  const user = userEvent.setup();
  const row = screen.getByText(email).closest("tr");
  if (!row) throw new Error(`Row for ${email} not found`);
  await user.click(row);
  await waitFor(() => {
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });
}

describe("UserDetailSheet", () => {
  it("S-14: IAM summary shows user data after getUser resolves", async () => {
    renderSheet(
      {
        listUsers: async () => ({ users: [baseUser], nextPageToken: "" }),
        getUser: async () => create(GetUserResponseSchema, { user: baseUser }),
      },
      {
        getUserProfile: async () =>
          create(UserProfileSchema, {
            userId: "u1",
            givenNames: "Alice",
            lastNamePaternal: "Smith",
          }),
        getStudentProfile: async () =>
          create(StudentProfileSchema, { userId: "u1", admissionYear: 2022 }),
        getTeacherProfile: async () =>
          create(TeacherProfileSchema, {
            userId: "u1",
            department: "CS",
            title: "Prof",
          }),
        listTeacherQualifications: async () => ({ qualifications: [] }),
      },
    );

    await screen.findByText("alice@test.com");
    await openSheetForUser("alice@test.com");

    await screen.findByText("Detalle de usuario");
    await waitFor(() => {
      const dialog = screen.getByRole("dialog");
      expect(dialog).toHaveTextContent("alice@test.com");
    });
  });

  it("S-15: getUser returns user=undefined → not-found copy shown", async () => {
    renderSheet(
      {
        listUsers: async () => ({ users: [baseUser], nextPageToken: "" }),
        getUser: async () => create(GetUserResponseSchema, {}),
      },
      {
        getUserProfile: async () => create(UserProfileSchema, {}),
        getStudentProfile: async () => create(StudentProfileSchema, {}),
        getTeacherProfile: async () => create(TeacherProfileSchema, {}),
        listTeacherQualifications: async () => ({ qualifications: [] }),
      },
    );

    await screen.findByText("alice@test.com");
    await openSheetForUser("alice@test.com");

    await screen.findByText("El usuario no existe o fue eliminado.");
  });

  it("S-15: ConnectError NotFound → not-found copy shown", async () => {
    renderSheet(
      {
        listUsers: async () => ({ users: [baseUser], nextPageToken: "" }),
        getUser: async () => {
          throw new ConnectError("not found", Code.NotFound);
        },
      },
      {
        getUserProfile: async () => create(UserProfileSchema, {}),
        getStudentProfile: async () => create(StudentProfileSchema, {}),
        getTeacherProfile: async () => create(TeacherProfileSchema, {}),
        listTeacherQualifications: async () => ({ qualifications: [] }),
      },
    );

    await screen.findByText("alice@test.com");
    await openSheetForUser("alice@test.com");

    await screen.findByText("El usuario no existe o fue eliminado.");
  });

  it("S-16: getUserProfile error — profile section error, IAM summary unaffected", async () => {
    renderSheet(
      {
        listUsers: async () => ({ users: [baseUser], nextPageToken: "" }),
        getUser: async () => create(GetUserResponseSchema, { user: baseUser }),
      },
      {
        getUserProfile: async () => {
          throw new ConnectError("not found", Code.NotFound);
        },
        getStudentProfile: async () =>
          create(StudentProfileSchema, { userId: "u1", admissionYear: 2022 }),
        getTeacherProfile: async () =>
          create(TeacherProfileSchema, { userId: "u1" }),
        listTeacherQualifications: async () => ({ qualifications: [] }),
      },
    );

    await screen.findByText("alice@test.com");
    await openSheetForUser("alice@test.com");

    await screen.findByText("No se pudo cargar el perfil.");
    expect(
      screen.queryByText("No se pudo cargar la información del usuario."),
    ).not.toBeInTheDocument();
  });

  it("S-17a: student section shown when user has student role with admissionYear", async () => {
    renderSheet(
      {
        listUsers: async () => ({
          users: [studentOnlyUser],
          nextPageToken: "",
        }),
        getUser: async () =>
          create(GetUserResponseSchema, { user: studentOnlyUser }),
      },
      {
        getUserProfile: async () => create(UserProfileSchema, {}),
        getStudentProfile: async () =>
          create(StudentProfileSchema, { userId: "u2", admissionYear: 2021 }),
        getTeacherProfile: async () => create(TeacherProfileSchema, {}),
        listTeacherQualifications: async () => ({ qualifications: [] }),
      },
    );

    await screen.findByText("student@test.com");
    await openSheetForUser("student@test.com");

    await screen.findByText("Información académica");
    await screen.findByText("2021");
  });

  it("S-17b: student section NOT shown when user lacks student role", async () => {
    const getStudentProfile = vi.fn(async () =>
      create(StudentProfileSchema, {}),
    );

    renderSheet(
      {
        listUsers: async () => ({
          users: [teacherOnlyUser],
          nextPageToken: "",
        }),
        getUser: async () =>
          create(GetUserResponseSchema, { user: teacherOnlyUser }),
      },
      {
        getUserProfile: async () => create(UserProfileSchema, {}),
        getStudentProfile,
        getTeacherProfile: async () =>
          create(TeacherProfileSchema, { userId: "u3", department: "CS" }),
        listTeacherQualifications: async () => ({ qualifications: [] }),
      },
    );

    await screen.findByText("teacher@test.com");
    await openSheetForUser("teacher@test.com");

    await screen.findByText("Información docente");
    expect(screen.queryByText("Información académica")).not.toBeInTheDocument();
    expect(getStudentProfile).not.toHaveBeenCalled();
  });

  it("S-18a: teacher section shown when user has teacher role", async () => {
    renderSheet(
      {
        listUsers: async () => ({
          users: [teacherOnlyUser],
          nextPageToken: "",
        }),
        getUser: async () =>
          create(GetUserResponseSchema, { user: teacherOnlyUser }),
      },
      {
        getUserProfile: async () => create(UserProfileSchema, {}),
        getStudentProfile: async () => create(StudentProfileSchema, {}),
        getTeacherProfile: async () =>
          create(TeacherProfileSchema, {
            userId: "u3",
            department: "Engineering",
            title: "Professor",
          }),
        listTeacherQualifications: async () => ({
          qualifications: [
            { id: "q1", teacherId: "u3", degree: "PhD", year: 2010 },
          ],
        }),
      },
    );

    await screen.findByText("teacher@test.com");
    await openSheetForUser("teacher@test.com");

    await screen.findByText("Información docente");
    await screen.findByText("Engineering");
  });

  it("S-18b: teacher section NOT shown when user lacks teacher role", async () => {
    const getTeacherProfile = vi.fn(async () =>
      create(TeacherProfileSchema, {}),
    );

    renderSheet(
      {
        listUsers: async () => ({
          users: [studentOnlyUser],
          nextPageToken: "",
        }),
        getUser: async () =>
          create(GetUserResponseSchema, { user: studentOnlyUser }),
      },
      {
        getUserProfile: async () => create(UserProfileSchema, {}),
        getStudentProfile: async () =>
          create(StudentProfileSchema, { userId: "u2", admissionYear: 2021 }),
        getTeacherProfile,
        listTeacherQualifications: async () => ({ qualifications: [] }),
      },
    );

    await screen.findByText("student@test.com");
    await openSheetForUser("student@test.com");

    await screen.findByText("Información académica");
    expect(screen.queryByText("Información docente")).not.toBeInTheDocument();
    expect(getTeacherProfile).not.toHaveBeenCalled();
  });

  it("S-19: student profile errors, teacher loads normally — isolation", async () => {
    renderSheet(
      {
        listUsers: async () => ({ users: [baseUser], nextPageToken: "" }),
        getUser: async () => create(GetUserResponseSchema, { user: baseUser }),
      },
      {
        getUserProfile: async () => create(UserProfileSchema, {}),
        getStudentProfile: async () => {
          throw new ConnectError("error", Code.Internal);
        },
        getTeacherProfile: async () =>
          create(TeacherProfileSchema, { userId: "u1", department: "CS" }),
        listTeacherQualifications: async () => ({ qualifications: [] }),
      },
    );

    await screen.findByText("alice@test.com");
    await openSheetForUser("alice@test.com");

    await screen.findByText("No se pudo cargar la información académica.");
    await screen.findByText("Información docente");
    await screen.findByText("CS");
  });
});
