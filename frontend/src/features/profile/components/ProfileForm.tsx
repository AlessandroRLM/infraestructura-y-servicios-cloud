import { zodResolver } from "@hookform/resolvers/zod";
import { CalendarIcon, LoaderCircle, Save } from "lucide-react";
import { Controller, type UseFormSetError, useForm } from "react-hook-form";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/core/utils/cn";
import { type ProfileFormValues, profileSchema } from "../schemas/profile";

// Formats a Date to YYYY-MM-DD using LOCAL year/month/day parts.
// NEVER use toISOString() — UTC offset can shift the day.
function toYMD(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

// Parses a YYYY-MM-DD string to a LOCAL Date.
// NEVER use new Date("YYYY-MM-DD") — that parses as UTC midnight.
function parseLocalDate(s: string): Date | undefined {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(s);
  if (!match) return undefined;
  return new Date(Number(match[1]), Number(match[2]) - 1, Number(match[3]));
}

export interface ProfileFormHelpers {
  setError: UseFormSetError<ProfileFormValues>;
}

interface ProfileFormProps {
  onSubmit: (
    values: ProfileFormValues,
    helpers: ProfileFormHelpers,
  ) => Promise<void>;
  defaultValues?: Partial<ProfileFormValues>;
  submitLabel?: string;
}

export function ProfileForm({
  onSubmit,
  defaultValues,
  submitLabel = "Guardar",
}: ProfileFormProps) {
  const {
    register,
    control,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<ProfileFormValues>({
    resolver: zodResolver(profileSchema),
    mode: "onBlur",
    defaultValues: {
      birthDate: "",
      phone: "",
      personalEmail: "",
      addressStreet: "",
      commune: "",
      region: "",
      country: "",
      postalCode: "",
      emergencyContactName: "",
      emergencyContactPhone: "",
      ...defaultValues,
    },
  });

  return (
    <form
      noValidate
      onSubmit={handleSubmit((values) => onSubmit(values, { setError }))}
      className="flex flex-col gap-6"
    >
      {/* Personal */}
      <fieldset className="flex flex-col gap-4">
        <legend className="text-sm font-semibold text-foreground">
          Personal
        </legend>

        <div className="flex flex-col gap-2">
          <Label htmlFor="profile-birthDate">Fecha de nacimiento</Label>
          <Controller
            name="birthDate"
            control={control}
            render={({ field }) => (
              <Popover>
                <PopoverTrigger asChild>
                  <Button
                    id="profile-birthDate"
                    type="button"
                    variant="outline"
                    aria-invalid={!!errors.birthDate}
                    className={cn(
                      "w-full justify-start text-left font-normal",
                      !field.value && "text-muted-foreground",
                    )}
                  >
                    <CalendarIcon data-icon="inline-start" />
                    {field.value ? field.value : "Selecciona una fecha"}
                  </Button>
                </PopoverTrigger>
                <PopoverContent className="w-auto p-0" align="start">
                  <Calendar
                    mode="single"
                    selected={
                      field.value ? parseLocalDate(field.value) : undefined
                    }
                    onSelect={(d) => field.onChange(d ? toYMD(d) : "")}
                  />
                </PopoverContent>
              </Popover>
            )}
          />
          <div className="min-h-[1.25rem]">
            {errors.birthDate && (
              <p role="alert" className="text-destructive text-sm">
                {errors.birthDate.message}
              </p>
            )}
          </div>
        </div>
      </fieldset>

      {/* Contact */}
      <fieldset className="flex flex-col gap-4">
        <legend className="text-sm font-semibold text-foreground">
          Contacto
        </legend>

        <div className="flex flex-col gap-2">
          <Label htmlFor="profile-phone">Teléfono</Label>
          <Input
            id="profile-phone"
            placeholder="ej. +56912345678"
            aria-invalid={!!errors.phone}
            {...register("phone")}
          />
          <div className="min-h-[1.25rem]">
            {errors.phone && (
              <p role="alert" className="text-destructive text-sm">
                {errors.phone.message}
              </p>
            )}
          </div>
        </div>

        <div className="flex flex-col gap-2">
          <Label htmlFor="profile-personalEmail">Correo personal</Label>
          <Input
            id="profile-personalEmail"
            type="email"
            placeholder="ej. nombre@correo.com"
            aria-invalid={!!errors.personalEmail}
            {...register("personalEmail")}
          />
          <div className="min-h-[1.25rem]">
            {errors.personalEmail && (
              <p role="alert" className="text-destructive text-sm">
                {errors.personalEmail.message}
              </p>
            )}
          </div>
        </div>
      </fieldset>

      {/* Address */}
      <fieldset className="flex flex-col gap-4">
        <legend className="text-sm font-semibold text-foreground">
          Dirección
        </legend>

        <div className="flex flex-col gap-2">
          <Label htmlFor="profile-addressStreet">Calle y número</Label>
          <Input
            id="profile-addressStreet"
            placeholder="ej. Av. Libertador 1234"
            aria-invalid={!!errors.addressStreet}
            {...register("addressStreet")}
          />
          <div className="min-h-[1.25rem]">
            {errors.addressStreet && (
              <p role="alert" className="text-destructive text-sm">
                {errors.addressStreet.message}
              </p>
            )}
          </div>
        </div>

        <div className="flex flex-col gap-2">
          <Label htmlFor="profile-commune">Comuna</Label>
          <Input
            id="profile-commune"
            placeholder="ej. Santiago"
            aria-invalid={!!errors.commune}
            {...register("commune")}
          />
          <div className="min-h-[1.25rem]">
            {errors.commune && (
              <p role="alert" className="text-destructive text-sm">
                {errors.commune.message}
              </p>
            )}
          </div>
        </div>

        <div className="flex flex-col gap-2">
          <Label htmlFor="profile-region">Región</Label>
          <Input
            id="profile-region"
            placeholder="ej. Región Metropolitana"
            aria-invalid={!!errors.region}
            {...register("region")}
          />
          <div className="min-h-[1.25rem]">
            {errors.region && (
              <p role="alert" className="text-destructive text-sm">
                {errors.region.message}
              </p>
            )}
          </div>
        </div>

        <div className="flex flex-col gap-2">
          <Label htmlFor="profile-country">País</Label>
          <Input
            id="profile-country"
            placeholder="ej. Chile"
            aria-invalid={!!errors.country}
            {...register("country")}
          />
          <div className="min-h-[1.25rem]">
            {errors.country && (
              <p role="alert" className="text-destructive text-sm">
                {errors.country.message}
              </p>
            )}
          </div>
        </div>

        <div className="flex flex-col gap-2">
          <Label htmlFor="profile-postalCode">Código postal</Label>
          <Input
            id="profile-postalCode"
            placeholder="ej. 8320000"
            aria-invalid={!!errors.postalCode}
            {...register("postalCode")}
          />
          <div className="min-h-[1.25rem]">
            {errors.postalCode && (
              <p role="alert" className="text-destructive text-sm">
                {errors.postalCode.message}
              </p>
            )}
          </div>
        </div>
      </fieldset>

      {/* Emergency */}
      <fieldset className="flex flex-col gap-4">
        <legend className="text-sm font-semibold text-foreground">
          Contacto de emergencia
        </legend>

        <div className="flex flex-col gap-2">
          <Label htmlFor="profile-emergencyContactName">Nombre</Label>
          <Input
            id="profile-emergencyContactName"
            placeholder="ej. María González"
            aria-invalid={!!errors.emergencyContactName}
            {...register("emergencyContactName")}
          />
          <div className="min-h-[1.25rem]">
            {errors.emergencyContactName && (
              <p role="alert" className="text-destructive text-sm">
                {errors.emergencyContactName.message}
              </p>
            )}
          </div>
        </div>

        <div className="flex flex-col gap-2">
          <Label htmlFor="profile-emergencyContactPhone">Teléfono</Label>
          <Input
            id="profile-emergencyContactPhone"
            placeholder="ej. +56987654321"
            aria-invalid={!!errors.emergencyContactPhone}
            {...register("emergencyContactPhone")}
          />
          <div className="min-h-[1.25rem]">
            {errors.emergencyContactPhone && (
              <p role="alert" className="text-destructive text-sm">
                {errors.emergencyContactPhone.message}
              </p>
            )}
          </div>
        </div>
      </fieldset>

      <Button type="submit" className="w-full gap-2" disabled={isSubmitting}>
        {isSubmitting ? (
          <>
            <LoaderCircle className="size-4 animate-spin" aria-hidden />
            Guardando…
          </>
        ) : (
          <>
            <Save data-icon="inline-start" aria-hidden />
            {submitLabel}
          </>
        )}
      </Button>
    </form>
  );
}
