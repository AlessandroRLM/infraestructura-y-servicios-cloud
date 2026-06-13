import { Code, ConnectError } from "@connectrpc/connect";
import { zodResolver } from "@hookform/resolvers/zod";
import { useNavigate } from "@tanstack/react-router";
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
  } = useForm<LoginValues>({ resolver: zodResolver(loginSchema) });

  const onSubmit = handleSubmit(async (values) => {
    clearErrors("root");
    try {
      await login.mutateAsync(values);
      // Use href so search params and hash in redirectTo are preserved correctly.
      await navigate({ href: redirectTo });
    } catch (err) {
      if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
        // Domain error about user input — show inline, never reveal which field.
        setError("root", { message: "Email or password is incorrect" });
        return;
      }
      // Transport/infra failure — transient, non-blocking.
      toast.error("Couldn't connect. Please try again.");
    }
  });

  return (
    <Card data-testid="login-page" className="mx-auto mt-24 w-full max-w-sm">
      <CardHeader>
        <CardTitle>Sign in</CardTitle>
        <CardDescription>Enter your credentials to continue</CardDescription>
      </CardHeader>
      <CardContent>
        <form noValidate onSubmit={onSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="email">Email</Label>
            <Input
              id="email"
              type="email"
              autoComplete="email"
              {...register("email")}
            />
            {/* Reserve space so showing/clearing the error does not shift layout */}
            <div className="min-h-[1.25rem]">
              {errors.email && (
                <p role="alert" className="text-destructive text-sm">
                  {errors.email.message}
                </p>
              )}
            </div>
          </div>
          <div className="space-y-2">
            <Label htmlFor="password">Password</Label>
            <Input
              id="password"
              type="password"
              autoComplete="current-password"
              {...register("password")}
            />
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
          <Button type="submit" className="w-full" disabled={isSubmitting}>
            {isSubmitting ? "Signing in…" : "Sign in"}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
