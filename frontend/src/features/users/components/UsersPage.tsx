import { useState } from "react";
import { hasPermission, useSession } from "@/features/auth";
import { UserDetailSheet } from "./UserDetailSheet";
import { UsersTable } from "./UsersTable";

export function UsersPage() {
  const session = useSession();

  if (!hasPermission(session, "users.manage")) {
    return null;
  }

  return <UsersPageContent />;
}

function UsersPageContent() {
  const [selectedUserId, setSelectedUserId] = useState<string | undefined>(
    undefined,
  );

  return (
    <div className="flex flex-col gap-6">
      <h1 className="font-semibold text-2xl tracking-tight">Usuarios</h1>
      <UsersTable onRowClick={setSelectedUserId} />
      <UserDetailSheet
        userId={selectedUserId}
        open={!!selectedUserId}
        onOpenChange={(open) => {
          if (!open) setSelectedUserId(undefined);
        }}
      />
    </div>
  );
}
