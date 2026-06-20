import { create } from "@bufbuild/protobuf";
import { Link, linkOptions } from "@tanstack/react-router";
import { ArrowLeft, Info } from "lucide-react";
import { toast } from "sonner";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  eligibleAreas,
  readPreferredArea,
  useSession,
} from "@/features/auth";
import type { Area } from "@/features/auth";
import { UpsertOwnProfileRequestSchema } from "@/gen/profiles/v1/profiles_pb";
import type { UserProfile } from "@/gen/profiles/v1/profiles_pb";
import { useOwnProfile } from "../hooks/useOwnProfile";
import { useUpsertOwnProfile } from "../hooks/useUpsertOwnProfile";
import type { ProfileFormValues } from "../schemas/profile";
import { mapProfileMutationError } from "./errorMapping";
import { ProfileForm, type ProfileFormHelpers } from "./ProfileForm";
import { ProfileIdentityCard } from "./ProfileIdentityCard";

/** Resolves the home path for the active area to use as the back-nav target. */
function resolveAreaHome(areas: Area[], preferred: Area | null): "/admin" | "/app" {
  const active =
    preferred !== null && areas.includes(preferred)
      ? preferred
      : (areas[0] ?? "participant");
  return active === "admin" ? "/admin" : "/app";
}

function buildDefaultValues(profile: UserProfile): Partial<ProfileFormValues> {
  return {
    birthDate: profile.birthDate ?? "",
    phone: profile.phone ?? "",
    personalEmail: profile.personalEmail ?? "",
    addressStreet: profile.addressStreet ?? "",
    commune: profile.commune ?? "",
    region: profile.region ?? "",
    country: profile.country ?? "",
    postalCode: profile.postalCode ?? "",
    emergencyContactName: profile.emergencyContactName ?? "",
    emergencyContactPhone: profile.emergencyContactPhone ?? "",
  };
}

/** Back navigation link + page heading — rendered in all states for consistent chrome. */
function ProfileHeader({ areaHome }: { areaHome: "/admin" | "/app" }) {
  return (
    <div className="flex flex-col gap-1">
      <Link
        {...linkOptions({ to: areaHome })}
        className="inline-flex items-center gap-1.5 text-muted-foreground text-sm transition-colors hover:text-foreground w-fit"
      >
        <ArrowLeft className="size-4" aria-hidden />
        Volver
      </Link>
      <div className="mt-3">
        <h1 className="font-semibold text-2xl tracking-tight">Mi perfil</h1>
        <p className="text-muted-foreground text-sm mt-1">
          Administra tu información de contacto y datos de emergencia.
        </p>
      </div>
    </div>
  );
}

export function ProfilePage() {
  const session = useSession();
  const isAuth = session.status === "authenticated";
  const areas = isAuth ? eligibleAreas(session) : [];
  const preferred = readPreferredArea();
  const areaHome = resolveAreaHome(areas, preferred);

  const { profile, isLoading, isError, isNotFound, refetch } = useOwnProfile();
  const mutation = useUpsertOwnProfile();

  const handleSubmit = async (
    values: ProfileFormValues,
    { setError }: ProfileFormHelpers,
  ) => {
    try {
      await mutation.mutateAsync(
        create(UpsertOwnProfileRequestSchema, {
          birthDate: values.birthDate,
          phone: values.phone,
          personalEmail: values.personalEmail,
          addressStreet: values.addressStreet,
          commune: values.commune,
          region: values.region,
          country: values.country,
          postalCode: values.postalCode,
          emergencyContactName: values.emergencyContactName,
          emergencyContactPhone: values.emergencyContactPhone,
        }),
      );
      toast.success("Perfil actualizado");
    } catch (err) {
      const result = mapProfileMutationError(err, setError);
      if (result === "toast") {
        toast.error("Error al guardar el perfil");
      }
    }
  };

  if (isLoading) {
    return (
      <div className="max-w-3xl mx-auto flex flex-col gap-6">
        <ProfileHeader areaHome={areaHome} />
        <Skeleton className="h-64 w-full" />
        <Skeleton className="h-96 w-full" />
      </div>
    );
  }

  if (isNotFound) {
    return (
      <div className="max-w-3xl mx-auto flex flex-col gap-6">
        <ProfileHeader areaHome={areaHome} />
        <Alert>
          <Info className="size-4" />
          <AlertTitle>Completa tu perfil</AlertTitle>
          <AlertDescription>
            Agrega tus datos de contacto y guarda para completar tu perfil.
          </AlertDescription>
        </Alert>
        <ProfileForm onSubmit={handleSubmit} />
      </div>
    );
  }

  if (isError) {
    return (
      <div className="max-w-3xl mx-auto flex flex-col gap-6">
        <ProfileHeader areaHome={areaHome} />
        <p className="text-muted-foreground text-sm">
          No se pudo cargar el perfil.
        </p>
        <Button variant="outline" onClick={() => refetch()}>
          Reintentar
        </Button>
      </div>
    );
  }

  return (
    <div className="max-w-3xl mx-auto flex flex-col gap-6">
      <ProfileHeader areaHome={areaHome} />
      {profile && <ProfileIdentityCard profile={profile} />}
      <ProfileForm
        defaultValues={profile ? buildDefaultValues(profile) : undefined}
        onSubmit={handleSubmit}
      />
    </div>
  );
}
