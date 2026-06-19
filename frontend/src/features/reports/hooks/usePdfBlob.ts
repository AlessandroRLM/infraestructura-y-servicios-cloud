/**
 * usePdfBlob — single owner of PDF object URLs for a report view.
 *
 * Responsibility:
 *  - Render a ReportPdfModel to a Blob via the renderReportPdf seam.
 *  - Create an object URL for the rendered Blob (URL.createObjectURL).
 *  - Revoke the prior object URL when the model changes (no two live URLs).
 *  - Revoke the current object URL on unmount (no leaks).
 *  - Expose isRendering / error for UX state boundary.
 *
 * The download button reuses the returned url — no second createObjectURL call.
 */
import { useEffect, useRef, useState } from "react";
import type { ReportPdfModel } from "../pdf/model";
import { renderReportPdf } from "../pdf/renderPdf";

export interface UsePdfBlobResult {
  url: string | null;
  blob: Blob | null;
  isRendering: boolean;
  error: Error | null;
}

export function usePdfBlob(model: ReportPdfModel | null): UsePdfBlobResult {
  const [url, setUrl] = useState<string | null>(null);
  const [blob, setBlob] = useState<Blob | null>(null);
  const [isRendering, setIsRendering] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  // Track the current object URL so we can revoke it on the next render or unmount.
  const currentUrlRef = useRef<string | null>(null);

  useEffect(() => {
    if (model === null) {
      // No model — revoke any prior URL and reset state.
      if (currentUrlRef.current !== null) {
        URL.revokeObjectURL(currentUrlRef.current);
        currentUrlRef.current = null;
      }
      setUrl(null);
      setBlob(null);
      setIsRendering(false);
      setError(null);
      return;
    }

    let cancelled = false;

    setIsRendering(true);
    setError(null);

    renderReportPdf(model)
      .then((renderedBlob) => {
        if (cancelled) return;

        // Revoke the prior URL before creating the new one.
        if (currentUrlRef.current !== null) {
          URL.revokeObjectURL(currentUrlRef.current);
        }

        const newUrl = URL.createObjectURL(renderedBlob);
        currentUrlRef.current = newUrl;

        setBlob(renderedBlob);
        setUrl(newUrl);
        setIsRendering(false);
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        setIsRendering(false);
        setError(err instanceof Error ? err : new Error(String(err)));
      });

    // Cleanup: cancel the in-flight render and revoke the URL on unmount
    // or when the model changes (next effect fires after this cleanup).
    return () => {
      cancelled = true;
      if (currentUrlRef.current !== null) {
        URL.revokeObjectURL(currentUrlRef.current);
        currentUrlRef.current = null;
      }
      setUrl(null);
      setBlob(null);
      setIsRendering(false);
    };
  }, [model]);

  return { url, blob, isRendering, error };
}
