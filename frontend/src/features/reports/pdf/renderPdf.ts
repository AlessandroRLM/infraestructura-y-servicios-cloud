/**
 * PDF rendering seam — the ONLY file that imports @react-pdf/renderer directly.
 * All views and hooks depend on this function signature, never on the library.
 * Swap the implementation here if the primary library must change (e.g. pdfmake).
 */

import type { DocumentProps } from "@react-pdf/renderer";
import { pdf } from "@react-pdf/renderer";
import { createElement, type ReactElement } from "react";
import { ReportPdfDocument } from "./ReportPdfDocument";

export type { ReportPdfModel } from "./model";

/**
 * Renders a ReportPdfModel to a PDF Blob using @react-pdf/renderer.
 * This is the only allowed call-site for pdf(...).toBlob().
 *
 * The cast to DocumentProps is safe: ReportPdfDocument renders <Document> as
 * its root — the pdf() function requires an element that ultimately renders a
 * Document tree, which it does. The type system is narrower than the runtime.
 */
export async function renderReportPdf(
  model: import("./model").ReportPdfModel,
): Promise<Blob> {
  const element = createElement(ReportPdfDocument, {
    model,
  }) as unknown as ReactElement<DocumentProps>;
  const instance = pdf(element);
  return instance.toBlob();
}
