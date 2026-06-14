import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { UserProfile } from "@/gen/profiles/v1/profiles_pb";

function displayOrDash(v: string | undefined): string {
  return v && v.length > 0 ? v : "—";
}

interface ProfileIdentityCardProps {
  profile: UserProfile;
}

export function ProfileIdentityCard({ profile }: ProfileIdentityCardProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Datos de identidad</CardTitle>
      </CardHeader>
      <CardContent>
        <dl className="flex flex-col gap-3">
          <div className="flex flex-col gap-0.5">
            <dt className="text-muted-foreground text-xs">Nombres</dt>
            <dd className="text-sm font-medium">
              {displayOrDash(profile.givenNames)}
            </dd>
          </div>
          <div className="flex flex-col gap-0.5">
            <dt className="text-muted-foreground text-xs">Apellido paterno</dt>
            <dd className="text-sm font-medium">
              {displayOrDash(profile.lastNamePaternal)}
            </dd>
          </div>
          <div className="flex flex-col gap-0.5">
            <dt className="text-muted-foreground text-xs">Apellido materno</dt>
            <dd className="text-sm font-medium">
              {displayOrDash(profile.lastNameMaternal)}
            </dd>
          </div>
          <div className="flex flex-col gap-0.5">
            <dt className="text-muted-foreground text-xs">Tipo de documento</dt>
            <dd className="text-sm font-medium">
              {displayOrDash(profile.nationalIdType)}
            </dd>
          </div>
          <div className="flex flex-col gap-0.5">
            <dt className="text-muted-foreground text-xs">
              Número de documento
            </dt>
            <dd className="text-sm font-medium">
              {displayOrDash(profile.nationalId)}
            </dd>
          </div>
          <div className="flex flex-col gap-0.5">
            <dt className="text-muted-foreground text-xs">Sexo</dt>
            <dd className="text-sm font-medium">
              {displayOrDash(profile.sex)}
            </dd>
          </div>
          <div className="flex flex-col gap-0.5">
            <dt className="text-muted-foreground text-xs">Nacionalidad</dt>
            <dd className="text-sm font-medium">
              {displayOrDash(profile.nationality)}
            </dd>
          </div>
        </dl>
      </CardContent>
    </Card>
  );
}
