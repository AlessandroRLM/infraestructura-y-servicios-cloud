/**
 * Parameterized PDF document shell consumed by the renderReportPdf seam.
 * Uses only @react-pdf/renderer primitives — no HTML/DOM components.
 * All 4 reports use this same shell; per-report variation lives in ReportPdfModel.
 *
 * Visual style mirrors the app: accent = --primary (oklch(0.205 0 0) ≈ #171717),
 * on-accent text = --primary-foreground (≈ #fafafa). The header badge reuses the
 * login logo (lucide GraduationCap) so printed reports match the product identity.
 */
import {
  Document,
  Page,
  Path,
  StyleSheet,
  Svg,
  Text,
  View,
} from "@react-pdf/renderer";
import { formatGeneratedAt } from "./formatGeneratedAt";
import type { ReportPdfModel } from "./model";

// App theme tokens resolved to concrete colors (@react-pdf cannot read CSS vars).
const ACCENT = "#171717"; // --primary
const ACCENT_FG = "#fafafa"; // --primary-foreground
const MUTED = "#71717a";
const BORDER = "#e4e4e7";
const ZEBRA = "#f4f4f5";

const styles = StyleSheet.create({
  page: {
    fontFamily: "Helvetica",
    fontSize: 9,
    paddingTop: 32,
    paddingBottom: 44,
    paddingHorizontal: 32,
    color: "#18181b",
    backgroundColor: "#ffffff",
  },
  header: {
    flexDirection: "row",
    alignItems: "center",
    marginBottom: 6,
  },
  badge: {
    width: 26,
    height: 26,
    borderRadius: 6,
    backgroundColor: ACCENT,
    alignItems: "center",
    justifyContent: "center",
    marginRight: 8,
  },
  kicker: {
    fontFamily: "Helvetica-Bold",
    fontSize: 7,
    letterSpacing: 1.5,
    color: MUTED,
  },
  title: {
    fontFamily: "Helvetica-Bold",
    fontSize: 15,
    color: "#18181b",
  },
  accentRule: {
    height: 2,
    backgroundColor: ACCENT,
    marginBottom: 8,
  },
  filterLine: {
    fontSize: 10,
    color: "#3f3f46",
    marginBottom: 1,
  },
  generatedAt: {
    fontSize: 8,
    color: MUTED,
    marginBottom: 10,
  },
  tableHeader: {
    flexDirection: "row",
    backgroundColor: ACCENT,
  },
  tableHeaderCell: {
    fontFamily: "Helvetica-Bold",
    fontSize: 8,
    color: ACCENT_FG,
    paddingVertical: 5,
    paddingHorizontal: 4,
  },
  tableRow: {
    flexDirection: "row",
    borderBottomColor: BORDER,
    borderBottomWidth: 1,
  },
  cell: {
    fontSize: 8,
    paddingVertical: 4,
    paddingHorizontal: 4,
  },
  footer: {
    position: "absolute",
    bottom: 22,
    left: 32,
    right: 32,
    flexDirection: "row",
    justifyContent: "space-between",
    fontSize: 8,
    color: MUTED,
    borderTopColor: BORDER,
    borderTopWidth: 1,
    paddingTop: 5,
  },
});

/** The login logo (lucide GraduationCap) drawn with @react-pdf SVG primitives. */
function LogoBadge() {
  return (
    <View style={styles.badge}>
      <Svg viewBox="0 0 24 24" width={15} height={15}>
        <Path
          d="M21.42 10.922a1 1 0 0 0-.019-1.838L12.83 5.18a2 2 0 0 0-1.66 0L2.6 9.08a1 1 0 0 0 0 1.832l8.57 3.908a2 2 0 0 0 1.66 0z"
          stroke={ACCENT_FG}
          strokeWidth={2}
          strokeLinecap="round"
          strokeLinejoin="round"
          fill="none"
        />
        <Path
          d="M22 10v6"
          stroke={ACCENT_FG}
          strokeWidth={2}
          strokeLinecap="round"
          strokeLinejoin="round"
          fill="none"
        />
        <Path
          d="M6 12.5V16a6 3 0 0 0 12 0v-3.5"
          stroke={ACCENT_FG}
          strokeWidth={2}
          strokeLinecap="round"
          strokeLinejoin="round"
          fill="none"
        />
      </Svg>
    </View>
  );
}

interface ReportPdfDocumentProps {
  model: ReportPdfModel;
}

export function ReportPdfDocument({ model }: ReportPdfDocumentProps) {
  const lastCol = model.columns.length - 1;
  return (
    <Document>
      <Page size="A4" style={styles.page}>
        {/* Header: login logo + kicker + title */}
        <View style={styles.header}>
          <LogoBadge />
          <View>
            <Text style={styles.kicker}>SISTEMA ACADÉMICO</Text>
            <Text style={styles.title}>{model.title}</Text>
          </View>
        </View>
        <View style={styles.accentRule} />

        {model.appliedFilter ? (
          <Text style={styles.filterLine}>{model.appliedFilter}</Text>
        ) : null}
        <Text style={styles.generatedAt}>
          Generado el {formatGeneratedAt(model.generatedAt)}
        </Text>

        {/* Table */}
        <View>
          {/* Column headers */}
          <View style={styles.tableHeader}>
            {model.columns.map((col, i) => (
              <Text
                key={col.key}
                style={[
                  styles.tableHeaderCell,
                  {
                    width: `${col.width}%`,
                    textAlign: col.align ?? "left",
                    borderRightColor: ACCENT_FG,
                    borderRightWidth: i < lastCol ? 0.5 : 0,
                  },
                ]}
              >
                {col.label}
              </Text>
            ))}
          </View>

          {/* Data rows */}
          {model.rows.map((row, rowIdx) => (
            <View
              key={rowIdx}
              style={[
                styles.tableRow,
                rowIdx % 2 === 1 ? { backgroundColor: ZEBRA } : {},
              ]}
            >
              {row.map((cell, cellIdx) => {
                const col = model.columns[cellIdx];
                return (
                  <Text
                    key={cellIdx}
                    style={[
                      styles.cell,
                      {
                        width: `${col?.width ?? 10}%`,
                        textAlign: col?.align ?? "left",
                        borderRightColor: BORDER,
                        borderRightWidth: cellIdx < lastCol ? 0.5 : 0,
                      },
                    ]}
                  >
                    {cell}
                  </Text>
                );
              })}
            </View>
          ))}
        </View>

        {/* Footer: institutional line + page number */}
        <View style={styles.footer} fixed>
          <Text>{model.footer}</Text>
          <Text
            render={({ pageNumber, totalPages }) =>
              `${pageNumber} / ${totalPages}`
            }
          />
        </View>
      </Page>
    </Document>
  );
}
