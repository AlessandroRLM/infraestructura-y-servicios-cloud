/**
 * Pure weight conversion utilities for evaluation schemes.
 *
 * The editor model is INTEGER PERCENTS (1..100). These helpers convert
 * between that model and the wire format expected by the grades backend
 * (3-decimal-place strings summing to exactly 1.000).
 *
 * Float-safety proof: every percent value pᵢ is a JS integer (1..100),
 * enforced by the zod `.int()` constraint in evaluationScheme.ts before
 * any call reaches this module. `.toFixed(3)` produces the correct
 * 3-decimal string for all integers 1..100 (verified exhaustively); the
 * zod `.int()` constraint ensures only integers reach this function.
 * The backend parses each string as exact big.Rat, so Σ(pᵢ/100) =
 * (Σpᵢ)/100 = 100/100 = 1 exactly, provided the integer sum gate
 * (sumPercents === 100) is enforced by the caller (zod schema).
 */

/**
 * Converts an integer percent value (1..100) to a 3-decimal weight string.
 * The input MUST be an integer; callers are expected to enforce this via zod.
 *
 * @param p - Integer percent, e.g. 30
 * @returns 3-decimal string, e.g. "0.300"
 *
 * @example
 * percentToWeight(30) // "0.300"
 * percentToWeight(100) // "1.000"
 * percentToWeight(5)  // "0.050"
 */
export function percentToWeight(p: number): string {
  return (p / 100).toFixed(3);
}

/**
 * Converts a 3-decimal weight string to an integer percent for display.
 * This is a display-only helper — the result MUST NOT be round-tripped
 * into a submit without going through the integer-percent editor model.
 *
 * @param w - Decimal weight string, e.g. "0.400"
 * @returns Integer percent, e.g. 40
 *
 * @example
 * weightToPercent("0.400") // 40
 * weightToPercent("1.000") // 100
 * weightToPercent("0.050") // 5
 */
export function weightToPercent(w: string): number {
  return Math.round(parseFloat(w) * 100);
}

/**
 * Sums the percent values of a row array. Used for the live running total
 * and for the submit gate (`sumPercents(rows) === 100`).
 *
 * @param rows - Array of objects with an integer `percent` field
 * @returns Integer sum of all percent values
 *
 * @example
 * sumPercents([{ percent: 30 }, { percent: 30 }, { percent: 40 }]) // 100
 */
export function sumPercents(rows: { percent: number }[]): number {
  return rows.reduce((acc, row) => acc + row.percent, 0);
}
