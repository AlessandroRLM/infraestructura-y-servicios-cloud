import { Code, ConnectError } from "@connectrpc/connect";
import { zodResolver } from "@hookform/resolvers/zod";
import { useNavigate } from "@tanstack/react-router";
import { GraduationCap, LoaderCircle, Lock, LogIn, Mail } from "lucide-react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useLogin } from "../hooks/useLogin";
import { type LoginValues, loginSchema } from "../schemas/credentials";

interface LoginFormProps {
  redirectTo: string;
}

export function LoginForm({ redirectTo }: LoginFormProps) {
  const navigate = useNavigate();
  const login = useLogin();
  const {
    register,
    handleSubmit,
    setError,
    clearErrors,
    formState: { errors, isSubmitting },
  } = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
    mode: "onBlur",
  });

  const onSubmit = handleSubmit(async (values) => {
    clearErrors("root");
    try {
      await login.mutateAsync(values);
      // Use href so search params and hash in redirectTo are preserved correctly.
      await navigate({ href: redirectTo });
    } catch (err) {
      if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
        // Domain error about user input — show inline, never reveal which field.
        setError("root", { message: "Correo o contraseña incorrectos" });
        return;
      }
      // Transport/infra failure — transient, non-blocking.
      toast.error("No se pudo conectar. Inténtalo de nuevo.");
    }
  });

  return (
    <div className="flex min-h-svh items-center justify-center bg-muted/40 p-4">
      <Card
        data-testid="login-page"
        className="w-full max-w-md gap-0 p-8 shadow-sm"
      >
        <CardHeader className="flex flex-row items-center gap-3 p-0 text-left">
          <span className="flex size-12 items-center justify-center rounded-xl bg-primary text-primary-foreground">
            <GraduationCap className="size-6" aria-hidden />
          </span>
          <div className="space-y-1">
            <CardTitle className="text-2xl tracking-tight">
              Iniciar sesión
            </CardTitle>
            <CardDescription>
              Sistema de administración académica
            </CardDescription>
          </div>
        </CardHeader>

        <CardContent className="p-0 pt-8">
          <form noValidate onSubmit={onSubmit} className="space-y-5">
            <div className="space-y-2">
              <Label htmlFor="email">Correo electrónico</Label>
              <div className="relative">
                <Mail
                  className="-translate-y-1/2 pointer-events-none absolute top-1/2 left-3 size-5 text-muted-foreground"
                  aria-hidden
                />
                <Input
                  id="email"
                  type="email"
                  autoComplete="email"
                  placeholder="tu@universidad.edu"
                  aria-invalid={errors.email ? true : undefined}
                  className="h-12 pl-10 text-base"
                  {...register("email")}
                />
              </div>
              <div className="min-h-[1.25rem]">
                {errors.email && (
                  <p role="alert" className="text-destructive text-sm">
                    {errors.email.message}
                  </p>
                )}
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="password">Contraseña</Label>
              <div className="relative">
                <Lock
                  className="-translate-y-1/2 pointer-events-none absolute top-1/2 left-3 size-5 text-muted-foreground"
                  aria-hidden
                />
                <Input
                  id="password"
                  type="password"
                  autoComplete="current-password"
                  placeholder="Tu contraseña"
                  aria-invalid={errors.password ? true : undefined}
                  className="h-12 pl-10 text-base"
                  {...register("password")}
                />
              </div>
              <div className="min-h-[1.25rem]">
                {errors.password && (
                  <p role="alert" className="text-destructive text-sm">
                    {errors.password.message}
                  </p>
                )}
              </div>
            </div>

            {/* Reserve space for form-level error slot to prevent CLS */}
            <div className="min-h-[1.25rem]">
              {errors.root && (
                <p role="alert" className="text-destructive text-sm">
                  {errors.root.message}
                </p>
              )}
            </div>

            <Button
              type="submit"
              className="h-12 w-full gap-2 text-base"
              disabled={isSubmitting}
            >
              {isSubmitting ? (
                <>
                  <LoaderCircle className="size-5 animate-spin" aria-hidden />
                  Iniciando sesión…
                </>
              ) : (
                <>
                  <LogIn className="size-5" aria-hidden />
                  Iniciar sesión
                </>
              )}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
