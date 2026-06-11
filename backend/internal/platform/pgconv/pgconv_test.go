package pgconv_test

import (
	"math/big"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/pgconv"
)

// makeNumeric parses a decimal string into a pgtype.Numeric using the standard
// pgtype.Numeric.Scan path (same as PostgreSQL driver round-trip).
func makeNumeric(t *testing.T, s string) pgtype.Numeric {
	t.Helper()
	var n pgtype.Numeric
	if err := n.Scan(s); err != nil {
		t.Fatalf("makeNumeric(%q): %v", s, err)
	}
	return n
}

func TestNumericToString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input pgtype.Numeric
		want  string
	}{
		// ---- Valid=false (NULL from database) ----
		{
			name:  "invalid (NULL) returns empty string",
			input: pgtype.Numeric{Valid: false},
			want:  "",
		},
		// ---- Int==nil paths (zero-valued coefficient) ----
		{
			name:  "nil Int with Exp 0 returns 0",
			input: pgtype.Numeric{Valid: true, Int: nil, Exp: 0},
			want:  "0",
		},
		{
			name:  "nil Int with positive Exp returns 0",
			input: pgtype.Numeric{Valid: true, Int: nil, Exp: 2},
			want:  "0",
		},
		{
			name:  "nil Int with Exp -1 returns 0.0",
			input: pgtype.Numeric{Valid: true, Int: nil, Exp: -1},
			want:  "0.0",
		},
		{
			name:  "nil Int with Exp -3 returns 0.000",
			input: pgtype.Numeric{Valid: true, Int: nil, Exp: -3},
			want:  "0.000",
		},
		// ---- Exp == 0 (plain integer) ----
		{
			name:  "integer 7 no decimal point",
			input: makeNumeric(t, "7"),
			want:  "7",
		},
		{
			name:  "negative integer -42",
			input: pgtype.Numeric{Valid: true, Int: big.NewInt(-42), Exp: 0},
			want:  "-42",
		},
		// ---- Exp > 0 (scale-up: multiply by 10^Exp) ----
		{
			name:  "coefficient 5 Exp 2 produces 500",
			input: pgtype.Numeric{Valid: true, Int: big.NewInt(5), Exp: 2},
			want:  "500",
		},
		// ---- Exp < 0 (fixed-point decimal) ----
		{
			name:  "grade 5.5 NUMERIC(3,1) preserves one decimal",
			input: makeNumeric(t, "5.5"),
			want:  "5.5",
		},
		{
			name:  "grade 5.0 NUMERIC(3,1) preserves trailing zero",
			input: makeNumeric(t, "5.0"),
			want:  "5.0",
		},
		{
			name:  "grade 3.9 NUMERIC(3,1)",
			input: makeNumeric(t, "3.9"),
			want:  "3.9",
		},
		{
			name:  "grade 4.0 NUMERIC(3,1)",
			input: makeNumeric(t, "4.0"),
			want:  "4.0",
		},
		{
			name:  "weight 0.300 NUMERIC(4,3) preserves three decimals",
			input: makeNumeric(t, "0.300"),
			want:  "0.300",
		},
		{
			name:  "weight 0.40 NUMERIC(4,2) preserves two decimals",
			input: makeNumeric(t, "0.40"),
			want:  "0.40",
		},
		{
			name:  "negative fixed-point -1.5",
			input: pgtype.Numeric{Valid: true, Int: big.NewInt(-15), Exp: -1},
			want:  "-1.5",
		},
		{
			name:  "fractional part left-padded with zeros 0.05",
			input: pgtype.Numeric{Valid: true, Int: big.NewInt(5), Exp: -2},
			want:  "0.05",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := pgconv.NumericToString(tt.input)
			if got != tt.want {
				t.Errorf("NumericToString(...) = %q, want %q", got, tt.want)
			}
		})
	}
}
