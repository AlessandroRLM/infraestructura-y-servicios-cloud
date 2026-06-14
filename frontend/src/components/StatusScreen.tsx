import { Link } from "@tanstack/react-router";
import { Button } from "@/components/ui/button";

interface StatusScreenProps {
  code: string;
  title: string;
  description: string;
}

// Shared full-height status page for terminal route states (404, 403). The
// 70svh floor centers it whether it renders bare (root not-found, no shell) or
// inside the authenticated <main>.
export function StatusScreen({ code, title, description }: StatusScreenProps) {
  return (
    <div className="flex min-h-[70svh] flex-col items-center justify-center gap-3 px-6 text-center">
      <p className="font-bold font-mono text-7xl text-muted-foreground/40">
        {code}
      </p>
      <h1 className="font-semibold text-2xl tracking-tight">{title}</h1>
      <p className="max-w-md text-muted-foreground">{description}</p>
      <Button asChild className="mt-2">
        <Link to="/">Volver al inicio</Link>
      </Button>
    </div>
  );
}
