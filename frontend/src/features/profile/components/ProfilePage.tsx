import { create } from "@bufbuild/protobuf";
import { Info } from "lucide-react";
import { toast } from "sonner";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import type { UserProfile } from "@/gen/profiles/v1/profiles_pb";
import { UpsertOwnProfileRequestSchema } from "@/gen/profiles/v1/profiles_pb";
import { useOwnProfile } from "../hooks/useOwnProfile";
import { useUpsertOwnProfile } from "../hooks/useUpsertOwnProfile";
import type { ProfileFormValues } from "../schemas/profile";
import { mapProfileMutationError } from "./errorMapping";
import { ProfileForm, type ProfileFormHelpers } from "./ProfileForm";
import { ProfileIdentityCard } from "./ProfileIdentityCard";

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

export function ProfilePage() {
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
      <div className="flex flex-col gap-6">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full" />
        <Skeleton className="h-96 w-full" />
      </div>
    );
  }

  if (isNotFound) {
    return (
      <div className="flex flex-col gap-6">
        <h1 className="font-semibold text-2xl tracking-tight">Mi perfil</h1>
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
      <div className="flex flex-col gap-6">
        <h1 className="font-semibold text-2xl tracking-tight">Mi perfil</h1>
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
    <div className="flex flex-col gap-6">
      <h1 className="font-semibold text-2xl tracking-tight">Mi perfil</h1>
      {profile && <ProfileIdentityCard profile={profile} />}
      <ProfileForm
        defaultValues={profile ? buildDefaultValues(profile) : undefined}
        onSubmit={handleSubmit}
      />
    </div>
  );
}
