import { Code, ConnectError } from "@connectrpc/connect";
import { useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { useLogout } from "../hooks/useLogout";

export function LogoutButton() {
  const navigate = useNavigate();
  const logout = useLogout();

  const onClick = async () => {
    try {
      await logout.mutateAsync({});
      await navigate({ to: "/login" });
    } catch (err) {
      // Cookie is already expired; the hook cleared the cache — just redirect.
      if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
        await navigate({ to: "/login" });
        return;
      }
      toast.error("Could not sign out. Please try again.");
    }
  };

  return (
    <Button
      type="button"
      variant="outline"
      onClick={onClick}
      disabled={logout.isPending}
    >
      Sign out
    </Button>
  );
}
