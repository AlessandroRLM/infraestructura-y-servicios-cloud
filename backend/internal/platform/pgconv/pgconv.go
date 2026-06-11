// Package pgconv provides pure conversion helpers between pgtype values and
// standard Go / wire-format types.
package pgconv

import (
	"fmt"
	"math/big"

	"github.com/jackc/pgx/v5/pgtype"
)

// NumericToString converts a pgtype.Numeric to its exact decimal string
// representation without rounding.
//
// Rules:
//   - Invalid (NULL) → ""
//   - nil coefficient with Exp ≥ 0 → "0"
//   - nil coefficient with Exp < 0 → "0.000…" (|Exp| decimal digits)
//   - Exp == 0 → integer string, e.g. "7"
//   - Exp > 0  → integer scaled up by 10^Exp, e.g. "500"
//   - Exp < 0  → fixed-point with |Exp| digits after the decimal point,
//     e.g. "5.50", "-1.5", "0.05"
func NumericToString(n pgtype.Numeric) string {
	if !n.Valid {
		return ""
	}

	if n.Int == nil {
		if n.Exp >= 0 {
			return "0"
		}
		scale := int(-n.Exp)
		return "0." + repeatZero(scale)
	}

	coeff := new(big.Int).Set(n.Int)

	switch {
	case n.Exp == 0:
		return coeff.String()

	case n.Exp > 0:
		mul := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n.Exp)), nil)
		coeff.Mul(coeff, mul)
		return coeff.String()

	default: // n.Exp < 0
		scale := int(-n.Exp)
		ten := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil)

		neg := coeff.Sign() < 0
		abs := new(big.Int).Abs(coeff)

		intPart := new(big.Int).Quo(abs, ten)
		fracPart := new(big.Int).Mod(abs, ten)

		// Left-pad the fractional part with zeros to exactly scale digits.
		fracStr := fmt.Sprintf("%0*s", scale, fracPart.String())

		sign := ""
		if neg {
			sign = "-"
		}
		return fmt.Sprintf("%s%s.%s", sign, intPart.String(), fracStr)
	}
}

// repeatZero returns a string of n ASCII '0' characters.
func repeatZero(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = '0'
	}
	return string(b)
}
