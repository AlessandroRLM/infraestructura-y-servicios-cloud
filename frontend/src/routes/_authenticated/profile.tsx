import { createFileRoute } from "@tanstack/react-router";
import { SharedAreaLayout } from "@/components/layout/SharedAreaLayout";
import { ProfilePage } from "@/features/profile";

function ProfileRoute() {
  return (
    <SharedAreaLayout>
      <ProfilePage />
    </SharedAreaLayout>
  );
}

export const Route = createFileRoute("/_authenticated/profile")({
  component: ProfileRoute,
});
