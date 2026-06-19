import { AlertTriangle } from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";

interface TruncationBannerProps {
  count: number;
}

/**
 * Displays a prominent notice when a report dataset has been truncated.
 * The message is in neutral Spanish (tú register) per ux-rules.
 */
export function TruncationBanner({ count }: TruncationBannerProps) {
  return (
    <Alert
      variant="default"
      className="border-amber-400/60 bg-amber-50 dark:bg-amber-950/20"
    >
      <AlertTriangle className="text-amber-600" aria-hidden />
      <AlertTitle className="text-amber-800 dark:text-amber-300">
        Reporte truncado
      </AlertTitle>
      <AlertDescription className="text-amber-700 dark:text-amber-400">
        {`Este reporte muestra solo las primeras ${count} filas.`}
      </AlertDescription>
    </Alert>
  );
}
