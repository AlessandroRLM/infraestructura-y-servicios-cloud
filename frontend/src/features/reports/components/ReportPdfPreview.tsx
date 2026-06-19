import { Download } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/core/utils/cn";
import { formatGeneratedAt } from "../pdf/formatGeneratedAt";

interface ReportPdfPreviewProps {
  url: string | null;
  isRendering: boolean;
  fileName?: string;
  /** ISO 8601 timestamp shown above the iframe (RF-5.1 / AC-5.a). */
  generatedAt?: string;
}

/**
 * PDF preview + download control.
 *
 * - Renders an iframe with the PDF blob URL for in-browser preview.
 *   Uses `src="about:blank"` when url is null to avoid the same-document reload
 *   that `src=""` triggers in browsers.
 * - Provides a download anchor that reuses the same url (no second createObjectURL).
 * - Download is visually inert (aria-disabled) until the blob is ready.
 * - Shows a human-readable "Generado el…" timestamp above the iframe when provided.
 *
 * This component does NOT own the blob lifecycle — that is usePdfBlob's
 * responsibility. It only consumes the url/isRendering/generatedAt it receives.
 */
export function ReportPdfPreview({
  url,
  isRendering,
  fileName = "reporte.pdf",
  generatedAt,
}: ReportPdfPreviewProps) {
  const isReady = url !== null && !isRendering;

  return (
    <div className="flex flex-col gap-3">
      {/* Download control */}
      <div className="flex items-center justify-end">
        <Button
          variant="outline"
          size="sm"
          asChild={isReady}
          aria-disabled={!isReady}
          className={cn(!isReady && "pointer-events-none opacity-50")}
        >
          {isReady ? (
            <a href={url} download={fileName} aria-disabled={!isReady}>
              <Download data-icon="inline-start" aria-hidden />
              Descargar PDF
            </a>
          ) : (
            <span>
              <Download data-icon="inline-start" aria-hidden />
              Descargar PDF
            </span>
          )}
        </Button>
      </div>

      {/* On-screen generatedAt label — RF-5.1 / AC-5.a */}
      {generatedAt ? (
        <p className="text-xs text-muted-foreground">
          Generado el {formatGeneratedAt(generatedAt)}
        </p>
      ) : null}

      {/* PDF iframe preview — intentionally NOT sandboxed.
          The content is our own client-generated binary PDF served from a same-origin
          blob: URL — there is no untrusted markup or script to contain, so a sandbox
          adds no meaningful security here. Empirically (verified 2026-06-18), ANY
          sandbox value breaks Chromium's built-in PDF viewer: sandbox="allow-same-origin"
          leaves the document blank, and even adding allow-scripts does not restore it
          (and that combination would void the sandbox per MDN anyway). Removing the
          attribute is the correct, honest choice for trusted same-origin PDF content.
          Do NOT add a sandbox attribute here without re-verifying the PDF still renders. */}
      <iframe
        title="Vista previa del reporte"
        src={url ?? "about:blank"}
        className={cn(
          "w-full rounded-md border bg-white",
          "h-[600px]",
          !url && "opacity-0",
        )}
        aria-hidden={!url}
      />
    </div>
  );
}
