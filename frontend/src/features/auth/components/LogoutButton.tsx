import { Code, ConnectError } from "@connectrpc/connect";
import { useNavigate } from "@tanstack/react-router";
import { LogOut } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { cn } from "@/core/utils/cn";
import { useLogout } from "../hooks/useLogout";

export function LogoutButton({ className }: { className?: string }) {
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
      toast.error("No se pudo cerrar sesión. Inténtalo de nuevo.");
    }
  };

  return (
    <Button
      type="button"
      variant="outline"
      onClick={onClick}
      disabled={logout.isPending}
      className={cn("gap-2", className)}
    >
      <LogOut className="size-4" aria-hidden />
      Cerrar sesión
    </Button>
  );
}
