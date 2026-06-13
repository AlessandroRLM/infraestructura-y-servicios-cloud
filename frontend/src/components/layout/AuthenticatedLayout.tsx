import { Outlet } from "@tanstack/react-router";
import { AppSidebar } from "./AppSidebar";

export function AuthenticatedLayout() {
  return (
    <div className="flex h-svh">
      <AppSidebar />
      <main className="flex-1 overflow-y-auto p-8">
        <Outlet />
      </main>
    </div>
  );
}
