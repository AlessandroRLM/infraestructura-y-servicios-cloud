import { Badge } from "@/components/ui/badge";
import { UserStatus } from "@/gen/iam/v1/iam_pb";

export function UserStatusBadge({ status }: { status: UserStatus }) {
  if (status === UserStatus.DISABLED) {
    return <Badge variant="outline">Deshabilitado</Badge>;
  }
  return <Badge variant="secondary">Activo</Badge>;
}
