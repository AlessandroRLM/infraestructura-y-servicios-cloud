import { LoaderCircle, X } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ROLE_LABELS, ROLES, type Role, useSession } from "@/features/auth";
import { useAssignRole } from "../hooks/useAssignRole";
import { useRevokeRole } from "../hooks/useRevokeRole";
import { mapRevokeError } from "./roleErrors";

interface RoleManagementProps {
  userId: string;
  roles: string[];
}

export function RoleManagement({ userId, roles }: RoleManagementProps) {
  const session = useSession();
  const assign = useAssignRole(userId);
  const revoke = useRevokeRole(userId);

  // Role whose X was clicked — controls AlertDialog visibility.
  const [pendingRevoke, setPendingRevoke] = useState<Role | null>(null);
  // Inline error shown inside the dialog while it stays open (last-admin case).
  const [revokeError, setRevokeError] = useState<
    "last-admin" | "transport" | null
  >(null);

  const isSelf =
    session.status === "authenticated" && session.userId === userId;
  // Hide the revoke-X only for the admin role on self (R-14).
  const canRevoke = (role: Role) => !(isSelf && role === "admin");

  // Filter ROLES (already priority-ordered: admin→teacher→student).
  const heldRoles = ROLES.filter((r) => roles.includes(r));
  const availableRoles = ROLES.filter((r) => !roles.includes(r));

  const isPending = assign.isPending || revoke.isPending;

  async function handleAssign(role: Role) {
    try {
      await assign.mutateAsync({ userId, role });
      toast.success("Rol asignado");
    } catch {
      toast.error("No se pudo completar la acción. Inténtalo de nuevo.");
    }
  }

  async function handleRevokeConfirm(e: React.MouseEvent) {
    // Prevent AlertDialog from closing automatically on action click (R-11).
    e.preventDefault();
    setRevokeError(null);
    try {
      await revoke.mutateAsync({ userId, role: pendingRevoke! });
      toast.success("Rol revocado");
      setPendingRevoke(null);
    } catch (err) {
      const kind = mapRevokeError(err);
      if (kind === "last-admin") {
        // Keep dialog open, show inline error (R-11, S-RM-10).
        setRevokeError("last-admin");
      } else {
        // Transport error: close dialog, show toast (R-12, S-RM-11).
        setPendingRevoke(null);
        toast.error("No se pudo completar la acción. Inténtalo de nuevo.");
      }
    }
  }

  return (
    <div className="flex flex-col gap-3">
      {heldRoles.length === 0 ? (
        <p className="text-muted-foreground text-sm">Sin roles asignados</p>
      ) : (
        <div className="flex flex-wrap gap-2">
          {heldRoles.map((role) => (
            <Badge
              key={role}
              variant="secondary"
              className="flex items-center gap-1"
            >
              {ROLE_LABELS[role]}
              {canRevoke(role) && (
                <button
                  type="button"
                  aria-label={`Quitar rol ${ROLE_LABELS[role]}`}
                  disabled={isPending}
                  onClick={() => {
                    setPendingRevoke(role);
                    setRevokeError(null);
                  }}
                  className="ml-0.5 rounded-full p-0.5 hover:bg-muted-foreground/20 disabled:pointer-events-none disabled:opacity-50"
                >
                  <X className="size-3" aria-hidden />
                </button>
              )}
            </Badge>
          ))}
        </div>
      )}

      {availableRoles.length > 0 && (
        <Select
          disabled={isPending}
          onValueChange={(v) => handleAssign(v as Role)}
        >
          <SelectTrigger>
            <SelectValue placeholder="Agregar rol" />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              {availableRoles.map((role) => (
                <SelectItem key={role} value={role}>
                  {ROLE_LABELS[role]}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
      )}

      <AlertDialog
        open={pendingRevoke !== null}
        onOpenChange={(open) => {
          if (!open) {
            setPendingRevoke(null);
            setRevokeError(null);
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>¿Quitar rol?</AlertDialogTitle>
            <AlertDialogDescription>
              {pendingRevoke &&
                `¿Quitar el rol ${ROLE_LABELS[pendingRevoke]} a este usuario? Podrás volver a asignarlo.`}
            </AlertDialogDescription>
          </AlertDialogHeader>

          {revokeError === "last-admin" && (
            <p role="alert" className="text-destructive text-sm">
              No se puede quitar el último administrador del sistema.
            </p>
          )}

          <AlertDialogFooter>
            <AlertDialogCancel
              onClick={() => {
                setPendingRevoke(null);
                setRevokeError(null);
              }}
            >
              Cancelar
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={handleRevokeConfirm}
              disabled={revoke.isPending}
              className="gap-2"
            >
              {revoke.isPending ? (
                <>
                  <LoaderCircle className="size-4 animate-spin" aria-hidden />
                  Quitando…
                </>
              ) : (
                "Quitar"
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
