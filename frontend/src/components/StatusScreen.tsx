import { Link } from "@tanstack/react-router";
import type { ReactNode } from "react";
import { Button } from "@/components/ui/button";

interface StatusScreenProps {
  /** Large status glyph (e.g. "404", "403"). Omitted for non-HTTP states. */
  code?: string;
  title: string;
  description: string;
  /** Call to action; defaults to a "Volver al inicio" link. */
  action?: ReactNode;
}

/**
 * Shared full-height screen for terminal states — route not-found (404),
 * forbidden (403), and the root error boundary. The `70svh` floor centers it
 * whether it renders bare (root not-found, no shell) or inside the
 * authenticated `<main>`.
 */
export function StatusScreen({
  code,
  title,
  description,
  action,
}: StatusScreenProps) {
  return (
    <div className="flex min-h-[70svh] flex-col items-center justify-center gap-3 px-6 text-center">
      {code && (
        <p className="font-bold font-mono text-7xl text-muted-foreground/40">
          {code}
        </p>
      )}
      <h1 className="font-semibold text-2xl tracking-tight">{title}</h1>
      <p className="max-w-md text-muted-foreground">{description}</p>
      <div className="mt-2">
        {action ?? (
          <Button asChild>
            <Link to="/">Volver al inicio</Link>
          </Button>
        )}
      </div>
    </div>
  );
}
